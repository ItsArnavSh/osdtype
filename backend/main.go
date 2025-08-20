package main

import (
	"context"
	"fmt"
	"osdtype/application/api"
	"osdtype/application/entity"
	"osdtype/database"

	"github.com/jackc/pgx/v5"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	log, err := zap.NewProduction()
	defer log.Sync()
	if err != nil {
		fmt.Print("Could not set up logger")
		return
	}
	conn, err := pgx.Connect(ctx, "postgres://user:123@localhost:5432/typedata?sslmode=disable")
	if err != nil {
		log.Error("Could not connect to the database")
		return
	}
	query := database.New(conn)

	essen := entity.Essentials{Db: query, Logger: log}

	server, _ := api.NewServer(ctx, essen)
	server.SetRoutes()
	server.StartServer(ctx)
}
