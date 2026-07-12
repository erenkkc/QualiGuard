package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/qualiguard/qualiguard/internal/server"
	"github.com/qualiguard/qualiguard/internal/store"
	"github.com/spf13/cobra"
)

const version = "1.0.0"

func main() {
	root := &cobra.Command{
		Use:     "qg-server",
		Short:   "QualiGuard analysis server",
		Version: version,
		RunE:    runServer,
	}

	root.Flags().String("port", "9000", "HTTP port")
	root.Flags().String("host", "127.0.0.1", "HTTP host")
	root.Flags().String("data-dir", defaultDataDir(), "Directory for SQLite database")
	root.Flags().String("work-dir", "", "Project root for scan (default: current directory)")
	root.Flags().String("config", "qualiguard.yaml", "Path to qualiguard.yaml")

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func runServer(cmd *cobra.Command, _ []string) error {
	host, _ := cmd.Flags().GetString("host")
	port, _ := cmd.Flags().GetString("port")
	dataDir, _ := cmd.Flags().GetString("data-dir")
	workDir, _ := cmd.Flags().GetString("work-dir")
	configPath, _ := cmd.Flags().GetString("config")

	var err error
	if workDir == "" {
		workDir, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	workDir, err = filepath.Abs(workDir)
	if err != nil {
		return err
	}
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(workDir, configPath)
	}

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}

	dbPath := filepath.Join(dataDir, "qualiguard.db")
	st, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer st.Close()

	token, err := st.EnsureDefaultToken(cmd.Context())
	if err != nil {
		return err
	}

	tokenFile := filepath.Join(dataDir, "token.txt")
	if err := os.WriteFile(tokenFile, []byte(token+"\n"), 0o600); err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%s", host, port)
	httpSrv := &http.Server{
		Addr:    addr,
		Handler: server.New(st, token, workDir, configPath).Handler(),
	}

	fmt.Println("QualiGuard Server")
	fmt.Println("=================")
	fmt.Printf("Landing:   http://%s/\n", addr)
	fmt.Printf("Dashboard: http://%s/app\n", addr)
	fmt.Printf("Health:    http://%s/api/health\n", addr)
	fmt.Printf("Work dir:  %s\n", workDir)
	fmt.Printf("Config:    %s\n", configPath)
	fmt.Printf("Database:  %s\n", dbPath)
	fmt.Printf("API Token: %s\n", token)
	fmt.Printf("Token file: %s\n", tokenFile)
	if pw := os.Getenv("QG_PANEL_PASSWORD"); pw != "" {
		fmt.Println("Panel auth:  şifre korumalı (/login)")
	} else {
		fmt.Println("Panel auth:  yerel mod (localhost bootstrap)")
	}
	fmt.Println("")
	fmt.Println("Endpoints:")
	fmt.Println("  POST /api/v1/analyses")
	fmt.Println("  GET  /api/v1/projects")
	fmt.Println("  GET  /api/v1/projects/{key}/issues")
	fmt.Println("")
	fmt.Println("Press Ctrl+C to stop.")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	}()

	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "data"
	}
	return filepath.Join(home, ".qualiguard")
}
