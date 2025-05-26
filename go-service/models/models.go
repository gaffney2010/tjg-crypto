package models

type Game struct {
	Date  int32  `db:"date"`
	TeamA string `db:"team_a"`
	TeamB string `db:"team_b"`
}

type Mapping struct {
	Namespace string `db:"namespace"`
	Secondary string `db:"secondary"`
	Primary   string `db:"primry"`
}
