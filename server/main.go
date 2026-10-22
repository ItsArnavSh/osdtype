package main

import (
	"osdtyp/app/api"
	"osdtyp/boot"
)

func main() {
	logger, err := boot.Initialize_App()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	server, err := api.NewServer(logger)
	if err != nil {
		logger.Fatalw("failed to initialize server", "error", err)
	}
	defer server.Close()
	server.StartServer()
}
