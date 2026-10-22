package anticheat

import (
	"osdtyp/app/entity"
	"testing"
)

func TestHumanRunNotFlagged(t *testing.T) {
	ts := []int32{0, 140, 275, 420, 590, 730, 880, 1010, 1160, 1310}
	report := Analyze(ts, nil, 400, 120000, false)
	if report.Flagged {
		t.Fatalf("human run flagged: %+v", report)
	}
}

func TestMachineRunFlagged(t *testing.T) {
	ts := make([]int32, 0, 200)
	for i := 0; i < 200; i++ {
		ts = append(ts, int32(i*10))
	}
	report := Analyze(ts, nil, 400, 10000, false)
	if !report.Flagged {
		t.Fatalf("machine run not flagged: %+v", report)
	}
}

func TestPasteFlagged(t *testing.T) {
	ts := []int32{0, 100, 200, 300, 400, 500, 600, 700}
	events := []entity.Keypress{{Action: entity.KEYPRESS, Value: "abcdefghij"}}
	report := Analyze(ts, events, 100, 60000, false)
	if !report.Flagged {
		t.Fatalf("paste not flagged: %+v", report)
	}
}

func TestBotsAndTinyRunsIgnored(t *testing.T) {
	if Analyze([]int32{0, 1, 2, 3, 4, 5, 6, 7}, nil, 50, 1000, true).Flagged {
		t.Fatal("bot flagged")
	}
	if Analyze([]int32{0, 1, 2}, nil, 3, 1000, false).Flagged {
		t.Fatal("tiny run flagged")
	}
}
