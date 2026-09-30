package entity

// The friend graph.
//
// A relation is a directed edge with a composite primary key of (a, b), so a
// mutual pair is stored as two rows: (me, them, FRIENDS) and (them, me,
// FRIENDS). That is what makes listing connections a join rather than a plain
// read, and it is why an unqualified "where a = me or b = me" returns every
// friend twice.

// Relation is a friendship's state.
type Relation int

const (
	// FOLLOWS is a one-way follow: a follows b, b has not followed back.
	FOLLOWS Relation = iota
	// FRIENDS is mutual.
	FRIENDS
	// PENDING is a request that has not been accepted. Nothing writes it yet;
	// the model is follow-immediately-mutual, which is what a friend list
	// without a pending state means. The value exists so a future
	// request-and-accept flow does not have to renumber the table.
	PENDING
)

// String is the wire name for a relation.
func (r Relation) String() string {
	switch r {
	case FOLLOWS:
		return "following"
	case FRIENDS:
		return "friends"
	case PENDING:
		return "pending"
	}
	return "unknown"
}

// Friends is the stored edge.
//
// The two zero-valued field names are the table's column names. They are
// unexported types' worth of naming, not a style choice: renaming them would
// need a migration, and the composite primary key is built from them.
type Friends struct {
	A        uint32 `gorm:"primaryKey"`
	B        uint32 `gorm:"primaryKey"`
	Relation Relation
}

// Other returns the person on the far side of the edge from id.
//
// This is the single place that knows which end of the row is "them", because
// getting it backwards lists a player's own account as a friend, and doing it
// per query is how that happened before.
func (f Friends) Other(id uint32) uint32 {
	if f.A == id {
		return f.B
	}
	return f.A
}

// Involves reports whether the edge touches a user.
func (f Friends) Involves(id uint32) bool { return f.A == id || f.B == id }

// Friendship is one person's connection to another, as a query returns it.
//
// It is a projection, not a table. The stored edge cannot carry the other
// person's name, rating or online state without a join, and the join is what
// ListFriends does once rather than once per friend.
type Friendship struct {
	// FriendID is the other person, whichever direction the stored edge points.
	FriendID uint32 `json:"friend_id"`
	// Followed is true when the caller follows them.
	Followed bool `json:"followed"`
	// Follows is true when they follow the caller.
	Follows bool `json:"follows"`
	// Mutual is both, which is what the interface calls a friend rather than
	// someone merely followed.
	Mutual bool `json:"mutual"`

	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	Rank      uint16 `json:"rank"`
}

// Invite is the legacy invitation payload.
//
// It is no longer sent. Invitations are entity.LobbyInvite, carried in a
// notification with a share code, a password hint and an expiry, so a stale
// one can be ignored. This stays only so an older in-flight client that still
// reads the old shape does not fail to decode the frame.
type Invite struct {
	From    string `json:"from"`
	LobbyID uint32 `json:"lobby_id"`
}
