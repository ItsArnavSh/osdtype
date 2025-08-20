package langauge

import (
	"context"
	"encoding/json"
	"osdtype/application/entity"
	"osdtype/database"

	"github.com/google/uuid"
)

func InsertSnippet(ctx context.Context, db database.Queries, language, snippet string) error {
	id := uuid.NewString()

	tokens := Tokenize(snippet)
	json_bytes, err := json.Marshal(tokens)
	if err != nil {
		return err
	}
	encoded_snippet := string(json_bytes)
	return db.UpsertLanguageSnippet(ctx, database.UpsertLanguageSnippetParams{
		Language: language,
		Snippet:  encoded_snippet,
		ID:       id,
	})
}
func GetSnippet(ctx context.Context, essen entity.Essentials, lang string) (database.LanguageStore, error) {
	return essen.Db.GetRandomSnippetByLanguage(ctx, lang)

}
