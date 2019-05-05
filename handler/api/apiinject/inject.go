package apiinject

import (
	"github.com/go-chi/chi"

	"github.com/drone/drone/core"
	"github.com/drone/drone/handler/api/acl"
	repospkgo "github.com/drone/drone/handler/api/repos"

	repospkg "github.com/drone/drone/handler/api/apiinject/repos"
	userspkg "github.com/drone/drone/handler/api/apiinject/users"
)

// Create ...
func Create(
	repoz core.RepositoryService,

	users core.UserStore,
	perms core.PermStore,
	repos core.RepositoryStore,
) func(r chi.Router) {
	return func (r chi.Router) {
		r.Route("/repos/{owner}/{name}", func(r chi.Router) {
			
			r.With(
				acl.InjectRepository(repoz, repos, perms),
				acl.CheckReadAccess(),
			).Get("/", repospkgo.HandleFind()) // 获取 repo 信息

			r.With(
				acl.CheckAdminAccess(),
			).Post("/", repospkg.HandleCreate(repos, perms)) // 创建 repo

			r.With(
				acl.InjectRepository(repoz, repos, perms),
				acl.CheckAdminAccess(),
			).Delete("/", repospkg.HandleDelete(repos, perms)) // 删除 repo

			r.With(
				acl.InjectRepository(repoz, repos, perms),
				acl.CheckWriteAccess(),
			).Patch("/", repospkg.HandleUpdate(repos, perms)) // 更新 repo
		})

		r.Route("/users", func(r chi.Router) {

			r.With(
				
			).Post("/", userspkg.HandleCreate(users)) // 添加用户

			r.With(
				// 管理员用户可以删除任意账户,当前用户可以删除自身
			).Delete("/:username", userspkg.HandleDelete(users)) // 删除用户

			r.With(
				// 管理员用户可以更新任意账户,当前用户可以更新自身
			).Patch("/:username", userspkg.HandleUpdate(users))
		})
	}
}