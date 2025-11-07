package services

import (
	"osdtyp/app/internal/postgresql"
	"osdtyp/app/utils"

	"go.uber.org/zap"
)

type ServiceLayer struct {
	db      postgresql.Database
	logger  *zap.SugaredLogger
	int_gen utils.Generator
}

func NewServiceLayer(logger *zap.SugaredLogger) (ServiceLayer, error) {
	db, err := postgresql.ConnectDatabase(logger)
	if err != nil {
		return ServiceLayer{}, err
	}
	gen := utils.NewGenerator()
	return ServiceLayer{logger: logger, db: db, int_gen: gen}, nil
}
