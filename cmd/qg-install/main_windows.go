//go:build windows

package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:embed assets/QualiGuard.exe
var appBin []byte

//go:embed assets/qualiguard.yaml
var configYaml []byte

func main() {
	fmt.Println()
	fmt.Println("  ==================================")
	fmt.Println("   QualiGuard Kurulum")
	fmt.Println("  ==================================")
	fmt.Println()

	installDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "QualiGuard")
	fmt.Println("  Kurulum:", installDir)

	if err := os.MkdirAll(installDir, 0o755); err != nil {
		fail(err)
	}

	appPath := filepath.Join(installDir, "QualiGuard.exe")
	configPath := filepath.Join(installDir, "qualiguard.yaml")

	fmt.Println("  Uygulama dosyalari yaziliyor...")
	if err := os.WriteFile(appPath, appBin, 0o755); err != nil {
		fail(err)
	}
	if err := os.WriteFile(configPath, configYaml, 0o644); err != nil {
		fail(err)
	}

	fmt.Println("  Baslat menusune ekleniyor...")
	createShortcut(
		filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "QualiGuard.lnk"),
		appPath, installDir,
	)
	fmt.Println("  Masaustu kisayolu olusturuluyor...")
	createShortcut(
		filepath.Join(os.Getenv("USERPROFILE"), "Desktop", "QualiGuard.lnk"),
		appPath, installDir,
	)

	fmt.Println("  QualiGuard baslatiliyor...")
	cmd := exec.Command(appPath)
	cmd.Dir = installDir
	if err := cmd.Start(); err != nil {
		fail(err)
	}

	fmt.Println()
	fmt.Println("  Kurulum tamam!")
	fmt.Println("  QualiGuard artik bilgisayarinda bir uygulama olarak acildi.")
	fmt.Println("  Sonraki sefer: Baslat menusu veya masaustundeki QualiGuard")
	fmt.Println()
	time.Sleep(2 * time.Second)
}

func createShortcut(lnk, target, workDir string) {
	_ = os.MkdirAll(filepath.Dir(lnk), 0o755)
	ps := fmt.Sprintf(
		"$ws = New-Object -ComObject WScript.Shell; $s = $ws.CreateShortcut(%q); $s.TargetPath = %q; $s.WorkingDirectory = %q; $s.Description = 'QualiGuard'; $s.Save()",
		lnk, target, workDir,
	)
	_ = exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", ps).Run()
}

func fail(err error) {
	msg := strings.ReplaceAll(err.Error(), "'", "''")
	ps := fmt.Sprintf(
		`Add-Type -AssemblyName PresentationFramework; [System.Windows.MessageBox]::Show('Kurulum hatasi: %s', 'QualiGuard', 'OK', 'Error')`,
		msg,
	)
	_ = exec.Command("powershell", "-NoProfile", "-Command", ps).Run()
	fmt.Println("  Kurulum hatasi:", err)
	fmt.Print("  Enter...")
	fmt.Scanln()
	os.Exit(1)
}
