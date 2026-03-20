package miniapp

import (
	"encoding/json"
	"io"
	"net/http"
)

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func decodeJSONBody(r *http.Request, limit int64, dst any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, limit))
	return dec.Decode(dst)
}
