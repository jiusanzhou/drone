package users

import (
	"encoding/json"
	"net/http"
	"errors"
	"time"

	// "github.com/dchest/uniuri"

	"github.com/drone/drone/core"
	"github.com/drone/drone/handler/api/request"
	"github.com/drone/drone/handler/api/render"
	"github.com/drone/drone/logger"

	"github.com/go-chi/chi"
)

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
func HandleCreate(users core.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 通过接口新增用户

		// 在外面通过插件先判断了当前用户是否具备管理员去权限

		uci := new(userCreateInput)

		err := json.NewDecoder(r.Body).Decode(uci)
		if err != nil {
			render.BadRequest(w, err)
			logger.FromRequest(r).
				WithError(err).
				WithField("action", "create user").
				Debugln("api: cannot unmarshal json input")
			return
		}

		// 首先查看该用户是否在远程仓库中存在

		// 先查下是否存在用户
		// 然后创建并同步所有的项目

		now := time.Now().Unix()
		user := &core.User{
			Login: uci.Username,
			Email: uci.Email,
			Avatar: uci.Avatar,
			
			Active: true,

			Created: now,
			Updated: now,

			Token: uci.Token, //
			Hash: uci.Token, // uniuri.NewLen(32),
		}

		err = users.Create(r.Context(), user)
		if err != nil {
			render.BadRequest(w, err)
			logger.FromRequest(r).
				WithError(err).
				WithField("username", uci.Username).
				Debugln("api: connot create user account")
			return
		}

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