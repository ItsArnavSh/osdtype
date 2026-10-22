package entity

type WPM struct {
	OriginalSnippet string
	UserSnippet     string
	DurationMS      int64
}

type WPMRes struct {
	Name     string  `json:"name"`
	ID       uint32  `json:"id"`
	RAW      float64 `json:"raw"`
	WPM      float64 `json:"wpm"`
	Accuracy float64 `json:"accuracy"`
	Correct  int     `json:"correct"`
	Wrong    int     `json:"wrong"`
}
