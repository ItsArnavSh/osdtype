package main

import (
	"osdtyp/app/api"
	"osdtyp/boot"
)

func main() {
	logger := boot.Initialize_App()
	if logger == nil {
		return
	}
	defer logger.Sync()
	server := api.NewServer(logger)
	server.SetupRoutes()
	server.StartServer()
}
