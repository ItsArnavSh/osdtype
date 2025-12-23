package services

import (
	"osdtyp/app/core"
	"osdtyp/app/internal/postgresql"
	"osdtyp/app/utils"

	"go.uber.org/zap"
)

type ServiceLayer struct {
	db      *postgresql.Database
	logger  *zap.SugaredLogger
	int_gen utils.Generator
	core    *core.CodeCore
}

func NewServiceLayer(logger *zap.SugaredLogger, core *core.CodeCore, db *postgresql.Database) (ServiceLayer, error) {

	gen := utils.NewGenerator()
	return ServiceLayer{logger: logger, db: db, int_gen: gen, core: core}, nil
}
