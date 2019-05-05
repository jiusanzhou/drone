package handler

import (
	"errors"

	"github.com/go-chi/chi"
)

var (
	// ErrPatternConflict returns while pattern exsits
	ErrPatternConflict = errors.New("pattern conflict")

	// apirouters = make(map[string]func(r chi.Router))
	// webrouters = make(map[string]func(r chi.Router))
	routers = []map[string]func(r chi.Router){
		make(map[string]func(r chi.Router)),
		make(map[string]func(r chi.Router)),
	}

	// API type
	API = 0

	// WEB type
	WEB = 1
)

// Register ...
func Register(flag int, pattern string, fn func(r chi.Router)) error {
	if _, ok := routers[flag][pattern]; ok {
		return ErrPatternConflict
	}
	routers[flag][pattern] = fn
	return nil
}

// Routers ...
func Routers(flag int) map[string]func(r chi.Router) {
	return routers[flag]
}