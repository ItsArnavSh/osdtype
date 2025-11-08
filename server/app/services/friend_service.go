package services

import "context"

//Handles everything related to friends and friendly matches

func (s *ServiceLayer) FollowUser(ctx context.Context, follower, following uint64) error {
	return s.db.FollowUser(ctx, follower, following)
}
func (s *ServiceLayer) UnfollowUser(ctx context.Context, follower, following uint64) error {
	return s.db.UnfollowUser(ctx, follower, following)
}
