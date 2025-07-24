package livetype

import (
	"context"
	"osdtype/application/entity"
	"osdtype/database"

	"go.uber.org/zap"
)

type Typer struct {
	query   *database.Queries
	logger  zap.Logger
	keyChan chan entity.KeyDef
}

func (t *Typer) GetSnippet(ctx context.Context, Info entity.TypeInfo, query *database.Queries) (database.LanguageStore, error) {
	snippet, err := query.GetRandomSnippetByLanguage(ctx, Info.Lang.String())
	if err != nil {
		t.logger.Error("Could not load snippet")
		return database.LanguageStore{}, err
	}

	return snippet, nil
}

func (t *Typer) LiveSave(ctx context.Context, snippetid string) {
	recording := []byte{}

	var start, latest int64
	for keystroke := range t.keyChan {
		if start == 0 {
			start = keystroke.Time
			latest = start
		} else {
			diff := keystroke.Time - int64(latest)
			latest = keystroke.Time
			//Now diff has the time in milliseconds
			// We will round it to 20ms
			diff /= 20
			const bytesize = 256
			empty := diff / bytesize
			last := byte(diff % bytesize)
			//Add 0 empty times to the byte
			// Then add last to the array
			recording = append(recording, make([]byte, empty)...)
			recording = append(recording, last)
		}

	}
	//Now recording has a compressed recoding of each keystroke pressed.
}
