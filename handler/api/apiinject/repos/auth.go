package repos

import (
	"net/http"
	"context"

	"github.com/drone/drone/handler/api/request"
	"github.com/drone/drone/core"
	"github.com/drone/drone/handler/api/render"
	"github.com/drone/drone/logger"

	"github.com/go-chi/chi"
)

// 这是一个很hack的接口
// 返回项目的管理用户

type wrapper struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	Machine   bool   `json:"machine"`
	Admin     bool   `json:"admin"`
	Active    bool   `json:"active"`
	Avatar    string `json:"avatar"`
	Syncing   bool   `json:"syncing"`
	Synced    int64  `json:"synced"`
	Created   int64  `json:"created"`
	Updated   int64  `json:"updated"`
	LastLogin int64  `json:"last_login"`
	Token     string `json:"token"`
	Refresh   string `json:"refresh"`
	Expiry    int64  `json:"expiry"`
	Hash      string `json:"hash"`
}

// HandleUsers 返回项目的所有管理用户包含 token 信息
func HandleUsers(
	repos core.RepositoryStore,
	perms core.PermStore,
	users core.UserStore,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取到当前用户，判断是否未管理员
		user, _ := request.UserFrom(r.Context())
		if !user.Admin {
			logger.FromRequest(r).
				Warnln("api: user is not admin")
			return
		}

		// 获取 namespace and name
		var (
			ctx = r.Context()

			owner = chi.URLParam(r, "owner")
			name  = chi.URLParam(r, "name")
		)

		// 获取项目信息
		repo, err := repos.FindName(r.Context(), owner, name)
		if err != nil {
			render.NotFound(w, render.ErrNotFound)
			logger.FromRequest(r).
				WithError(render.ErrNotFound).
				WithField("namespace", owner).
				WithField("name", name).
				Debugln("api: repository not found")
			return
		}

		// 查询该项目的所有用户
		tus, err := findUsers(ctx, perms, users, repo.UID)
		if err != nil {
			render.InternalError(w, err)
			logger.FromRequest(r).
				WithError(err).
				WithField("namespace", owner).
				WithField("name", name).
				Errorln("api: find user error")
			return
		}

		// 返回用户信息
		var tms []*wrapper
		for _, u := range tus {
			tms = append(tms, &wrapper{
				ID: u.ID,
				Login: u.Login,
				Email: u.Email,
				Machine: u.Machine,
				Admin: u.Admin,
				Active: u.Active,
				Avatar: u.Avatar,
				Syncing: u.Syncing,
				Synced: u.Synced,
				Created: u.Created,
				Updated: u.Updated,
				LastLogin: u.LastLogin,
				Token: u.Token,
				Refresh: u.Refresh,
				Expiry: u.Expiry,
				Hash: u.Hash,
			})
		}

		render.JSON(w, tms, 200)
	}
}

func findUsers(
	ctx context.Context,
	perms core.PermStore,
	users core.UserStore,
	repoUID string,
) (tms []*core.User, err error) {

	// 查询该项目的所有用户
	cators, err := perms.List(ctx, repoUID)
	if err != nil {
		return
	}

	// 返回用户信息
	for _, c := range cators {
		u, err := users.Find(ctx, c.UserID)
		if err != nil {
			continue
		}
		tms = append(tms, u)
	}

	return
}