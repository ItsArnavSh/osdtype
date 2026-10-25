package services

import (
	"context"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"osdtyp/app/entity"
)

// ProfileSettingsView is the authenticated account-settings projection. It is
// intentionally separate from UserView: sharing switches are never exposed on
// another player's profile.
type ProfileSettingsView struct {
	Username           string `json:"username"`
	DisplayName        string `json:"display_name"`
	Bio                string `json:"bio"`
	ProfileVisibility  string `json:"profile_visibility"`
	DoNotDisturb       bool   `json:"do_not_disturb"`
	HideOnline         bool   `json:"hide_online"`
	ShareRuns          string `json:"share_runs"`
	ShareRanked        bool   `json:"share_ranked"`
	ShareRooms         bool   `json:"share_rooms"`
	SharePractice      bool   `json:"share_practice"`
	ShareContests      bool   `json:"share_contests"`
	HidePrivateRooms   bool   `json:"hide_private_rooms"`
	ShareReplays       bool   `json:"share_replays"`
	SaveRunData        bool   `json:"save_run_data"`
	RetentionLimit     int    `json:"retention_limit"`
	RetentionKeepsPin  bool   `json:"retention_keeps_pinned"`
	RetentionKeepsBest bool   `json:"retention_keeps_best"`
}

// ProfileSettingsRequest uses pointers so a partial save never clears fields
// the caller did not intend to touch.
type ProfileSettingsRequest struct {
	DisplayName       *string `json:"display_name"`
	Bio               *string `json:"bio"`
	ProfileVisibility *string `json:"profile_visibility"`
	DoNotDisturb      *bool   `json:"do_not_disturb"`
	HideOnline        *bool   `json:"hide_online"`
	ShareRuns         *string `json:"share_runs"`
	ShareRanked       *bool   `json:"share_ranked"`
	ShareRooms        *bool   `json:"share_rooms"`
	SharePractice     *bool   `json:"share_practice"`
	ShareContests     *bool   `json:"share_contests"`
	HidePrivateRooms  *bool   `json:"hide_private_rooms"`
	ShareReplays      *bool   `json:"share_replays"`
	SaveRunData       *bool   `json:"save_run_data"`
}

func profileSettingsView(user entity.User) ProfileSettingsView {
	return ProfileSettingsView{
		Username:           user.Username,
		DisplayName:        user.DisplayName,
		Bio:                user.Bio,
		ProfileVisibility:  user.Visibility(),
		DoNotDisturb:       user.DoNotDisturb,
		HideOnline:         user.HideOnline,
		ShareRuns:          user.SharingMode(),
		ShareRanked:        user.ShareRanked,
		ShareRooms:         user.ShareRooms,
		SharePractice:      user.SharePractice,
		ShareContests:      user.ShareContests,
		HidePrivateRooms:   user.HidePrivateRooms,
		ShareReplays:       user.ShareReplays,
		SaveRunData:        user.SaveRunData,
		RetentionLimit:     entity.MatchRetentionLimit,
		RetentionKeepsPin:  true,
		RetentionKeepsBest: true,
	}
}

func (s *ServiceLayer) ProfileSettings(ctx context.Context, uid uint32) (ProfileSettingsView, error) {
	user, err := s.GetUser(ctx, uid)
	if err != nil {
		return ProfileSettingsView{}, err
	}
	return profileSettingsView(user), nil
}

func (s *ServiceLayer) UpdateProfileSettings(ctx context.Context, uid uint32, req ProfileSettingsRequest) (ProfileSettingsView, error) {
	updates := make(map[string]any, 12)

	if req.DisplayName != nil {
		name := strings.TrimSpace(*req.DisplayName)
		if utf8.RuneCountInString(name) > maxDisplayNameRunes {
			return ProfileSettingsView{}, errf(ErrBadRequest, "display name is too long")
		}
		for _, r := range name {
			if unicode.IsControl(r) {
				return ProfileSettingsView{}, errf(ErrBadRequest, "display name cannot contain line breaks or control characters")
			}
		}
		updates["display_name"] = name
	}
	if req.Bio != nil {
		bio := strings.TrimSpace(*req.Bio)
		if utf8.RuneCountInString(bio) > maxBioRunes {
			return ProfileSettingsView{}, errf(ErrBadRequest, "bio is too long")
		}
		for _, r := range bio {
			if unicode.IsControl(r) && r != '\n' && r != '\t' {
				return ProfileSettingsView{}, errf(ErrBadRequest, "bio contains an unsupported control character")
			}
		}
		updates["bio"] = bio
	}
	if req.ProfileVisibility != nil {
		visibility := strings.ToLower(strings.TrimSpace(*req.ProfileVisibility))
		switch visibility {
		case entity.ProfilePublic, entity.ProfileAnonymous, entity.ProfilePrivate:
			updates["profile_visibility"] = visibility
		default:
			return ProfileSettingsView{}, errf(ErrBadRequest, "profile visibility must be public, anonymous, or private")
		}
	}
	if req.DoNotDisturb != nil {
		updates["do_not_disturb"] = *req.DoNotDisturb
	}
	if req.HideOnline != nil {
		updates["hide_online"] = *req.HideOnline
	}
	if req.ShareRuns != nil {
		sharing := strings.ToLower(strings.TrimSpace(*req.ShareRuns))
		switch sharing {
		case entity.RunSharingAll, entity.RunSharingPinned, entity.RunSharingNone:
			updates["share_runs"] = sharing
		default:
			return ProfileSettingsView{}, errf(ErrBadRequest, "share runs must be all, pinned, or none")
		}
	}
	if req.ShareRanked != nil {
		updates["share_ranked"] = *req.ShareRanked
	}
	if req.ShareRooms != nil {
		updates["share_rooms"] = *req.ShareRooms
	}
	if req.SharePractice != nil {
		updates["share_practice"] = *req.SharePractice
	}
	if req.ShareContests != nil {
		updates["share_contests"] = *req.ShareContests
	}
	if req.HidePrivateRooms != nil {
		updates["hide_private_rooms"] = *req.HidePrivateRooms
	}
	if req.ShareReplays != nil {
		updates["share_replays"] = *req.ShareReplays
	}
	if req.SaveRunData != nil {
		updates["save_run_data"] = *req.SaveRunData
	}
	if len(updates) == 0 {
		return ProfileSettingsView{}, errf(ErrBadRequest, "nothing to update")
	}
	if _, err := s.GetUser(ctx, uid); err != nil {
		return ProfileSettingsView{}, err
	}
	if err := s.db.UpdateUserFields(ctx, uid, updates); err != nil {
		return ProfileSettingsView{}, err
	}
	updated, err := s.GetUser(ctx, uid)
	if err != nil {
		return ProfileSettingsView{}, err
	}
	return profileSettingsView(updated), nil
}

// sharedRuns filters a profile history list through the owner's switches.
func sharedRuns(user entity.User, rows []entity.MatchHistory) []entity.MatchHistory {
	out := make([]entity.MatchHistory, 0, len(rows))
	for _, row := range rows {
		if user.SharesRun(row.Mode, row.Pinned, row.RoomPrivate) {
			out = append(out, row)
		}
	}
	return out
}

// anonymousUserView removes the identifying fields from a public profile while
// preserving the ladder stats and whatever runs the owner chose to share.
func anonymousUserView(user entity.User) entity.UserView {
	view := entity.NewUserView(user)
	view.Username = "anonymous"
	view.DisplayName = "anonymous"
	view.Bio = ""
	view.AvatarURL = ""
	return view
}

// pruneAllMatchHistories applies the 30-unpinned-run retention policy to every
// account. Pinned runs are deliberately exempt.
func (s *ServiceLayer) pruneAllMatchHistories(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	defer cancel()
	removed, err := s.db.PruneAllMatchHistories(ctx, entity.MatchRetentionLimit)
	if err != nil {
		s.logger.Warnw("prune all match histories", "error", err)
		return
	}
	if removed > 0 {
		s.logger.Infow("pruned match histories", "runs", removed, "keep_unpinned", entity.MatchRetentionLimit)
	}
}
