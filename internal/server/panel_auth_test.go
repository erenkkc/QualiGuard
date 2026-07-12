package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qualiguard/qualiguard/internal/config"
)

func TestPanelLogin(t *testing.T) {
	t.Setenv("QG_PANEL_PASSWORD", "test-pass")

	s := &Server{
		token:         "qg_test",
		panelPassword: panelPasswordFromEnv(),
		brand:         config.DefaultBrand(),
	}

	t.Run("wrong password", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"password": "wrong"})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		s.handlePanelLogin(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("ok", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"password": "test-pass"})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		s.handlePanelLogin(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
		}
		var out map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		if out["token"] != "qg_test" {
			t.Fatalf("token = %v", out["token"])
		}
	})
}

func TestPublicConfigAuthFlag(t *testing.T) {
	s := &Server{brand: config.DefaultBrand(), panelPassword: "x"}
	rec := httptest.NewRecorder()
	s.handlePublicConfig(rec, httptest.NewRequest(http.MethodGet, "/api/public/config", nil))
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["auth_required"] != true {
		t.Fatalf("auth_required = %v", out["auth_required"])
	}
}
