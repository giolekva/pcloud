package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/giolekva/pcloud/core/kg/model"
)

type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// ServeHTTP calls f(w, r) and handles error
func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := f(w, r); err != nil {
		jsonError(w, err)
	}
}

func jsoner(w http.ResponseWriter, statusCode int, payload interface{}) error {
	w.Header().Set("Content-Type", "application/json")

	// If there is nothing to marshal then set status code and return.
	if payload == nil {
		_, err := w.Write([]byte("{}"))
		return err
	}

	w.WriteHeader(statusCode)

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(true)
	encoder.SetIndent("", "")

	if err := encoder.Encode(payload); err != nil {
		return err
	}

	return nil
}

// TODO test error statuses
func jsonError(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, model.ErrForbidden):
		code = http.StatusForbidden
	case errors.Is(err, model.ErrInvalidInput):
		code = http.StatusBadRequest
	case errors.Is(err, model.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, model.ErrUnauthorized):
		code = http.StatusUnauthorized
	}
	jsoner(w, code, err.Error())
}
