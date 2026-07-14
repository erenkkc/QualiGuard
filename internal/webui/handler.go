package webui

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/qualiguard/qualiguard/internal/config"
)

type PanelAuth struct {
	InjectToken bool
	Token       string
}

func Handler(auth PanelAuth, brand config.BrandConfig) http.Handler {
	sub, err := fs.Sub(Static, "static")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))

	token := ""
	if auth.InjectToken {
		token = auth.Token
	}
	replacements := brand.WithDefaults().HTMLReplacements(token)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		switch {
		case path == "/", path == "/index.html":
			serveStaticHTML(w, "static/landing.html", replacementsWithoutToken(replacements))
			return

		case path == "/login", path == "/login/":
			serveStaticHTML(w, "static/login.html", replacementsWithoutToken(replacements))
			return

		case path == "/register", path == "/register/":
			serveStaticHTML(w, "static/register.html", replacementsWithoutToken(replacements))
			return

		case path == "/indir", path == "/indir/", path == "/download", path == "/download/":
			serveStaticHTML(w, "static/indir.html", replacementsWithoutToken(replacements))
			return

		case path == "/app", path == "/app/", path == "/app/index.html":
			serveStaticHTML(w, "static/index.html", replacements)
			return

		case path == "/dashboard":
			http.Redirect(w, r, "/app", http.StatusFound)
			return
		}

		if strings.HasPrefix(path, "/assets/") {
			r.URL.Path = strings.TrimPrefix(path, "/assets/")
			fileServer.ServeHTTP(w, r)
			return
		}

		http.NotFound(w, r)
	})
}

func replacementsWithoutToken(reps map[string]string) map[string]string {
	out := make(map[string]string, len(reps))
	for k, v := range reps {
		if k == "__QG_TOKEN__" {
			continue
		}
		out[k] = v
	}
	return out
}

func serveStaticHTML(w http.ResponseWriter, name string, replacements map[string]string) {
	data, err := Static.ReadFile(name)
	if err != nil {
		http.Error(w, "page not found", http.StatusInternalServerError)
		return
	}
	html := string(data)
	for key, value := range replacements {
		html = strings.ReplaceAll(html, key, value)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}
