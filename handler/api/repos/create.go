package repos

import (
	"net/http"

	"github.com/drone/drone/core"
	"github.com/drone/drone/handler/api/render"
	"github.com/drone/drone/handler/api/request"
	"github.com/drone/drone/logger"

	"github.com/go-chi/chi"
)

// HandleCreate returns an http.HandlerFunc that processes http
// requests to create a repository to the currently authenticated user.
func HandleCreate(repos core.RepositoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			owner = chi.URLParam(r, "owner")
			name  = chi.URLParam(r, "name")
			slug = owner + "/" + name
		)

		_, err := repos.FindName(r.Context(), owner, name)
		if err == nil {
			render.Conflict(w, render.ErrConflict)
			logger.FromRequest(r).
				WithError(render.ErrConflict).
				WithField("namespace", owner).
				WithField("name", name).
				Debugln("api: repository exsits")
			return
		}

		user, _ := request.UserFrom(r.Context())

		repo := new(core.Repository)

		err = json.NewDecoder(r.Body).Decode(repo)
		if err != nil {
			render.BadRequest(w, err)
			logger.FromRequest(r).
				WithError(err).
				WithField("repository", slug).
				Debugln("api: cannot unmarshal json input")
			return
		}

		repo.UserID = user.ID

		err = repos.Create(r.Context(), repo)
		if err != nil {
			render.InternalError(w, err)
			logger.FromRequest(r).
				WithError(err).
				WithField("repository", slug).
				Warnln("api: cannot update repository")
			return
		}

		render.JSON(w, repo, 200)
	}
}
