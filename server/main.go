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
	// Sync flushes buffered log entries. It fails on some platforms when
	// stdout is a terminal, which is not worth treating as fatal.
	defer func() { _ = logger.Sync() }()
	server, err := api.NewServer(logger)
	if err != nil {
		logger.Errorf("could not build the server: %v", err)
		return
	}
	server.SetupRoutes()
	server.StartServer()
}
