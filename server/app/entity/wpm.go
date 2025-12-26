package entity

type WPM struct {
	OriginalSnippet string
	UserSnippet     string
	DurationMS      int64
}
type WPMRes struct {
	ID       uint32  `json:"id"`
	RAW      float32 `json:"raw"`
	WPM      float32 `json:"wpm"`
	Accuracy float32 `json:"accuracy"`
	Correct  int32   `correct:"correct"`
	Wrong    int32   `json:"wrong"`
}
