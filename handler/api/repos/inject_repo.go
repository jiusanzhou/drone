package repos

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/drone/drone/core"
	"github.com/drone/drone/handler/api/render"
	"github.com/drone/drone/handler/api/request"
	"github.com/drone/drone/logger"

	"github.com/go-chi/chi"
)

type (
	inputCreate struct {
		ID        string    `json:"id"`
		Namespace string    `json:"namespace"`
		Name      string    `json:"name"`
		Branch    string    `json:"default_branch"`
		SCM       string    `json:"scm"`
		Private   bool      `json:"private"`
		Clone     string    `json:"git_http_url"`
		CloneSSH  string    `json:"git_ssh_url"`
		Link      string    `json:"link"`
	}
)

// HandleRepoCreate returns an http.HandlerFunc that processes http
// requests to create a repository to the currently authenticated user.
func HandleRepoCreate(repos core.RepositoryStore, perms core.PermStore) http.HandlerFunc {
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

		src := new(inputCreate)

		err = json.NewDecoder(r.Body).Decode(src)
		if err != nil {
			render.BadRequest(w, err)
			logger.FromRequest(r).
				WithError(err).
				WithField("repository", slug).
				Debugln("api: cannot unmarshal json input")
			return
		}

		repo := &core.Repository{
			UID:        src.ID,
			// Namespace:  src.Namespace, // Use params from url
			// Name:       src.Name, // Use params from url
			Slug:       slug,
			HTTPURL:    src.Clone,
			SSHURL:     src.CloneSSH,
			Link:       src.Link,
			Private:    src.Private,
			Branch:     src.Branch,

			UserID:     user.ID,
			Namespace:  owner,
			Name:       name,

			Created: time.Now().Unix(),
		}

		if repo.Private {
			repo.Visibility = core.VisibilityPrivate
		} else {
			repo.Visibility = core.VisibilityPublic
		}

		// disable other fields

		if repo.Branch == "" {
			repo.Branch = "master"
		}

		if repo.Config == "" {
			repo.Config = ".drone.yml"
		}

		if repo.Timeout == 0 {
			repo.Timeout = 60
		}

		now := time.Now().Unix()
		// create perm
		perm := &core.Perm{
			UserID: user.ID,
			RepoUID: repo.UID,
			Read: true,
			Write: true,
			Admin: true,
			Synced: time.Now().AddDate(1000, 0, 0).Unix(), // make sure we never(1000 year) sync the perm
			Created: now,
			Updated: now,
		}
		err = perms.Update(r.Context(), perm)
		if err != nil {
			render.InternalError(w, err)
			logger.FromRequest(r).
				WithError(err).
				WithField("repository", slug).
				Warnln("api: cannot cache repository permissions")
			return
		}

		err = repos.Create(r.Context(), repo)
		if err != nil {
			render.InternalError(w, err)
			logger.FromRequest(r).
				WithError(err).
				WithField("repository", slug).
				Warnln("api: cannot create repository")
			return
		}

		render.JSON(w, repo, 200)
	}
}

// HandleRepoUpdate ...
func HandleRepoUpdate(repos core.RepositoryStore, perms core.PermStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			owner = chi.URLParam(r, "owner")
			name  = chi.URLParam(r, "name")
			slug = owner + "/" + name
		)

		oldrepo, err := repos.FindName(r.Context(), owner, name)
		if err != nil {
			render.NotFound(w, err)
			logger.FromRequest(r).
				WithError(err).
				WithField("namespace", owner).
				WithField("name", name).
				Debugln("api: repository not found")
			return
		}

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

		// check and purge update
		repo.ID = oldrepo.ID
		repo.Updated = time.Now().Unix()
		// TODO: use oldrepo as default value of repo

		err = repos.Update(r.Context(), repo)
		if err != nil {
			render.InternalError(w, err)
			logger.FromRequest(r).
				WithError(err).
				WithField("repository", slug).
				Warnln("api: cannot update repository")
			return
		}

		render.JSON(w, repo, 200)

		return 
	}
}

// HandleRepoDelete ...
func HandleRepoDelete(repos core.RepositoryStore, perms core.PermStore) http.HandlerFunc {
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
