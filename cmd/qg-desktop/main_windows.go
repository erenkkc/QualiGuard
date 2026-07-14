//go:build windows

package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/jchv/go-webview2"
	"github.com/qualiguard/qualiguard/internal/server"
	"github.com/qualiguard/qualiguard/internal/store"
)

func main() {
	base := appDir()
	dataDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "QualiGuard", "data")
	workDir := base
	configPath := filepath.Join(base, "qualiguard.yaml")
	if _, err := os.Stat(configPath); err != nil {
		configPath = filepath.Join(workDir, "qualiguard.yaml")
	}

	_ = os.MkdirAll(dataDir, 0o755)
	_ = os.MkdirAll(filepath.Join(base, "config"), 0o755)

	port, err := freePort()
	if err != nil {
		fatal("Port alınamadı", err)
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	baseURL := "http://" + addr

	dbPath := filepath.Join(dataDir, "qualiguard.db")
	st, err := store.Open(dbPath)
	if err != nil {
		fatal("Veritabanı açılamadı", err)
	}
	defer st.Close()

	token, err := st.EnsureDefaultToken(context.Background())
	if err != nil {
		fatal("Token oluşturulamadı", err)
	}
	_ = os.WriteFile(filepath.Join(dataDir, "token.txt"), []byte(token+"\n"), 0o600)

	httpSrv := &http.Server{
		Addr:    addr,
		Handler: server.New(st, token, workDir, configPath).Handler(),
	}

	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, "sunucu hatası:", err)
		}
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(ctx)
	}()

	if !waitReady(baseURL+"/api/health", 15*time.Second) {
		// Fallback: Edge app mode if WebView2 fails later; still need server ready
		_ = waitReady(baseURL+"/api/health", 5*time.Second)
	}

	startURL := baseURL + "/desktop"
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     false,
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  "QualiGuard",
			Width:  1280,
			Height: 820,
			Center: true,
		},
	})
	if w == nil {
		// WebView2 yoksa Edge uygulama penceresiyle aç
		if openEdgeApp(startURL) {
			waitProcess()
			return
		}
		fatal("WebView2 bulunamadı", fmt.Errorf("Microsoft Edge WebView2 Runtime gerekli"))
	}
	defer w.Destroy()
	w.SetSize(1280, 820, webview2.HintNone)
	w.Navigate(startURL)
	w.Run()
}

func appDir() string {
	exe, err := os.Executable()
	if err != nil {
		wd, _ := os.Getwd()
		return wd
	}
	return filepath.Dir(exe)
}

func freePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func waitReady(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return true
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

func openEdgeApp(url string) bool {
	candidates := []string{
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
		"msedge",
	}
	for _, edge := range candidates {
		cmd := exec.Command(edge, "--app="+url, "--new-window", "--window-size=1280,820")
		if err := cmd.Start(); err == nil {
			return true
		}
	}
	return false
}

func waitProcess() {
	// Edge app mode: keep helper alive while user may close quickly;
	// block until Ctrl+C is impractical in GUI — sleep forever lightly.
	select {}
}

func fatal(msg string, err error) {
	ps := fmt.Sprintf(
		`Add-Type -AssemblyName PresentationFramework; [System.Windows.MessageBox]::Show(%q, 'QualiGuard', 'OK', 'Error')`,
		fmt.Sprintf("%s\n%s", msg, err),
	)
	_ = exec.Command("powershell", "-NoProfile", "-Command", ps).Run()
	os.Exit(1)
}
