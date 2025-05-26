package server

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"sync"

	"go-service/db"
	"go-service/models"
	pb "go-service/proto"
)

type CryptoServer struct {
	pb.UnimplementedCryptoServiceServer
}

// Everything other than MapTeams should really only be called one-at-a-time.  We provide a lock to simplify logic.
var lock sync.RWMutex

func (s *CryptoServer) PrimarySeason(ctx context.Context, req *pb.PrimarySeasonReq) (*pb.PrimarySeasonResp, error) {
	lock.Lock()
	defer lock.Unlock()

	for _, game := range req.Games {
		// First check if the game already exists
		var exists bool
		query := `
			SELECT EXISTS (
				SELECT 1
				FROM primary_games
				WHERE namespace = $1 AND date = $2 AND team_a = $3 AND team_b = $4

				UNION ALL

				SELECT 1
				FROM primary_games
				WHERE namespace = $1 AND date = $2 AND team_a = $4 AND team_b = $3
			)`
		err := db.DB.QueryRowContext(ctx, query, req.Namespace, game.Date, game.TeamA, game.TeamB).Scan(&exists)
		if err != nil {
			return &pb.PrimarySeasonResp{
				Status: "Read from database failed",
			}, err
		}
		if exists {
			continue
		}

		_, err = db.DB.NamedExec(`INSERT INTO primary_games (namespace, date, team_a, team_b) VALUES (:namespace, :date, :team_a, :team_b)`,
			map[string]interface{}{
				"namespace": req.Namespace,
				"date":      game.Date,
				"team_a":    game.TeamA,
				"team_b":    game.TeamB,
			})
		if err != nil {
			return &pb.PrimarySeasonResp{
				Status: "Write to database failed",
			}, err
		}
	}

	seenTeams := make(map[string]bool)

	for _, game := range req.Games {
		seenTeams[game.TeamA] = true
		seenTeams[game.TeamB] = true
	}

	for team := range seenTeams {
		_, err := db.DB.NamedExec(`INSERT INTO mapping (namespace, secondary, primry)
		VALUES (:namespace, :team, :team) ON CONFLICT DO NOTHING`,
			map[string]interface{}{
				"namespace": req.Namespace,
				"team":      team,
			})
		if err != nil {
			return &pb.PrimarySeasonResp{Status: "Write to mapping database failed"}, err
		}
	}

	return &pb.PrimarySeasonResp{Status: "ok"}, nil
}

func (s *CryptoServer) SecondarySeason(ctx context.Context, req *pb.SecondarySeasonReq) (*pb.SecondarySeasonResp, error) {
	lock.Lock()
	defer lock.Unlock()

	// Fetch existing mappings
	mappings := make(map[string]string)
	rows, err := db.DB.Queryx(`SELECT secondary, primry FROM mapping WHERE namespace = $1`, req.Namespace)
	if err != nil {
		return &pb.SecondarySeasonResp{Status: "Read from mapping database failed, type 1"}, err
	}
	defer rows.Close()
	for rows.Next() {
		var m models.Mapping
		if err := rows.StructScan(&m); err != nil {
			return &pb.SecondarySeasonResp{Status: "Read from mapping database failed, type 2"}, err
		}
		mappings[m.Secondary] = m.Primary
	}

	// Initial seed
	if req.Seed.Primary == nil {
		return &pb.SecondarySeasonResp{
			Status: "Invalid initial seed",
		}, errors.New("Invalid initial seed")
	}
	mappings[req.Seed.Secondary] = *req.Seed.Primary

	// We store new mappings to both `mappings` and `newMappings`, with the latter used to write to table
	newMappings := make(map[string]string)
	atLeastOneChanged := true
	for atLeastOneChanged {
		atLeastOneChanged = false
		for _, game := range req.Games {
			teamAMapped, teamAExists := mappings[game.TeamA]
			teamBMapped, teamBExists := mappings[game.TeamB]

			if teamAExists && !teamBExists {
				// Check if teamAMapped has a game on the same date in primary_games
				var opponent string
				err := db.DB.Get(&opponent, `SELECT team_b FROM primary_games WHERE namespace = $1 AND date = $2 AND team_a = $3
					UNION
					SELECT team_a FROM primary_games WHERE namespace = $1 AND date = $2 AND team_b = $3`,
					req.Namespace, game.Date, teamAMapped)
				if err == nil {
					atLeastOneChanged = true
					newMappings[game.TeamB] = opponent
					mappings[game.TeamB] = opponent
				} else if err != sql.ErrNoRows {
					return &pb.SecondarySeasonResp{
						Status: "Read from database failed",
					}, err
				}
			} else if teamBExists && !teamAExists {
				var opponent string
				err := db.DB.Get(&opponent, `SELECT team_b FROM primary_games WHERE namespace = $1 AND date = $2 AND team_a = $3
					UNION
					SELECT team_a FROM primary_games WHERE namespace = $1 AND date = $2 AND team_b = $3`,
					req.Namespace, game.Date, teamBMapped)
				if err == nil {
					atLeastOneChanged = true
					newMappings[game.TeamA] = opponent
					mappings[game.TeamA] = opponent
				} else if err != sql.ErrNoRows {
					return &pb.SecondarySeasonResp{
						Status: "Read from database failed",
					}, err
				}
			}
		}
	}

	// Write down which games didn't get mapped
	unmappedGames := make([]*pb.Game, 0)
	for _, game := range req.Games {
		_, teamAExists := mappings[game.TeamA]
		_, teamBExists := mappings[game.TeamB]
		if !teamAExists || !teamBExists {
			unmappedGames = append(unmappedGames, game)
		}
	}

	// Insert new mappings
	tx := db.DB.MustBegin()
	for secondary, primary := range newMappings {
		_, err := tx.Exec(`INSERT INTO mapping (namespace, secondary, primry) VALUES ($1, $2, $3)
            ON CONFLICT (namespace, secondary) DO UPDATE SET primry = EXCLUDED.primry`,
			req.Namespace, secondary, primary)
		if err != nil {
			tx.Rollback()
			return &pb.SecondarySeasonResp{Status: "Write to mapping database failed"}, err
		}
	}
	tx.Commit()

	return &pb.SecondarySeasonResp{
		Status:        "ok",
		UnmappedGames: unmappedGames,
	}, nil
}

func (s *CryptoServer) MapTeams(ctx context.Context, req *pb.MapTeamsReq) (*pb.MapTeamsResp, error) {
	// Allow multiple MapTeams calls at once
	lock.RLock()
	defer lock.RUnlock()

	result := make([]*pb.Mapping, 0)
	for _, team := range req.Teams {
		singleResult := &pb.Mapping{}
		singleResult.Secondary = team

		var mappedTeam string
		err := db.DB.Get(&mappedTeam, `SELECT primry FROM mapping WHERE namespace = $1 AND secondary = $2`, req.Namespace, team)
		if err == nil {
			singleResult.Primary = new(string)
			*singleResult.Primary = mappedTeam
		} else if err != sql.ErrNoRows {
			return &pb.MapTeamsResp{
				Status: "Read from mapping database failed",
			}, err
		}

		result = append(result, singleResult)
	}
	return &pb.MapTeamsResp{
		Status:  "ok",
		Mapping: result,
	}, nil
}

func (s *CryptoServer) DeleteNamespace(ctx context.Context, req *pb.DeleteNamespaceReq) (*pb.DeleteNamespaceResp, error) {
	lock.Lock()
	defer lock.Unlock()

	_, err := db.DB.Exec(`DELETE FROM mapping WHERE namespace = $1`, req.Namespace)
	if err != nil {
		return &pb.DeleteNamespaceResp{Status: "Failed to delete namespace"}, err
	}
	_, err = db.DB.Exec(`DELETE FROM primary_games WHERE namespace = $1`, req.Namespace)
	if err != nil {
		return &pb.DeleteNamespaceResp{Status: "Failed to delete namespace"}, err
	}
	return &pb.DeleteNamespaceResp{
		Status: "ok",
	}, nil
}
