package livetype

import (
	"context"
	"osdtype/application/entity"
	"osdtype/database"

	"go.uber.org/zap"
)

type Typer struct {
	info    entity.TypeInfo
	query   *database.Queries
	logger  zap.Logger
	keyChan chan string
}

func (t *Typer) StartTyping(ctx context.Context, Info entity.TypeInfo, query *database.Queries) error {
	snippet, err := query.GetRandomSnippetByLanguage(ctx, Info.Lang.String())
	if err != nil {
		t.logger.Error("Could not load snippet")
		return err
	}

	return nil
}
