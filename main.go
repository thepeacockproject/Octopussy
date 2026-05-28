// Command octopussy starts the HTTP server: it builds the logger and JWT
// manager, wires them into the controller, and serves with graceful shutdown.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/thepeacockproject/Octopussy/internal/content"
	"github.com/thepeacockproject/Octopussy/internal/controller"
	"github.com/thepeacockproject/Octopussy/internal/jwt"
	"github.com/thepeacockproject/Octopussy/internal/logging"
	"github.com/thepeacockproject/Octopussy/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	dev := envBool("OCTOPUSSY_DEV", true)

	level := slog.LevelInfo
	if dev {
		level = slog.LevelDebug
	}

	log := logging.New(logging.Options{Level: level})

	// An RSA signing key may be provided as a PEM file for a stable kid across
	// restarts; otherwise an ephemeral key is generated.
	var keyPEM []byte
	if path := os.Getenv("OCTOPUSSY_JWT_KEY_FILE"); path != "" {
		var err error
		if keyPEM, err = os.ReadFile(path); err != nil {
			return err
		}
	}

	jwtMgr, err := jwt.NewManager(jwt.Config{
		PrivateKeyPEM: keyPEM,
		TTL:           2 * time.Hour,
		Logger:        log,
	})
	if err != nil {
		return err
	}

	dbDriver := os.Getenv("OCTOPUSSY_DB_DRIVER")
	if dbDriver == "" {
		dbDriver = "sqlite"
	}
	dbDSN := os.Getenv("OCTOPUSSY_DB")
	if dbDSN == "" {
		dbDSN = "octopussy.db"
	}
	st, err := store.Open(store.Config{Driver: dbDriver, DSN: dbDSN, Debug: dev})
	if err != nil {
		return err
	}
	defer st.Close()
	log.Info("opened database", slog.String("driver", dbDriver), slog.String("dsn", dbDSN))

	// Static game data lives on disk so it can be edited without recompiling.
	dataDir := os.Getenv("OCTOPUSSY_DATA")
	if dataDir == "" {
		dataDir = "data"
	}
	gameContent, err := content.Load(os.DirFS(dataDir))
	if err != nil {
		return err
	}
	log.Info("loaded game content",
		slog.String("dir", dataDir),
		slog.Int("missions", gameContent.MissionCount()),
	)

	ctrl := controller.New(log, jwtMgr, st, gameContent)

	addr := os.Getenv("OCTOPUSSY_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           ctrl.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Listen for shutdown signals.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Info("server listening", slog.String("addr", addr), slog.Bool("dev", dev))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("shutdown signal received, draining connections")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		log.Info("server stopped cleanly")
		return nil
	}
}

// envBool reads a boolean-ish environment variable, falling back to def when
// unset. Accepts the usual truthy strings (1, true, yes, on).
func envBool(key string, def bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}
