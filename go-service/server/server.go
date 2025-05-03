package server

import (
	"context"
	"go-service/db"
	models "go-service/models"
	pb "go-service/proto"
)

type UserServer struct {
	pb.UnimplementedUserServiceServer
}

func (s *UserServer) CreateUser(ctx context.Context, req *pb.User) (*pb.User, error) {
	var id int
	err := db.DB.QueryRow(`INSERT INTO users(name) VALUES($1) RETURNING id`, req.Name).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &pb.User{Id: int32(id), Name: req.Name}, nil
}

func (s *UserServer) GetUser(ctx context.Context, req *pb.UserID) (*pb.User, error) {
	var user models.User
	err := db.DB.Get(&user, "SELECT id, name FROM users WHERE id=$1", req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.User{Id: int32(user.ID), Name: user.Name}, nil
}
