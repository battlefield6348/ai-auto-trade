package httpapi

import (
	"encoding/json"
	"net/http"
)

type errorResponse struct {
	Success   bool   `json:"success"`
	Error     string `json:"error"`
	ErrorCode string `json:"error_code"`
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, errorResponse{
		Success:   false,
		Error:     msg,
		ErrorCode: code,
	})
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
