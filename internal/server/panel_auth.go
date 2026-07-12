package server

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/qualiguard/qualiguard/internal/ai"
)

func panelPasswordFromEnv() string {
	return strings.TrimSpace(os.Getenv("QG_PANEL_PASSWORD"))
}

func (s *Server) panelAuthRequired() bool {
	return s.panelPassword != ""
}

func (s *Server) handlePublicConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"brand":         s.brand,
		"auth_required": s.panelAuthRequired(),
	})
}

func (s *Server) handlePanelLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	if !s.panelAuthRequired() {
		writeError(w, http.StatusForbidden, "panel login disabled on this server")
		return
	}

	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if subtle.ConstantTimeCompare([]byte(body.Password), []byte(s.panelPassword)) != 1 {
		writeError(w, http.StatusUnauthorized, "wrong password")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token": s.token,
		"ai":    ai.AssistantStatus(),
	})
}
