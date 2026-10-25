package web

import (
	"encoding/json"
	"net/http"

	"databin/internal/paste"
)

type createResponse struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

func writeCreateResponse(w http.ResponseWriter, created paste.Created) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(createResponse{
		ID:    created.ID,
		Token: created.Token,
	})
}
