package entity

import "time"

type GameMode string

const (
	GameRanked   GameMode = "ranked"
	GameRoom     GameMode = "room"
	GameContest  GameMode = "contest"
	GamePractice GameMode = "practice"
)

type GameStatus string

const (
	GameCountdown GameStatus = "countdown"
	GameActive    GameStatus = "active"
	GameFinished  GameStatus = "finished"
	GameAbandoned GameStatus = "abandoned"
)

type Game struct {
	ID             uint32     `gorm:"primaryKey;autoIncrement:false" json:"id"`
	Mode           GameMode   `gorm:"size:16;not null" json:"mode"`
	RoomID         uint32     `gorm:"index;not null;default:0" json:"room_id"`
	ContestID      uint32     `gorm:"index;not null;default:0" json:"contest_id"`
	ContestStageID uint32     `gorm:"index;not null;default:0" json:"contest_stage_id"`
	Duration       int        `gorm:"not null" json:"duration"`
	Language       string     `gorm:"size:16;not null" json:"language"`
	Seed           uint64     `gorm:"not null" json:"seed"`
	Snippet        string     `gorm:"type:text" json:"-"`
	StartOffset    int        `gorm:"not null;default:0" json:"start_offset"`
	Status         GameStatus `gorm:"size:16;not null;default:countdown" json:"status"`
	// Temporary marks a practice run that is only kept as the player's
	// latest attempt. Saving it clears the flag and puts it in history.
	Temporary  bool       `gorm:"not null;default:false;index" json:"temporary"`
	StartsAt   *time.Time `json:"starts_at"`
	EndsAt     *time.Time `json:"ends_at"`
	CreatedAt  time.Time  `json:"created_at"`
	FinishedAt *time.Time `json:"finished_at"`

	Players []GamePlayer `gorm:"-" json:"players,omitempty"`
}

type GamePlayer struct {
	GameID        uint32  `gorm:"primaryKey" json:"game_id"`
	UserID        uint32  `gorm:"primaryKey" json:"user_id"`
	RatingBefore  int     `json:"rating_before"`
	RatingAfter   int     `json:"rating_after"`
	WPM           float64 `json:"wpm"`
	Raw           float64 `json:"raw"`
	Accuracy      float64 `json:"accuracy"`
	Correct       int     `json:"correct"`
	Wrong         int     `json:"wrong"`
	CurrentPoints int     `json:"current_points"`
	Placement     int     `json:"placement"`
	Finished      bool    `json:"finished"`
	Left          bool    `json:"left"`
	Flagged       bool    `json:"flagged"`
	IsBot         bool    `json:"is_bot"`
	Acked         bool    `json:"-"`

	// Shareable run presentation. These are set by the player from their
	// profile; they do not affect scoring.
	RunName        string     `gorm:"size:80" json:"run_name,omitempty"`
	RunDescription string     `gorm:"size:500" json:"run_description,omitempty"`
	Pinned         bool       `gorm:"not null;default:false;index" json:"pinned"`
	PinnedAt       *time.Time `json:"pinned_at,omitempty"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type ResultView struct {
	UserID       uint32  `json:"user_id"`
	Username     string  `json:"username"`
	AvatarURL    string  `json:"avatar_url"`
	Rank         int     `json:"rank"`
	WPM          float64 `json:"wpm"`
	Raw          float64 `json:"raw"`
	Accuracy     float64 `json:"accuracy"`
	Correct      int     `json:"correct"`
	Wrong        int     `json:"wrong"`
	RatingBefore int     `json:"rating_before"`
	RatingAfter  int     `json:"rating_after"`
	RatingDelta  int     `json:"rating_delta"`
	Flagged      bool    `json:"flagged"`
	Left         bool    `json:"left"`
	IsBot        bool    `json:"is_bot"`
}

type PlayerProgress struct {
	UserID        uint32  `json:"user_id"`
	Username      string  `json:"username"`
	AvatarURL     string  `json:"avatar_url"`
	CurrentPoints int     `json:"current_points"`
	WPM           float64 `json:"wpm"`
	Accuracy      float64 `json:"accuracy"`
	Finished      bool    `json:"finished"`
	Flagged       bool    `json:"flagged"`
	IsBot         bool    `json:"is_bot"`
}

type GameView struct {
	ID                uint32           `json:"id"`
	Mode              GameMode         `json:"mode"`
	Status            GameStatus       `json:"status"`
	RoomID            uint32           `json:"room_id"`
	RoomCode          string           `json:"room_code"`
	ContestID         uint32           `json:"contest_id"`
	ContestCode       string           `json:"contest_code,omitempty"`
	ContestStageID    uint32           `json:"contest_stage_id,omitempty"`
	ContestStageName  string           `json:"contest_stage_name,omitempty"`
	ContestStageIndex int              `json:"contest_stage_index,omitempty"`
	Duration          int              `json:"duration"`
	Language          string           `json:"language"`
	Seed              string           `json:"seed"`
	Snippet           string           `json:"snippet"`
	StartOffset       int              `json:"start_offset"`
	Temporary         bool             `json:"temporary,omitempty"`
	StartsAt          *time.Time       `json:"starts_at"`
	EndsAt            *time.Time       `json:"ends_at"`
	Players           []PlayerProgress `json:"players"`
	Leaderboard       []ResultView     `json:"leaderboard"`
	MyResult          *ResultView      `json:"my_result,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
}

type RatingHistory struct {
	ID          uint32    `gorm:"primaryKey;autoIncrement:true" json:"id"`
	UserID      uint32    `gorm:"index;not null" json:"user_id"`
	GameID      uint32    `gorm:"index;not null" json:"game_id"`
	RatingDelta int       `json:"rating_delta"`
	RatingAfter int       `json:"rating_after"`
	Mode        GameMode  `gorm:"size:16" json:"mode"`
	CreatedAt   time.Time `json:"created_at"`
}

type CheatFlag struct {
	ID        uint32    `gorm:"primaryKey;autoIncrement:true" json:"id"`
	GameID    uint32    `gorm:"index;not null" json:"game_id"`
	UserID    uint32    `gorm:"index;not null" json:"user_id"`
	Reason    string    `gorm:"size:120" json:"reason"`
	Risk      int       `json:"risk"`
	CreatedAt time.Time `json:"created_at"`
}

type Recording struct {
	ID        uint32    `gorm:"primaryKey;autoIncrement:true" json:"id"`
	GameID    uint32    `gorm:"index;not null" json:"game_id"`
	UserID    uint32    `gorm:"index;not null" json:"user_id"`
	Events    []byte    `gorm:"type:bytea" json:"-"`
	Final     string    `gorm:"type:text" json:"final"`
	CreatedAt time.Time `json:"created_at"`
}

type MatchHistory struct {
	GameID         uint32     `json:"game_id"`
	UserID         uint32     `json:"user_id"`
	Mode           GameMode   `json:"mode"`
	RoomCode       string     `json:"room_code,omitempty"`
	Language       string     `json:"language"`
	Duration       int        `json:"duration"`
	WPM            float64    `json:"wpm"`
	Accuracy       float64    `json:"accuracy"`
	Rank           int        `json:"rank"`
	Players        int        `json:"players"`
	RatingDelta    int        `json:"rating_delta"`
	Flagged        bool       `json:"flagged"`
	PlayedAt       time.Time  `json:"played_at"`
	RunName        string     `json:"run_name,omitempty"`
	RunDescription string     `json:"run_description,omitempty"`
	Pinned         bool       `json:"pinned"`
	PinnedAt       *time.Time `json:"pinned_at,omitempty"`
}
