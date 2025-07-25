package main

import (
	"context"
	"osdtype/application/api"
	"osdtype/database"

	"github.com/jackc/pgx/v5"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	log, _ := zap.NewProduction()
	conn, err := pgx.Connect(ctx, "ipostgres://user:123@localhost:5432/typedata?sslmode=disable")
	if err != nil {
		log.Error("Could not connect to the database")
		return
	}
	defer conn.Close(ctx)
	log.Sync()
	query := database.New(conn)
	api.StartServer(ctx, log, query)
}
