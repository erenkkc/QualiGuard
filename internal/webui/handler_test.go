package webui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/qualiguard/qualiguard/internal/config"
)

func TestHandlerRoutes(t *testing.T) {
	brand := config.BrandConfig{
		Name:    "CodeShield",
		Tagline: "Güvenli kod",
		Accent:  "#ff6600",
		Accent2: "#cc3300",
	}
	h := Handler(PanelAuth{InjectToken: true, Token: "qg_test_token"}, brand)

	t.Run("landing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "CodeShield") {
			t.Fatalf("expected brand name on landing")
		}
		if strings.Contains(body, "__QG_") {
			t.Fatalf("placeholder leaked on landing")
		}
	})

	t.Run("login", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/login", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "Giriş yap") {
			t.Fatalf("expected login page")
		}
	})

	t.Run("app panel", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/app", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		body := rec.Body.String()
		if !strings.Contains(body, "qg_test_token") {
			t.Fatalf("expected injected token")
		}
		if !strings.Contains(body, "CodeShield") {
			t.Fatalf("expected brand in panel")
		}
	})

	t.Run("app without token injection", func(t *testing.T) {
		h2 := Handler(PanelAuth{InjectToken: false, Token: "secret"}, brand)
		rec := httptest.NewRecorder()
		h2.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app", nil))
		body := rec.Body.String()
		if strings.Contains(body, "secret") {
			t.Fatalf("token must not be injected when auth required")
		}
		if !strings.Contains(body, "__QG_TOKEN__") {
			t.Fatalf("expected empty token placeholder")
		}
	})
}
