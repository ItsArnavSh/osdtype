package entity

import "time"

type RoomStatus string

const (
	RoomLobby   RoomStatus = "lobby"
	RoomPlaying RoomStatus = "playing"
	RoomClosed  RoomStatus = "closed"
)

type RoomRole string

const (
	RoleOwner  RoomRole = "owner"
	RoleMod    RoomRole = "mod"
	RoleMember RoomRole = "member"
)

type Room struct {
	ID           uint32     `gorm:"primaryKey;autoIncrement:false" json:"id"`
	Code         string     `gorm:"size:12;uniqueIndex;not null" json:"code"`
	Name         string     `gorm:"size:64;not null" json:"name"`
	Description  string     `gorm:"size:280" json:"description"`
	IsPublic     bool       `gorm:"not null;default:false" json:"is_public"`
	PasswordHash string     `gorm:"size:120" json:"-"`
	MaxPlayers   int        `gorm:"not null;default:8" json:"max_players"`
	HostID       uint32     `gorm:"not null" json:"host_id"`
	Status       RoomStatus `gorm:"size:16;not null;default:lobby" json:"status"`
	Duration     int        `gorm:"not null;default:90" json:"duration"`
	Language     string     `gorm:"size:16;not null;default:go" json:"language"`
	GameID       uint32     `gorm:"not null;default:0" json:"game_id"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	Members []RoomMember `gorm:"-" json:"members,omitempty"`
}

type RoomMember struct {
	RoomID   uint32    `gorm:"primaryKey" json:"room_id"`
	UserID   uint32    `gorm:"primaryKey" json:"user_id"`
	Role     RoomRole  `gorm:"size:16;not null;default:member" json:"role"`
	JoinedAt time.Time `json:"joined_at"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type RoomBan struct {
	RoomID    uint32    `gorm:"primaryKey" json:"room_id"`
	UserID    uint32    `gorm:"primaryKey" json:"user_id"`
	BannedBy  uint32    `json:"banned_by"`
	CreatedAt time.Time `json:"created_at"`
}

type MemberView struct {
	UserID    uint32    `json:"user_id"`
	Username  string    `json:"username"`
	AvatarURL string    `json:"avatar_url"`
	Rating    int       `json:"rating"`
	Rank      RankInfo  `json:"rank"`
	Role      RoomRole  `json:"role"`
	JoinedAt  time.Time `json:"joined_at"`
	IsBot     bool      `json:"is_bot"`
	Online    bool      `json:"online"`
}

type RoomView struct {
	ID          uint32       `json:"id"`
	Code        string       `json:"code"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	IsPublic    bool         `json:"is_public"`
	HasPassword bool         `json:"has_password"`
	MaxPlayers  int          `json:"max_players"`
	HostID      uint32       `json:"host_id"`
	Status      RoomStatus   `json:"status"`
	Duration    int          `json:"duration"`
	Language    string       `json:"language"`
	GameID      uint32       `json:"game_id"`
	MemberCount int          `json:"member_count"`
	Members     []MemberView `json:"members"`
	MyRole      RoomRole     `json:"my_role,omitempty"`
	InviteLink  string       `json:"invite_link,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
}
