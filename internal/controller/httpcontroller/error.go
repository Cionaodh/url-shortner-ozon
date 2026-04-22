package httpcontroller

import (
	"encoding/json"
	"net/http"
)

type ErrorResp struct {
	Error string `json:"error"`
}

func errorResponse(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResp{Error: msg})
}
