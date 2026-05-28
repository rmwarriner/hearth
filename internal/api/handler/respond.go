package handler

import (
	"encoding/json"
	"net/http"

	"github.com/hearth-ledger/hearth/internal/api/middleware"
	hearth "github.com/hearth-ledger/hearth/pkg/errors"
)

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		_ = err // status 200 already implicit; nothing more to do
	}
}

func jsonCreated(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		_ = err
	}
}

func jsonNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func jsonError(w http.ResponseWriter, err error) {
	var he *hearth.HearthError
	if ok := errAs(err, &he); ok {
		status := middleware.ErrorCodeToStatus(he.Code)
		writeError(w, status, string(he.Code), he.Message)
		return
	}
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	type body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body{Code: code, Message: message}); err != nil {
		_ = err
	}
}

func errAs(err error, target **hearth.HearthError) bool {
	if he, ok := err.(*hearth.HearthError); ok {
		*target = he
		return true
	}
	return false
}
