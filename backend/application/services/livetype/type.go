package livetype

import (
	"context"
	"osdtype/application/entity"
	"osdtype/database"

	"go.uber.org/zap"
)

type Typer struct {
	info   entity.TypeInfo
	query  *database.Queries
	logger zap.Logger
}

func (t *Typer) StartTyping(ctx context.Context, Info entity.TypeInfo, query *database.Queries) {
	snippet, err := query.GetRandomSnippetByLanguage(ctx, Info.Lang.String())
	if err != nil {

	}
}
