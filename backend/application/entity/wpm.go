package entity

type WPM struct {
	OriginalSnippet string
	UserSnippet     string
	DurationMS      int64
}
type WPMRes struct {
	RAW      float64
	WPM      float64
	Accuracy float64
	Correct  int32
	Wrong    int32
}
