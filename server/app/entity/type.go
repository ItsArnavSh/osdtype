package entity

// Languages is the canonical order used by the browser WASM generator and by
// the server. Keeping one list is required for seed -> language parity.
var Languages = []string{
	"c",
	"cpp",
	"go",
	"haskell",
	"html",
	"java",
	"js",
	"python",
	"rust",
	"sql",
}

func LanguageForSeed(seed uint64) string {
	return Languages[seed%uint64(len(Languages))]
}
