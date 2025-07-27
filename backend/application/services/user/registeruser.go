package user

import (
	"context"
	"osdtype/application/entity"
	"osdtype/database"
)

func RegisterUser(ctx context.Context, user map[string]string, db database.Queries) {
	us := entity.User{
		Login:     user["login"],
		Name:      user["name"],
		Url:       user["url"],
		AvatarUrl: user["avatar_url"],
		HTMLUrl:   user["html_url"],
		GithubId:  user["id"],
	}
	//Todo: Update the DB rows to include all rows in this struct
	db.UpsertUser(ctx, database.UpsertUserParams{TopWpm: 0, UserID: us.GithubId, GithubID: us.GithubId, DpLink: us.HTMLUrl, Username: us.Login})
}
