package repos

import (
	"net/http"

	"github.com/drone/drone/core"
	"github.com/drone/drone/handler/api/render"
	"github.com/drone/drone/logger"

	"github.com/go-chi/chi"
)


// HandleDelete ...
func HandleDelete(repos core.RepositoryStore, perms core.PermStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			owner = chi.URLParam(r, "owner")
			name  = chi.URLParam(r, "name")
			slug = owner + "/" + name
		)

		repo, err := repos.FindName(r.Context(), owner, name)
		if err != nil {
			render.NotFound(w, err)
			logger.FromRequest(r).
				WithError(err).
				WithField("namespace", owner).
				WithField("name", name).
				Debugln("api: repository not found")
			return
		}

		// else get the cached permissions from the database
		// for the user and repository.
		perm, err := perms.Find(r.Context(), repo.UID, repo.UserID)
		if err == nil {
			perms.Delete(r.Context(), perm)
		}

		err = repos.Delete(r.Context(), repo)
		if err != nil {
			render.InternalError(w, err)
			logger.FromRequest(r).
				WithError(err).
				WithField("repository", slug).
				Warnln("api: cannot delete repository")
		}

		render.JSON(w, nil, 200)

		return 
	}
}
