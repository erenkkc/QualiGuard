package server

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
)

type downloadItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Platform    string `json:"platform"`
	Filename    string `json:"filename"`
}

func (s *Server) downloadsDir() string {
	if s.workDir == "" {
		return "releases"
	}
	return filepath.Join(s.workDir, "releases")
}

func (s *Server) catalogDownloads() []downloadItem {
	return []downloadItem{
		{
			ID:          "windows-panel",
			Title:       "Windows — QualiGuard uygulaması",
			Description: "İndir, kur — Başlat menüsünde gerçek bir uygulama olarak açılır (splash + giriş).",
			Platform:    "windows",
			Filename:    "QualiGuard-Kurulum.exe",
		},
		{
			ID:          "mac-panel",
			Title:       "Mac — QualiGuard uygulaması",
			Description: "Zip'i aç, QualiGuard-Kur.command ile başlat — splash ve giriş ile açılır.",
			Platform:    "mac",
			Filename:    "qualiguard-mac-kurulum.zip",
		},
		{
			ID:          "windows-cli",
			Title:       "Windows — CLI (qg.exe)",
			Description: "Komut satırından kod tarayın.",
			Platform:    "windows",
			Filename:    "qg-windows-amd64.exe",
		},
		{
			ID:          "mac-cli",
			Title:       "Mac — CLI",
			Description: "Terminalden kod tarayın.",
			Platform:    "mac",
			Filename:    "qg-darwin-amd64",
		},
	}
}

func (s *Server) handleDownloadsList(w http.ResponseWriter, r *http.Request) {
	dir := s.downloadsDir()
	items := s.catalogDownloads()
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		path := filepath.Join(dir, it.Filename)
		info, err := os.Stat(path)
		available := err == nil && !info.IsDir()
		size := int64(0)
		if available {
			size = info.Size()
		}
		out = append(out, map[string]any{
			"id":          it.ID,
			"title":       it.Title,
			"description": it.Description,
			"platform":    it.Platform,
			"filename":    it.Filename,
			"available":   available,
			"size_bytes":  size,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (s *Server) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var item *downloadItem
	for i := range s.catalogDownloads() {
		it := s.catalogDownloads()[i]
		if it.ID == id {
			cp := it
			item = &cp
			break
		}
	}
	if item == nil {
		writeError(w, http.StatusNotFound, "unknown download")
		return
	}
	path := filepath.Join(s.downloadsDir(), item.Filename)
	if _, err := os.Stat(path); err != nil {
		writeError(w, http.StatusNotFound, "dosya henüz hazır değil — yönetici release paketlerini üretmeli")
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename=\""+item.Filename+"\"")
	http.ServeFile(w, r, path)
}
