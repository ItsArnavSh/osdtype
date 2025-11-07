package entity

type WPM struct {
	OriginalSnippet string
	UserSnippet     string
	DurationMS      int64
}
type WPMRes struct {
	ID       uint64  `json:"id"`
	RAW      float64 `json:"raw"`
	WPM      float64 `json:"wpm"`
	Accuracy float64 `json:"accuracy"`
	Correct  int32   `correct:"correct"`
	Wrong    int32   `json:"wrong"`
}
