package main

import (
	"context"
	"database/sql"
	"osdtype/application/api"
	"osdtype/database"

	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	log, _ := zap.NewProduction()
	conn, err := sql.Open("postgresql", "ipostgres://user:123@localhost:5432/typedata?sslmode=disable")
	if err != nil {
		log.Error("Could not connect to the database")
		return
	}
	defer conn.Close()
	log.Sync()
	query := database.New(conn)
	api.StartServer(ctx, log, query)
}
