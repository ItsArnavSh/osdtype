package main

import (
	"context"
	"osdtype/application/api"
	"osdtype/application/entity"
	"osdtype/database"

	"github.com/jackc/pgx/v5"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	log, _ := zap.NewProduction()
	conn, err := pgx.Connect(ctx, "postgres://user:123@localhost:5432/typedata?sslmode=disable")
	if err != nil {
		log.Error("Could not connect to the database")
		return
	}
	defer conn.Close(ctx)
	log.Sync()
	query := database.New(conn)

	essen := entity.Essentials{Db: query, Logger: log}

	server, _ := api.NewServer(ctx, essen)
	server.SetRoutes()
	server.StartServer(ctx, log, query)
}
