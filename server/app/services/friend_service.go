package services

import (
	"context"
	"osdtyp/app/entity"
)

type FriendView struct {
	User     entity.UserView `json:"user"`
	Online   bool            `json:"online"`
	IsFriend bool            `json:"is_friend"`
	Follows  bool            `json:"follows_you"`
}

func (s *ServiceLayer) FollowUser(ctx context.Context, follower, following uint32) error {
	if follower == following {
		return errf(ErrBadRequest, "you cannot follow yourself")
	}
	if _, err := s.db.GetUser(following); err != nil {
		return errf(ErrNotFound, "user not found")
	}
	return s.db.FollowUser(ctx, follower, following)
}

func (s *ServiceLayer) UnfollowUser(ctx context.Context, follower, following uint32) error {
	return s.db.UnfollowUser(ctx, follower, following)
}

func (s *ServiceLayer) Friends(ctx context.Context, uid uint32) ([]FriendView, error) {
	ids, err := s.db.GetFriends(ctx, uid)
	if err != nil {
		return nil, err
	}
	following, err := s.db.GetFollowing(ctx, uid)
	if err != nil {
		return nil, err
	}
	seen := make(map[uint32]bool, len(ids))
	out := make([]FriendView, 0, len(ids))
	for _, id := range ids {
		u, err := s.db.GetUser(id)
		if err != nil {
			continue
		}
		online, _ := s.rdb().IsOnline(ctx, id)
		out = append(out, FriendView{User: entity.NewUserView(u), Online: online, IsFriend: true})
		seen[id] = true
	}
	for _, id := range following {
		if seen[id] {
			continue
		}
		u, err := s.db.GetUser(id)
		if err != nil {
			continue
		}
		online, _ := s.rdb().IsOnline(ctx, id)
		out = append(out, FriendView{User: entity.NewUserView(u), Online: online})
	}
	return out, nil
}
