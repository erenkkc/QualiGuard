//go:build windows

package main

import (
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

//go:embed assets/qg-server.exe
var serverBin []byte

//go:embed assets/qualiguard.yaml
var configYaml []byte

func main() {
	fmt.Println()
	fmt.Println("  ==================================")
	fmt.Println("   QualiGuard Kurulum")
	fmt.Println("  ==================================")
	fmt.Println()

	installDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "QualiGuard")
	fmt.Println("  Kurulum klasoru:", installDir)

	if err := os.MkdirAll(installDir, 0o755); err != nil {
		fail(err)
	}

	serverPath := filepath.Join(installDir, "qg-server.exe")
	configPath := filepath.Join(installDir, "qualiguard.yaml")
	launcherPath := filepath.Join(installDir, "QualiGuard.bat")

	fmt.Println("  Dosyalar yaziliyor...")
	if err := os.WriteFile(serverPath, serverBin, 0o755); err != nil {
		fail(err)
	}
	if err := os.WriteFile(configPath, configYaml, 0o644); err != nil {
		fail(err)
	}

	launcher := "@echo off\r\n" +
		"title QualiGuard\r\n" +
		"cd /d \"%~dp0\"\r\n" +
		"start \"\" http://127.0.0.1:9000/app\r\n" +
		"qg-server.exe --host 127.0.0.1 --port 9000 --data-dir \"%USERPROFILE%\\.qualiguard-local\" --work-dir \"%~dp0\" --config qualiguard.yaml\r\n"
	if err := os.WriteFile(launcherPath, []byte(launcher), 0o644); err != nil {
		fail(err)
	}

	fmt.Println("  Masaustu kisayolu olusturuluyor...")
	createDesktopShortcut(launcherPath, installDir)

	fmt.Println("  Panel baslatiliyor...")
	if err := startServer(serverPath, installDir); err != nil {
		fail(err)
	}

	if waitReady("http://127.0.0.1:9000/app", 20*time.Second) {
		fmt.Println("  Panel hazir — tarayici aciliyor...")
		openBrowser("http://127.0.0.1:9000/app")
	} else {
		fmt.Println("  Uyari: panel henuz yanit vermedi. Masaustundeki QualiGuard kisayolunu kullan.")
		openBrowser("http://127.0.0.1:9000/app")
	}

	fmt.Println()
	fmt.Println("  Kurulum tamam!")
	fmt.Println("  Panel: http://127.0.0.1:9000/app")
	fmt.Println("  Sonraki sefer: masaustundeki QualiGuard kisayolu")
	fmt.Println()
	fmt.Println("  Bu pencereyi kapatabilirsiniz.")
	fmt.Println()
	fmt.Print("  Kapatmak icin Enter...")
	fmt.Scanln()
}

func createDesktopShortcut(target, workDir string) {
	desktop := filepath.Join(os.Getenv("USERPROFILE"), "Desktop", "QualiGuard.lnk")
	ps := fmt.Sprintf(
		"$ws = New-Object -ComObject WScript.Shell; $s = $ws.CreateShortcut(%q); $s.TargetPath = %q; $s.WorkingDirectory = %q; $s.Description = 'QualiGuard Panel'; $s.Save()",
		desktop, target, workDir,
	)
	_ = exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", ps).Run()
}

func startServer(serverPath, workDir string) error {
	cmd := exec.Command(serverPath,
		"--host", "127.0.0.1",
		"--port", "9000",
		"--data-dir", filepath.Join(os.Getenv("USERPROFILE"), ".qualiguard-local"),
		"--work-dir", workDir,
		"--config", "qualiguard.yaml",
	)
	cmd.Dir = workDir
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	return cmd.Start()
}

func waitReady(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return true
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

func openBrowser(url string) {
	_ = exec.Command("cmd", "/c", "start", "", url).Start()
}

func fail(err error) {
	fmt.Println()
	fmt.Println("  Kurulum hatasi:", err)
	fmt.Println()
	fmt.Print("  Enter'a basin...")
	fmt.Scanln()
	os.Exit(1)
}
