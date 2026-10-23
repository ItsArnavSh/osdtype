package entity

import "time"

type ContestStageFormat string

const (
	ContestStageQualifier   ContestStageFormat = "qualifier"
	ContestStageHotLaps     ContestStageFormat = "hotlaps"
	ContestStagePractice    ContestStageFormat = "practice"
	ContestStageMultiplayer ContestStageFormat = "multiplayer"
	ContestStageRanked      ContestStageFormat = "ranked"
)

// IsOpen reports whether a stage accepts asynchronous attempts during a
// window instead of launching scheduled live heats.
func (f ContestStageFormat) IsOpen() bool {
	return f == ContestStageQualifier || f == ContestStageHotLaps || f == ContestStagePractice
}

type ContestStageStatus string

const (
	ContestStagePending   ContestStageStatus = "pending"
	ContestStageActive    ContestStageStatus = "active"
	ContestStageCompleted ContestStageStatus = "completed"
	ContestStageCancelled ContestStageStatus = "cancelled"
)

type ContestLanguageMode string

const (
	ContestLanguageFixed  ContestLanguageMode = "fixed"
	ContestLanguageChoice ContestLanguageMode = "choice"
)

type ContestAdvanceType string

const (
	ContestAdvanceCount   ContestAdvanceType = "count"
	ContestAdvancePercent ContestAdvanceType = "percent"
	ContestAdvanceAll     ContestAdvanceType = "all"
)

type ContestStatus string

const (
	ContestUpcoming ContestStatus = "upcoming"
	// ContestLobby is retained for rows created by the original room-bound
	// contest feature. New contests move straight from upcoming to started.
	ContestLobby     ContestStatus = "lobby"
	ContestStarted   ContestStatus = "started"
	ContestEnded     ContestStatus = "ended"
	ContestCancelled ContestStatus = "cancelled"
)

type ContestVisibility string

const (
	ContestPublic  ContestVisibility = "public"
	ContestPrivate ContestVisibility = "private"
)

type Contest struct {
	ID          uint32            `gorm:"primaryKey;autoIncrement:false" json:"id"`
	Code        string            `gorm:"size:16;uniqueIndex;not null" json:"code"`
	HostID      uint32            `gorm:"index;not null" json:"host_id"`
	Title       string            `gorm:"size:120;not null" json:"title"`
	Description string            `gorm:"type:text" json:"description"`
	BannerURL   string            `gorm:"size:500" json:"banner_url"`
	Accent      string            `gorm:"size:16;not null;default:#00a19c" json:"accent"`
	CustomCSS   string            `gorm:"type:text" json:"custom_css"`
	Visibility  ContestVisibility `gorm:"size:16;not null;default:public" json:"visibility"`
	// PasswordHash is only used by private contests that opt into a password.
	// Public contests always remain open to everyone.
	PasswordHash string        `gorm:"size:120" json:"-"`
	StartsAt     time.Time     `gorm:"index;not null" json:"starts_at"`
	Duration     int           `gorm:"not null;default:90" json:"duration"`
	Language     string        `gorm:"size:16;not null;default:go" json:"language"`
	MaxPlayers   int           `gorm:"not null;default:16" json:"max_players"`
	Status       ContestStatus `gorm:"size:16;not null;default:upcoming" json:"status"`
	// RoomID is an internal staging room used to launch the contest game. It
	// is never exposed as part of the contest experience.
	RoomID      uint32    `gorm:"index;not null;default:0" json:"-"`
	GameID      uint32    `json:"game_id"`
	Leaderboard []byte    `gorm:"type:bytea" json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Host   *User          `gorm:"foreignKey:HostID" json:"-"`
	Stages []ContestStage `gorm:"foreignKey:ContestID" json:"-"`
}

// ContestStage is one leg of a contest funnel. The first stage starts at the
// contest start time; later stages start a host-defined delay after the
// previous stage finishes. Open stages (qualifier and HotLaps) accept attempts
// for their whole window; live stages launch heats at the scheduled moment.
type ContestStage struct {
	ID        uint32             `gorm:"primaryKey;autoIncrement:false" json:"id"`
	ContestID uint32             `gorm:"index;not null" json:"contest_id"`
	Position  int                `gorm:"not null" json:"position"`
	Name      string             `gorm:"size:80;not null" json:"name"`
	Format    ContestStageFormat `gorm:"size:16;not null" json:"format"`
	Status    ContestStageStatus `gorm:"size:16;not null;default:pending" json:"status"`
	StartsAt  time.Time          `gorm:"index" json:"starts_at"`
	// DelayMinutes is ignored for the first stage and otherwise measured from
	// the moment the previous stage closes.
	DelayMinutes  int                 `gorm:"not null;default:0" json:"delay_minutes"`
	WindowMinutes int                 `gorm:"not null;default:0" json:"window_minutes"`
	Duration      int                 `gorm:"not null;default:90" json:"duration"`
	LanguageMode  ContestLanguageMode `gorm:"size:16;not null;default:fixed" json:"language_mode"`
	Language      string              `gorm:"size:16;not null;default:go" json:"language"`
	MaxTries      int                 `gorm:"not null;default:0" json:"max_tries"`
	MinWPM        int                 `gorm:"not null;default:0" json:"min_wpm"`
	AdvanceType   ContestAdvanceType  `gorm:"size:16;not null;default:all" json:"advance_type"`
	AdvanceValue  int                 `gorm:"not null;default:0" json:"advance_value"`
	// MaxPlayers is the maximum number of racers in a single live heat. It is
	// deliberately capped at 10 by the service layer.
	MaxPlayers int        `gorm:"not null;default:8" json:"max_players"`
	Seed       uint64     `gorm:"not null;default:0" json:"-"`
	StartedAt  *time.Time `json:"started_at"`
	EndsAt     *time.Time `json:"ends_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// ContestEntry is a durable enrollment. Contests are registration-based and
// deliberately independent from room membership. CurrentStage is the position
// of the next stage the entry is eligible to play.
type ContestEntry struct {
	ContestID         uint32    `gorm:"primaryKey" json:"contest_id"`
	UserID            uint32    `gorm:"primaryKey" json:"user_id"`
	JoinedAt          time.Time `json:"joined_at"`
	CurrentStage      int       `gorm:"not null;default:1" json:"current_stage"`
	Eliminated        bool      `gorm:"not null;default:false" json:"eliminated"`
	EliminatedAtStage int       `gorm:"not null;default:0" json:"eliminated_at_stage"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
