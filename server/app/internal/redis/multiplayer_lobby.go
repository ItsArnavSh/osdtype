//This is for the multiplayer lobby

package redis

import (
	"context"
	"fmt"

	"github.com/go-redis/redis"
)

func (r *RedisClient) JoinLobby(ctx context.Context, lobbyID uint32, playerID uint32, rank uint16) error {
	key := fmt.Sprintf("Lobby:%d", lobbyID)
	_, err := r.redis_conn.ZAdd(key, redis.Z{
		Score:  float64(rank),
		Member: playerID,
	}).Result()
	return err
}

func (r *RedisClient) GetClosestLobby() {

}
