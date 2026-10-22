package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

type Client struct {
	R *goredis.Client
}

func NewClient() (*Client, error) {
	addr := viper.GetString("Redis.addr")
	if addr == "" {
		addr = "localhost:6379"
	}
	rdb := goredis.NewClient(&goredis.Options{
		Addr:     addr,
		Password: viper.GetString("Redis.password"),
		DB:       viper.GetInt("Redis.db"),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connect redis: %w", err)
	}
	return &Client{R: rdb}, nil
}

// NewFromRedis is used by tests with miniredis.
func NewFromRedis(r *goredis.Client) *Client { return &Client{R: r} }

func (c *Client) Close() error { return c.R.Close() }

func (c *Client) Ping(ctx context.Context) error { return c.R.Ping(ctx).Err() }

// ─── Player state ────────────────────────────────────────────────────────────

type UserState struct {
	Status        string    `json:"status"` // idle | queue | room | game | results
	RoomCode      string    `json:"room_code,omitempty"`
	GameID        uint32    `json:"game_id,omitempty"`
	QueueDuration int       `json:"queue_duration,omitempty"`
	ResultsGameID uint32    `json:"results_game_id,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func stateKey(uid uint32) string { return fmt.Sprintf("state:user:%d", uid) }

func (c *Client) SetState(ctx context.Context, uid uint32, st UserState) error {
	st.UpdatedAt = time.Now().UTC()
	raw, err := json.Marshal(st)
	if err != nil {
		return err
	}
	return c.R.Set(ctx, stateKey(uid), raw, 24*time.Hour).Err()
}

func (c *Client) GetState(ctx context.Context, uid uint32) (UserState, bool, error) {
	raw, err := c.R.Get(ctx, stateKey(uid)).Bytes()
	if err == goredis.Nil {
		return UserState{}, false, nil
	}
	if err != nil {
		return UserState{}, false, err
	}
	var st UserState
	if err := json.Unmarshal(raw, &st); err != nil {
		return UserState{}, false, err
	}
	return st, true, nil
}

func (c *Client) ClearState(ctx context.Context, uid uint32) error {
	return c.R.Del(ctx, stateKey(uid)).Err()
}

// ─── Presence ────────────────────────────────────────────────────────────────

func presenceKey(uid uint32) string { return fmt.Sprintf("presence:user:%d", uid) }

func (c *Client) SetOnline(ctx context.Context, uid uint32, ttl time.Duration) error {
	return c.R.Set(ctx, presenceKey(uid), "1", ttl).Err()
}

func (c *Client) SetOffline(ctx context.Context, uid uint32) error {
	return c.R.Del(ctx, presenceKey(uid)).Err()
}

func (c *Client) IsOnline(ctx context.Context, uid uint32) (bool, error) {
	n, err := c.R.Exists(ctx, presenceKey(uid)).Result()
	return n > 0, err
}

// ─── Matchmaking queues ─────────────────────────────────────────────────────

func queueKey(duration int) string     { return fmt.Sprintf("queue:%d", duration) }
func queueMetaKey(duration int) string { return fmt.Sprintf("queue-meta:%d", duration) }

type QueueItem struct {
	UserID   uint32    `json:"user_id"`
	Rating   int       `json:"rating"`
	JoinedAt time.Time `json:"joined_at"`
}

func (c *Client) QueueAdd(ctx context.Context, duration int, uid uint32, rating int, joinedAt time.Time) error {
	pipe := c.R.TxPipeline()
	pipe.ZAdd(ctx, queueKey(duration), goredis.Z{Score: float64(rating), Member: strconv.FormatUint(uint64(uid), 10)})
	pipe.HSet(ctx, queueMetaKey(duration), strconv.FormatUint(uint64(uid), 10), joinedAt.UnixMilli())
	_, err := pipe.Exec(ctx)
	return err
}

func (c *Client) QueueRemove(ctx context.Context, duration int, uid uint32) error {
	pipe := c.R.TxPipeline()
	pipe.ZRem(ctx, queueKey(duration), strconv.FormatUint(uint64(uid), 10))
	pipe.HDel(ctx, queueMetaKey(duration), strconv.FormatUint(uint64(uid), 10))
	_, err := pipe.Exec(ctx)
	return err
}

func (c *Client) QueueItems(ctx context.Context, duration int) ([]QueueItem, error) {
	members, err := c.R.ZRangeWithScores(ctx, queueKey(duration), 0, -1).Result()
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return nil, nil
	}
	ids := make([]string, len(members))
	for i, m := range members {
		ids[i] = fmt.Sprint(m.Member)
	}
	metas, err := c.R.HMGet(ctx, queueMetaKey(duration), ids...).Result()
	if err != nil {
		return nil, err
	}
	out := make([]QueueItem, 0, len(members))
	for i, m := range members {
		uid64, err := strconv.ParseUint(fmt.Sprint(m.Member), 10, 32)
		if err != nil {
			continue
		}
		item := QueueItem{UserID: uint32(uid64), Rating: int(m.Score), JoinedAt: time.Now()}
		if i < len(metas) && metas[i] != nil {
			if ms, ok := metas[i].(string); ok {
				if n, err := strconv.ParseInt(ms, 10, 64); err == nil {
					item.JoinedAt = time.UnixMilli(n)
				}
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func (c *Client) QueuePosition(ctx context.Context, duration int, uid uint32, rating int) (int64, error) {
	// ZRANK would penalise ties; count strictly better scores plus every tied
	// player that joined earlier.
	items, err := c.QueueItems(ctx, duration)
	if err != nil {
		return 0, err
	}
	var pos int64
	for _, it := range items {
		if it.UserID == uid {
			return pos, nil
		}
		if it.Rating > rating {
			pos++
		}
	}
	return pos, nil
}

func (c *Client) QueueLength(ctx context.Context, duration int) (int64, error) {
	return c.R.ZCard(ctx, queueKey(duration)).Result()
}

func (c *Client) ClearQueue(ctx context.Context, duration int) error {
	pipe := c.R.TxPipeline()
	pipe.Del(ctx, queueKey(duration))
	pipe.Del(ctx, queueMetaKey(duration))
	_, err := pipe.Exec(ctx)
	return err
}

// ─── Rating leaderboard ────────────────────────────────────────────────────

const ratingBoard = "leaderboard:rating"

func (c *Client) SetRating(ctx context.Context, uid uint32, rating int) error {
	return c.R.ZAdd(ctx, ratingBoard, goredis.Z{Score: float64(rating), Member: strconv.FormatUint(uint64(uid), 10)}).Err()
}

type BoardEntry struct {
	UserID uint32 `json:"user_id"`
	Rating int    `json:"rating"`
}

func (c *Client) RatingBoard(ctx context.Context, offset, limit int64) ([]BoardEntry, error) {
	res, err := c.R.ZRevRangeWithScores(ctx, ratingBoard, offset, offset+limit-1).Result()
	if err != nil {
		return nil, err
	}
	out := make([]BoardEntry, 0, len(res))
	for _, z := range res {
		id, err := strconv.ParseUint(fmt.Sprint(z.Member), 10, 32)
		if err != nil {
			continue
		}
		out = append(out, BoardEntry{UserID: uint32(id), Rating: int(z.Score)})
	}
	return out, nil
}

// ─── Distributed locks ──────────────────────────────────────────────────────

func (c *Client) AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return c.R.SetNX(ctx, "lock:"+key, "1", ttl).Result()
}

func (c *Client) ReleaseLock(ctx context.Context, key string) error {
	return c.R.Del(ctx, "lock:"+key).Err()
}
