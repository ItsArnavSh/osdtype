package entity

import (
	"reflect"
	"testing"
)

func TestSharesRunModeAndPrivacyCombinations(t *testing.T) {
	public := User{
		ShareRuns:        RunSharingAll,
		ShareRanked:      true,
		ShareRooms:       true,
		SharePractice:    true,
		ShareContests:    true,
		HidePrivateRooms: true,
	}
	tests := []struct {
		name      string
		user      User
		mode      GameMode
		pinned    bool
		roomPriv  bool
		wantShare bool
	}{
		{name: "all ranked", user: public, mode: GameRanked, wantShare: true},
		{name: "all practice", user: public, mode: GamePractice, wantShare: true},
		{name: "all contest", user: public, mode: GameContest, wantShare: true},
		{name: "all public room", user: public, mode: GameRoom, wantShare: true},
		{name: "all private room hidden", user: public, mode: GameRoom, roomPriv: true, wantShare: false},
		{
			name:      "all private room allowed",
			user:      withPrivateRooms(public, false),
			mode:      GameRoom,
			roomPriv:  true,
			wantShare: true,
		},
		{
			name:      "pinned rejects unpinned",
			user:      withSharing(public, RunSharingPinned),
			mode:      GameRanked,
			wantShare: false,
		},
		{
			name:      "pinned accepts pinned",
			user:      withSharing(public, RunSharingPinned),
			mode:      GameRanked,
			pinned:    true,
			wantShare: true,
		},
		{
			name:      "none rejects even pinned",
			user:      withSharing(public, RunSharingNone),
			mode:      GameRanked,
			pinned:    true,
			wantShare: false,
		},
		{
			name:      "legacy empty sharing is all",
			user:      withSharing(public, ""),
			mode:      GameRanked,
			wantShare: true,
		},
		{
			name:      "ranked tick off",
			user:      withMode(public, GameRanked, false),
			mode:      GameRanked,
			wantShare: false,
		},
		{
			name:      "rooms tick off",
			user:      withMode(public, GameRoom, false),
			mode:      GameRoom,
			wantShare: false,
		},
		{
			name:      "practice tick off",
			user:      withMode(public, GamePractice, false),
			mode:      GamePractice,
			wantShare: false,
		},
		{
			name:      "contests tick off",
			user:      withMode(public, GameContest, false),
			mode:      GameContest,
			wantShare: false,
		},
		{name: "unknown mode", user: public, mode: GameMode("unknown"), wantShare: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.user.SharesRun(tc.mode, tc.pinned, tc.roomPriv); got != tc.wantShare {
				t.Fatalf("SharesRun(%q, pinned=%v, private=%v) = %v, want %v",
					tc.mode, tc.pinned, tc.roomPriv, got, tc.wantShare)
			}
		})
	}
}

func TestSharedModes(t *testing.T) {
	user := User{ShareRanked: true, ShareRooms: false, SharePractice: true, ShareContests: false}
	want := []GameMode{GameRanked, GamePractice}
	if got := user.SharedModes(); !reflect.DeepEqual(got, want) {
		t.Fatalf("SharedModes() = %v, want %v", got, want)
	}
}

func withSharing(user User, sharing string) User {
	user.ShareRuns = sharing
	return user
}

func withPrivateRooms(user User, hide bool) User {
	user.HidePrivateRooms = hide
	return user
}

func withMode(user User, mode GameMode, enabled bool) User {
	switch mode {
	case GameRanked:
		user.ShareRanked = enabled
	case GameRoom:
		user.ShareRooms = enabled
	case GamePractice:
		user.SharePractice = enabled
	case GameContest:
		user.ShareContests = enabled
	}
	return user
}
