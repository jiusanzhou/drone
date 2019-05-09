package repos

import (
	"net/http"
	
	"github.com/drone/drone/handler/api/request"
	"github.com/drone/drone/logger"
	"github.com/drone/drone/core"
	"github.com/drone/drone/handler/api/render"

	"github.com/go-chi/chi"
)

// HandlerBuildCreate 偷梁换柱，替换为可用的user
func HandlerBuildCreate(
	repos core.RepositoryStore,
	perms core.PermStore,
	users core.UserStore,
	next http.HandlerFunc,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		
		var (
			ctx = r.Context()

			owner = chi.URLParam(r, "owner")
			name  = chi.URLParam(r, "name")
		)
		
		// 先看看当前的用户有没有 admin 权限
		user, _   := request.UserFrom(ctx)

		if !user.Admin {
			logger.FromRequest(r).
				Warnln("api: user is not admin")
			return
		}
		
		// 获取项目信息
		repo, err := repos.FindName(ctx, owner, name)
		if err != nil {
			render.NotFound(w, render.ErrNotFound)
			logger.FromRequest(r).
				WithError(render.ErrNotFound).
				WithField("namespace", owner).
				WithField("name", name).
				Debugln("api: repository not found")
			return
		}

		// 找出所有的用户
		tms, err := findUsers(ctx, perms, users, repo.UID)
		if err != nil {
			render.InternalError(w, err)
			logger.FromRequest(r).
				WithError(err).
				WithField("namespace", owner).
				WithField("name", name).
				Errorln("api: find user error")
			return
		}

		// 选择一个合适的用户替换到context
		for _, u := range tms {
			if !u.Admin || !u.Active {
				continue
			}
			// 设置到context
			logger.FromRequest(r).
				WithField("namespace", owner).
				WithField("name", name).
				WithField("user", u.Login).
				Debugln("update user to access repo builds")
				
			r = r.WithContext(request.WithUser(ctx, u))
			break
		}
		

		next(w, r)
	}
}