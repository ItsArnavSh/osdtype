//go:build integration

package postgresql

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"

	"osdtyp/app/entity"
)

// newDB connects to the throwaway test database and truncates the tables the
// suite touches, so each run starts from a known state.
func newDB(t *testing.T) Database {
	t.Helper()
	db := connect(t)

	// Delete in dependency order. The shared test database may hold rows from
	// an earlier run.
	ctx := context.Background()
	for _, model := range []any{
		&entity.Room_User{}, &entity.Room{}, &entity.Contest{},
		&entity.Task{}, &entity.Friends{}, &entity.User{},
	} {
		if err := db.db.WithContext(ctx).Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("could not clean %T: %v", model, err)
		}
	}
	return db
}

func TestConnectRunsMigrations(t *testing.T) {
	db := connect(t)

	for _, model := range []any{
		&entity.User{}, &entity.Room{}, &entity.Room_User{},
		&entity.Contest{}, &entity.Task{}, &entity.Friends{},
	} {
		if !db.db.Migrator().HasTable(model) {
			t.Errorf("table for %T was not created", model)
		}
	}
}

// ---------------------------------------------------------------------- users

func TestAddAndGetUser(t *testing.T) {
	db := newDB(t)
	ctx := context.Background()

	user := entity.User{ID: 1001, Username: "ada", AvatarURL: "", CurrentRank: 1200}
	if err := db.AddUser(ctx, user); err != nil {
		t.Fatalf("could not add the user: %v", err)
	}

	got, err := db.GetUser(1001)
	if err != nil {
		t.Fatalf("could not read the user back: %v", err)
	}
	if got.Username != "ada" || got.CurrentRank != 1200 {
		t.Errorf("got %+v, want username=ada rank=1200", got)
	}
}

func TestUserExists(t *testing.T) {
	db := newDB(t)
	ctx := context.Background()

	exists, err := db.UserExists(ctx, "grace")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected the user not to exist yet")
	}

	if err := db.AddUser(ctx, entity.User{ID: 1002, Username: "grace"}); err != nil {
		t.Fatalf("could not add the user: %v", err)
	}

	exists, err = db.UserExists(ctx, "grace")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected the user to exist")
	}
}

func TestGetUserFromName(t *testing.T) {
	db := newDB(t)
	ctx := context.Background()
	if err := db.AddUser(ctx, entity.User{ID: 1003, Username: "linus"}); err != nil {
		t.Fatalf("could not add the user: %v", err)
	}

	got, err := db.GetUserFromName(ctx, "linus")
	if err != nil {
		t.Fatalf("could not look the user up: %v", err)
	}
	if got.ID != 1003 {
		t.Errorf("expected id 1003, got %d", got.ID)
	}
}

func TestGetUserFromNameMissing(t *testing.T) {
	db := newDB(t)
	if _, err := db.GetUserFromName(context.Background(), "nobody"); err == nil {
		t.Error("expected an error for a missing user")
	}
}

func TestChangeRank(t *testing.T) {
	db := newDB(t)
	ctx := context.Background()
	if err := db.AddUser(ctx, entity.User{ID: 1004, Username: "ken", CurrentRank: 1000}); err != nil {
		t.Fatalf("could not add the user: %v", err)
	}

	if err := db.ChangeRank(1004, 1450); err != nil {
		t.Fatalf("could not change the rank: %v", err)
	}

	got, err := db.GetRank(ctx, 1004)
	if err != nil {
		t.Fatalf("could not read the rank: %v", err)
	}
	if got != 1450 {
		t.Errorf("expected rank 1450, got %d", got)
	}
}

// TestSearchPeopleQueriesTheUsernameColumn is a regression test.
//
// SearchPeople filtered on "user_name", but the User model has a field named
// Username, which gorm maps to the column "username". Every search therefore
// failed with "column user_name does not exist".
func TestSearchPeopleQueriesTheUsernameColumn(t *testing.T) {
	db := newDB(t)
	ctx := context.Background()
	for _, u := range []entity.User{
		{ID: 2001, Username: "alanna"},
		{ID: 2002, Username: "alan"},
		{ID: 2003, Username: "someone-else"},
	} {
		if err := db.AddUser(ctx, u); err != nil {
			t.Fatalf("could not add %s: %v", u.Username, err)
		}
	}

	got, err := db.SearchPeople(ctx, "alan", 10)
	if err != nil {
		t.Fatalf("SearchPeople failed: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 matches, got %d (%v)", len(got), got)
	}
	for _, u := range got {
		if u.ID == 2003 {
			t.Error("SearchPeople returned a non-matching user")
		}
	}
}

// ---------------------------------------------------------------------- rooms

func TestCreateRoomAndMembership(t *testing.T) {
	db := newDB(t)
	ctx := context.Background()

	room := entity.Room{ID: 3001, Name: "devs", Desc: "the good room", Public: entity.PUBLIC}
	if err := db.CreateRoom(ctx, room); err != nil {
		t.Fatalf("could not create the room: %v", err)
	}

	member := entity.Room_User{RoomID: 3001, UserID: 1001, Perm: entity.MOD}
	if err := db.AddMember(ctx, member); err != nil {
		t.Fatalf("could not add the member: %v", err)
	}

	got, err := db.SeePerms(ctx, entity.Room_User{RoomID: 3001, UserID: 1001})
	if err != nil {
		t.Fatalf("could not read the membership: %v", err)
	}
	if got.Perm != entity.MOD {
		t.Errorf("expected MOD, got %v", got.Perm)
	}
}

func TestUpdateMembership(t *testing.T) {
	db := newDB(t)
	ctx := context.Background()

	if err := db.CreateRoom(ctx, entity.Room{ID: 3002, Name: "r"}); err != nil {
		t.Fatalf("could not create the room: %v", err)
	}
	if err := db.AddMember(ctx, entity.Room_User{RoomID: 3002, UserID: 1001, Perm: entity.MEMBER}); err != nil {
		t.Fatalf("could not add the member: %v", err)
	}

	if err := db.UpdateMembership(ctx, entity.Room_User{RoomID: 3002, UserID: 1001, Perm: entity.BLOCKED}); err != nil {
		t.Fatalf("could not update: %v", err)
	}

	got, err := db.SeePerms(ctx, entity.Room_User{RoomID: 3002, UserID: 1001})
	if err != nil {
		t.Fatalf("could not read the membership: %v", err)
	}
	if got.Perm != entity.BLOCKED {
		t.Errorf("expected BLOCKED, got %v", got.Perm)
	}
}

func TestSeePermsMissingMembership(t *testing.T) {
	db := newDB(t)
	if _, err := db.SeePerms(context.Background(), entity.Room_User{RoomID: 9999, UserID: 9999}); err == nil {
		t.Error("expected an error for a membership that does not exist")
	}
}

// TestPageListReturnsRoomsForTheUser is a regression test.
//
// PageList filtered the rooms table on "user_id", which is not a column there;
// membership lives in room_users. The query always failed at runtime, so the
// room list endpoint could never have worked.
func TestPageListReturnsRoomsForTheUser(t *testing.T) {
	db := newDB(t)
	ctx := context.Background()

	if err := db.CreateRoom(ctx, entity.Room{ID: 3010, Name: "alpha"}); err != nil {
		t.Fatalf("could not create the room: %v", err)
	}
	if err := db.CreateRoom(ctx, entity.Room{ID: 3011, Name: "beta"}); err != nil {
		t.Fatalf("could not create the room: %v", err)
	}
	for _, m := range []entity.Room_User{
		{RoomID: 3010, UserID: 5001, Perm: entity.MOD},
		{RoomID: 3011, UserID: 5001, Perm: entity.MEMBER},
		{RoomID: 3011, UserID: 5002, Perm: entity.MOD},
	} {
		if err := db.AddMember(ctx, m); err != nil {
			t.Fatalf("could not add membership: %v", err)
		}
	}

	rooms, err := db.PageList(ctx, 5001, 0, 10)
	if err != nil {
		t.Fatalf("PageList failed: %v", err)
	}
	if len(rooms) != 2 {
		t.Fatalf("expected 2 rooms for user 5001, got %d", len(rooms))
	}

	// A different user must not see those rooms.
	other, err := db.PageList(ctx, 5002, 0, 10)
	if err != nil {
		t.Fatalf("PageList failed: %v", err)
	}
	if len(other) != 1 || other[0].ID != 3011 {
		t.Errorf("expected only room 3011, got %+v", other)
	}
}

func TestPageListPaginates(t *testing.T) {
	db := newDB(t)
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		id := uint32(4000 + i)
		if err := db.CreateRoom(ctx, entity.Room{ID: id}); err != nil {
			t.Fatalf("could not create room %d: %v", id, err)
		}
		if err := db.AddMember(ctx, entity.Room_User{RoomID: id, UserID: 6001, Perm: entity.MEMBER}); err != nil {
			t.Fatalf("could not add membership: %v", err)
		}
	}

	first, err := db.PageList(ctx, 6001, 0, 2)
	if err != nil {
		t.Fatalf("PageList failed: %v", err)
	}
	if len(first) != 2 {
		t.Fatalf("expected 2 rooms on the first page, got %d", len(first))
	}

	second, err := db.PageList(ctx, 6001, 1, 2)
	if err != nil {
		t.Fatalf("PageList failed: %v", err)
	}
	if len(second) != 2 {
		t.Fatalf("expected 2 rooms on the second page, got %d", len(second))
	}

	for _, r := range first {
		for _, s := range second {
			if r.ID == s.ID {
				t.Errorf("room %d appeared on both pages", r.ID)
			}
		}
	}
}

func TestPageListForUserWithNoRooms(t *testing.T) {
	db := newDB(t)
	rooms, err := db.PageList(context.Background(), 999999, 0, 10)
	if err != nil {
		t.Fatalf("PageList failed: %v", err)
	}
	if len(rooms) != 0 {
		t.Errorf("expected no rooms, got %d", len(rooms))
	}
}

// ------------------------------------------------------------------- contests

func TestNewContestAndReadBack(t *testing.T) {
	db := newDB(t)
	ctx := context.Background()

	contest := entity.Contest{
		ID:       "contest-1",
		JobID:    1,
		RoomID:   3001,
		Time:     time.Now().Add(time.Hour).UTC().Truncate(time.Second),
		Data:     "Friday Blitz",
		Lang:     entity.CPP,
		Duration: entity.STANDARD,
		LobbyID:  0,
		Status:   entity.UPCOMING,
	}
	if err := db.NewContest(ctx, contest); err != nil {
		t.Fatalf("could not create the contest: %v", err)
	}

	got, err := db.GetContestData(1)
	if err != nil {
		t.Fatalf("GetContestData failed: %v", err)
	}
	if got.ID != "contest-1" || got.Data != "Friday Blitz" {
		t.Errorf("unexpected contest: %+v", got)
	}
}

// TestGetContestDataUsesABoundParameter is a regression test.
//
// The query read d.db.Where("job_id = ", job_id): the placeholder was missing,
// so gorm emitted the truncated fragment "job_id = " and every call failed with
// a SQL syntax error. This is the function the scheduler uses to drive a whole
// contest, so no contest could ever start.
func TestGetContestDataUsesABoundParameter(t *testing.T) {
	db := newDB(t)
	ctx := context.Background()

	if err := db.NewContest(ctx, entity.Contest{ID: "c-2", JobID: 42}); err != nil {
		t.Fatalf("could not create the contest: %v", err)
	}

	if _, err := db.GetContestData(42); err != nil {
		t.Fatalf("GetContestData failed, the query is not binding its argument: %v", err)
	}
}

func TestGetContestDataMissing(t *testing.T) {
	db := newDB(t)
	if _, err := db.GetContestData(12345); err == nil {
		t.Error("expected an error for a contest that does not exist")
	}
}

func TestUpdateContest(t *testing.T) {
	db := newDB(t)
	ctx := context.Background()

	if err := db.NewContest(ctx, entity.Contest{ID: "c-3", JobID: 7, Status: entity.UPCOMING}); err != nil {
		t.Fatalf("could not create the contest: %v", err)
	}

	if err := db.UpdateContest(entity.Contest{JobID: 7, Status: entity.LOBBY, LobbyID: 55}); err != nil {
		t.Fatalf("could not update the contest: %v", err)
	}

	got, err := db.GetContestData(7)
	if err != nil {
		t.Fatalf("could not read the contest: %v", err)
	}
	if got.Status != entity.LOBBY || got.LobbyID != 55 {
		t.Errorf("expected LOBBY/55, got %s/%d", contestStatusName(got.Status), got.LobbyID)
	}
}

func contestStatusName(s entity.ContestStatus) string {
	return [...]string{"UPCOMING", "LOBBY", "STARTED", "ENDED"}[s]
}

// TestListContestsPaginates checks the contract the service relies on: the
// second argument is the page index and the third is the page size, and pages
// do not overlap.
func TestListContestsPaginates(t *testing.T) {
	db := newDB(t)
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		c := entity.Contest{
			ID:     "paged-" + itoa(i),
			JobID:  uint32(100 + i),
			RoomID: 7001,
			Time:   time.Now().Add(time.Duration(i) * time.Hour),
		}
		if err := db.NewContest(ctx, c); err != nil {
			t.Fatalf("could not create contest %d: %v", i, err)
		}
	}

	page0, err := db.ListContests(7001, 0, 2)
	if err != nil {
		t.Fatalf("ListContests failed: %v", err)
	}
	page1, err := db.ListContests(7001, 1, 2)
	if err != nil {
		t.Fatalf("ListContests failed: %v", err)
	}

	if len(page0) != 2 {
		t.Errorf("expected 2 contests on page 0, got %d", len(page0))
	}
	if len(page1) != 2 {
		t.Errorf("expected 2 contests on page 1, got %d", len(page1))
	}
	for _, a := range page0 {
		for _, b := range page1 {
			if a.ID == b.ID {
				t.Errorf("contest %s appears on both pages", a.ID)
			}
		}
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

// --------------------------------------------------------------------- tasks

func TestNewTaskAndPop(t *testing.T) {
	db := newDB(t)

	if err := db.NewTask(entity.Task{Category: entity.CONTEST, JobID: 9001, Time: time.Now().Add(time.Hour)}); err != nil {
		t.Fatalf("could not add the task: %v", err)
	}

	task, err := db.PopRecentTask()
	if err != nil {
		t.Fatalf("PopRecentTask failed: %v", err)
	}
	if task.JobID != 9001 {
		t.Errorf("expected job 9001, got %d", task.JobID)
	}
}

func TestPopRecentTaskOrdersByTime(t *testing.T) {
	db := newDB(t)
	now := time.Now()

	if err := db.NewTask(entity.Task{JobID: 9002, Time: now.Add(2 * time.Hour)}); err != nil {
		t.Fatalf("could not add the far task: %v", err)
	}
	if err := db.NewTask(entity.Task{JobID: 9003, Time: now}); err != nil {
		t.Fatalf("could not add the near task: %v", err)
	}

	first, err := db.PopRecentTask()
	if err != nil {
		t.Fatalf("PopRecentTask failed: %v", err)
	}
	if first.JobID != 9003 {
		t.Errorf("expected the sooner task first, got %d", first.JobID)
	}
}

func TestPopRecentTaskRemovesTheRow(t *testing.T) {
	db := newDB(t)
	if err := db.NewTask(entity.Task{JobID: 9004, Time: time.Now()}); err != nil {
		t.Fatalf("could not add the task: %v", err)
	}
	if _, err := db.PopRecentTask(); err != nil {
		t.Fatalf("PopRecentTask failed: %v", err)
	}

	var count int64
	if err := db.db.Model(&entity.Task{}).Where("job_id = ?", 9004).Count(&count).Error; err != nil {
		t.Fatalf("could not count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected the task to be deleted, %d rows remain", count)
	}
}

// TestPopRecentTaskWithEmptyTable covers the idle path the scheduler relies on
// to decide it has nothing scheduled.
func TestPopRecentTaskWithEmptyTable(t *testing.T) {
	db := newDB(t)

	task, err := db.PopRecentTask()
	if err != nil {
		t.Fatalf("an empty queue should not be an error, got %v", err)
	}
	if task.JobID != 0 {
		t.Errorf("expected the zero task, got %+v", task)
	}
}

// ------------------------------------------------------------------- friends

func TestFollowThenBecomeFriends(t *testing.T) {
	db := newDB(t)
	ctx := context.Background()

	// ada follows grace: one-way, so a single FOLLOWS row.
	if err := db.FollowUser(ctx, 1, 2); err != nil {
		t.Fatalf("could not follow: %v", err)
	}

	var rel entity.Friends
	if err := db.db.Where("A = ? AND B = ?", 1, 2).First(&rel).Error; err != nil {
		t.Fatalf("could not read the relation: %v", err)
	}
	if rel.Relation != entity.FOLLOWS {
		t.Errorf("expected FOLLOWS, got %v", rel.Relation)
	}

	// grace follows ada back: that should upgrade the pair to FRIENDS.
	if err := db.FollowUser(ctx, 2, 1); err != nil {
		t.Fatalf("could not follow back: %v", err)
	}

	friends, err := db.GetFriends(ctx, 1)
	if err != nil {
		t.Fatalf("GetFriends failed: %v", err)
	}
	if len(friends) != 1 || friends[0] != 2 {
		t.Errorf("expected 1 friend (2), got %v", friends)
	}
}

func TestUnfollowOneWay(t *testing.T) {
	db := newDB(t)
	ctx := context.Background()

	if err := db.FollowUser(ctx, 1, 2); err != nil {
		t.Fatalf("could not follow: %v", err)
	}
	if err := db.UnfollowUser(ctx, 1, 2); err != nil {
		t.Fatalf("could not unfollow: %v", err)
	}

	var count int64
	if err := db.db.Model(&entity.Friends{}).Where("A = ? AND B = ?", 1, 2).Count(&count).Error; err != nil {
		t.Fatalf("could not count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected the follow to be gone, %d rows remain", count)
	}
}

func TestUnfollowWithNoRelationIsNoop(t *testing.T) {
	db := newDB(t)
	if err := db.UnfollowUser(context.Background(), 1, 2); err != nil {
		t.Errorf("unfollowing a stranger should be a no-op, got %v", err)
	}
}

func TestGetFriendsIsSymmetric(t *testing.T) {
	db := newDB(t)
	ctx := context.Background()

	if err := db.FollowUser(ctx, 1, 2); err != nil {
		t.Fatalf("could not follow: %v", err)
	}
	if err := db.FollowUser(ctx, 2, 1); err != nil {
		t.Fatalf("could not follow back: %v", err)
	}

	for _, id := range []uint32{1, 2} {
		friends, err := db.GetFriends(ctx, id)
		if err != nil {
			t.Fatalf("GetFriends(%d) failed: %v", id, err)
		}
		if len(friends) != 1 {
			t.Errorf("user %d: expected 1 friend, got %v", id, friends)
		}
	}
}

// gormErrRecordNotFound keeps the import of gorm meaningful if the assertions
// above ever need to compare against it.
var _ = gorm.ErrRecordNotFound
