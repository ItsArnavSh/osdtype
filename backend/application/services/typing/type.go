package livetype

import (
	"context"
	"osdtype/application/entity"
	"osdtype/database"
	"sync"

	"go.uber.org/zap"
)

type Typer struct {
	Query   *database.Queries
	Logger  zap.Logger
	KeyChan chan entity.KeyDef
	Rec     entity.Recording
}

func (t *Typer) GetSnippet(ctx context.Context, lang string, query *database.Queries) (database.LanguageStore, error) {
	snippet, err := query.GetRandomSnippetByLanguage(ctx, lang)
	if err != nil {
		t.Logger.Error("Could not load snippet")
		return database.LanguageStore{}, err
	}

	return snippet, nil
}

func (t *Typer) LiveSave(ctx context.Context, wg *sync.WaitGroup) entity.Recording {
	defer wg.Done()
	recording := []byte{}
	var timestamps []int64
	var start, latest int64
	for keystroke := range t.KeyChan {
		if start == 0 {
			start = keystroke.Time
			latest = start
		} else {
			diff := keystroke.Time - int64(latest)
			latest = keystroke.Time
			timestamps = append(timestamps, keystroke.Time)
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

	return entity.Recording{Recording: recording, Timestamps: timestamps}
}
