package server

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

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
		"user_auth":     true,
	})
}

// handlePanelLogin kept for old clients that only send {"password":"..."}
func (s *Server) handlePanelLogin(w http.ResponseWriter, r *http.Request) {
	s.handleUserLogin(w, r)
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	user, err := s.store.CreateUser(r.Context(), body.Email, body.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sess, err := s.store.CreateSession(r.Context(), user.ID, 30*24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"token": sess.Token,
		"user":  map[string]string{"id": user.ID, "email": user.Email},
		"ai":    ai.AssistantStatus(),
	})
}

func (s *Server) handleUserLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	email := strings.TrimSpace(body.Email)
	if email == "" {
		if s.panelPassword == "" {
			writeError(w, http.StatusBadRequest, "email required")
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
		return
	}

	user, err := s.store.AuthenticateUser(r.Context(), email, body.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	sess, err := s.store.CreateSession(r.Context(), user.ID, 30*24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token": sess.Token,
		"user":  map[string]string{"id": user.ID, "email": user.Email},
		"ai":    ai.AssistantStatus(),
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token != "" {
		_ = s.store.DeleteSession(r.Context(), token)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	user, err := s.store.GetUserBySessionToken(r.Context(), token)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if user != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"user": map[string]string{"id": user.ID, "email": user.Email},
		})
		return
	}
	if s.store.ValidateToken(r.Context(), token) {
		writeJSON(w, http.StatusOK, map[string]any{
			"user": map[string]string{"email": "admin", "role": "panel"},
		})
		return
	}
	writeError(w, http.StatusUnauthorized, "not logged in")
}

func bearerToken(r *http.Request) string {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	token = strings.TrimSpace(token)
	if token == "" {
		token = r.Header.Get("X-QualiGuard-Token")
	}
	return token
}
