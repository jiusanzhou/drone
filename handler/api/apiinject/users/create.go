package users

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"errors"
	"time"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/drone/drone/core"
	"github.com/drone/drone/handler/api/request"
	"github.com/drone/drone/handler/api/render"
	"github.com/drone/drone/logger"
	"github.com/drone/go-login/login"

	"github.com/dchest/uniuri"

	"github.com/go-chi/chi"
)

// period at which the user account is synchronized
// with the remote system. Default is weekly.
var syncPeriod = time.Hour * 24 * 7

// period at which the sync should timeout
var syncTimeout = time.Minute * 30

// UserCreate stores account information used to bootstrap
// the admin user account when the system initializes.
type userCreateInput struct {
	core.User
	Username string `json:"username"`
	Token    string `json:"token"`
}

type userUpdateInput struct {
	Username  *string `json:"username"`
	Email     *string `json:"email"`
	Avatar    *string `json:"avatar"`
	Token     *string `json:"token"`
}

// HandleCreate ...
func HandleCreate(
	users core.UserStore,
	userz core.UserService,
	admission core.AdmissionService,
	syncer core.Syncer,

	sender core.WebhookSender,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 通过接口新增用户

		// 在外面通过插件先判断了当前用户是否具备管理员去权限
		tok := new(login.Token)

		err := json.NewDecoder(r.Body).Decode(tok)
		if err != nil {
			render.BadRequest(w, err)
			logger.FromRequest(r).
				WithError(err).
				WithField("action", "create user").
				Debugln("api: cannot unmarshal json input")
			return
		}

		ctx := r.Context()

		// 首先查看该用户是否在远程仓库中存在
		account, err := userz.Find(ctx, tok.Access, tok.Refresh)
		if err != nil {
			render.BadRequest(w, err)
			logger.FromRequest(r).
				WithError(err).
				WithField("action", "create user").
				Debugf("cannot find remote user.")
			return
		}

		// 先查下是否存在用户
		user, err := users.FindLogin(ctx, account.Login)
		if err == sql.ErrNoRows {
			user = &core.User{
				Login:     account.Login,
				Email:     account.Email,
				Avatar:    account.Avatar,
				Admin:     false,
				Machine:   false,
				Active:    true,
				Syncing:   true,
				Synced:    0,
				LastLogin: time.Now().Unix(),
				Created:   time.Now().Unix(),
				Updated:   time.Now().Unix(),
				Token:     tok.Access,
				Refresh:   tok.Refresh,
				Hash:      uniuri.NewLen(32),
			}
			if !tok.Expires.IsZero() {
				user.Expiry = tok.Expires.Unix()
			}

			err = admission.Admit(ctx, user)
			if err != nil {
				render.BadRequest(w, err)
				logger.FromRequest(r).
					WithError(err).
					WithField("action", "create user").
					Errorf("cannot admit user.")
				return
			}

			err = users.Create(ctx, user)
			if err != nil {
				render.BadRequest(w, err)
				logger.FromRequest(r).
					WithError(err).
					WithField("action", "create user").
					Errorf("cannot create user.")
				return
			}

			err = sender.Send(ctx, &core.WebhookData{
				Event:  core.WebhookEventUser,
				Action: core.WebhookActionCreated,
				User:   user,
			})
			if err != nil {
				render.BadRequest(w, err)
				logger.FromRequest(r).
					WithError(err).
					WithField("action", "create user").
					Errorf("cannot send webhook.")
			} else {
				logger.FromRequest(r).
					WithError(err).
					WithField("action", "create user").
					Debugf("successfully created user.")
			}
		} else if err != nil {
			render.BadRequest(w, err)
			logger.FromRequest(r).
				WithError(err).
				WithField("action", "create user").
			 	Errorf("cannot find user: %s", err)
			return
		}

		// 然后创建并同步所有的项目

		user.Avatar = account.Avatar
		user.Email = account.Email
		user.Token = tok.Access
		user.Refresh = tok.Refresh
		user.LastLogin = time.Now().Unix()
		if !tok.Expires.IsZero() {
			user.Expiry = tok.Expires.Unix()
		}

		// If the user account has never been synchronized we
		// execute the synchonrization logic.
		if time.Unix(user.Synced, 0).Add(syncPeriod).Before(time.Now()) {
			user.Syncing = true
		}

		err = users.Update(ctx, user)
		if err != nil {
			// if the account update fails we should still
			// proceed to create the user session. This is
			// considered a non-fatal error.
			logger.FromRequest(r).
				WithError(err).
				Errorf("cannot update user.")
		}

		// launch the synchrnoization process in a go-routine,
		// since it is a long-running process and can take up
		// to a few minutes.
		if user.Syncing {
			go synchornize(ctx, syncer, user)
		}

		logger.FromRequest(r).
			WithError(err).
			Debugf("authentication successful")

		render.JSON(w, user, 200)
	}
}

// HandleDelete ...
func HandleDelete(users core.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		
		user, _ := request.UserFrom(r.Context()) // 获取当前用户

		if user.Admin {

		}

		// TODO:

		render.InternalErrorf(w, "用户删除接口未实现.")
	}
}

// HandleUpdate ...
func HandleUpdate(users core.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// TODO: 管理员
		var (
			username = chi.URLParam(r, "username")
		)

		user, _ := request.UserFrom(r.Context()) // 获取当前用户

		var err error

		if user.Login != username {
			if user.Admin {
				err = errors.New("not implement")
			} else {
				err = errors.New("cannot update account of other")
			}

			render.BadRequest(w, err)
			logger.FromRequest(r).
				WithError(err).
				WithField("username", username).
				Debugln("api: connot update user account")
			return
		}

		// 读取新的用户信息
		uui := new(userUpdateInput)
		
		// 更新用户信息
		if uui.Username != nil {
			user.Login = *(uui.Username)
		}
		if uui.Email != nil {
			user.Email = *(uui.Email)
		}
		if uui.Avatar != nil {
			user.Avatar = *(uui.Avatar)
		}
		if uui.Token != nil {
			user.Token = *(uui.Token)
			user.Hash = *(uui.Token)
		}

		err = users.Update(r.Context(), user)
		if err != nil {
			render.InternalError(w, err)
			logger.FromRequest(r).
				WithError(err).
				WithField("username", username).
				Warnln("api: cannot update user account")
			return
		}

		render.JSON(w, user, 200)

		return
	}
}

func synchornize(ctx context.Context, syncer core.Syncer, user *core.User) {
	log := logrus.WithField("login", user.Login)
	log.Debugf("begin synchronization")

	timeout, cancel := context.WithTimeout(context.Background(), syncTimeout)
	timeout = logger.WithContext(timeout, log)
	defer cancel()
	_, err := syncer.Sync(timeout, user)
	if err != nil {
		log.Debugf("synchronization failed: %s", err)
	} else {
		log.Debugf("synchronization success")
	}
}