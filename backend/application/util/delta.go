package util

import "github.com/sergi/go-diff/diffmatchpatch"

// Todo: Add a struct and make it a once intialized class of sorts for dmp
func GenerateDeltas(original, userwritten string) string {
	//Shrink this to user text size

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(original, userwritten, false)
	return dmp.DiffToDelta(diffs)
}
func RegenerateUserText(original, diffs string) (string, error) {
	dmp := diffmatchpatch.New()
	restored_diffs, err := dmp.DiffFromDelta(original, diffs)
	if err != nil {
		return "", err
	}
	restored := dmp.DiffText2(restored_diffs)
	return restored, nil
}
