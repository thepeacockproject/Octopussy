// Command octopussy starts the HTTP server: it builds the logger and JWT
// manager, wires them into the api, and serves with graceful shutdown.
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/thepeacockproject/Octopussy/internal/api"
	"github.com/thepeacockproject/Octopussy/internal/content"
	"github.com/thepeacockproject/Octopussy/internal/controller"
	"github.com/thepeacockproject/Octopussy/internal/jwt"
	"github.com/thepeacockproject/Octopussy/internal/logging"
	"github.com/thepeacockproject/Octopussy/internal/persistence"
	"github.com/thepeacockproject/Octopussy/internal/proxy"
	"github.com/thepeacockproject/Octopussy/internal/shared"
)

// embeddedData holds the static game data baked into the binary so the server
// runs as a single self-contained executable. OCTOPUSSY_DATA can override this
// with an on-disk directory for editing data without recompiling.
//
//go:embed data
var embeddedData embed.FS

func main() {
	if err := run(); err != nil {
		slog.Error("server exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	dev := envBool("OCTOPUSSY_DEV", true)
	reverseProxy := envBool("OCTOPUSSY_REVERSEPROXY", false)

	level := slog.LevelInfo
	if dev {
		level = slog.LevelDebug
	}

	logOptions := logging.Options{
		Level: level,
	}

	if dev {
		logFile, _ := newLogFile()
		logOptions.Writers = []io.Writer{logFile}
	}

	log := logging.New(logOptions)

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
	st, err := persistence.Open(persistence.Config{Driver: dbDriver, DSN: dbDSN, Debug: dev, Logger: log})
	if err != nil {
		return err
	}
	defer st.Close()
	log.Info("opened database", slog.String("driver", dbDriver), slog.String("dsn", dbDSN))

	// Static game data is embedded in the binary by default; OCTOPUSSY_DATA
	// points at an on-disk directory instead so data can be edited without
	// recompiling.
	var dataFS fs.FS
	dataSrc := "embedded"
	if dataDir := os.Getenv("OCTOPUSSY_DATA"); dataDir != "" {
		dataFS = os.DirFS(dataDir)
		dataSrc = dataDir
	} else if dataFS, err = fs.Sub(embeddedData, "data"); err != nil {
		return err
	}
	gameContent, err := content.Load(dataFS)
	if err != nil {
		return err
	}
	log.Info("loaded game content",
		slog.String("source", dataSrc),
		slog.Int("missions", gameContent.MissionCount()),
		slog.Int("challenges", gameContent.ChallengeCount()),
		slog.Int("unlockables", gameContent.UnlockableCount()),
	)

	ctrl := controller.New(log, jwtMgr, st, gameContent)
	api.RegisterRoutes(ctrl)

	addr := os.Getenv("OCTOPUSSY_ADDR")
	if addr == "" {
		addr = ":80"
	}

	srv := &http.Server{
		Addr: addr,
		Handler: ctrl.Handler(
			logging.HTTPMiddleware(),
			logging.HTTPMiddlewareSimple(log),
			shared.ShortHandIf(dev, logging.HTTPMiddlewareDetailed(log), nil),
			shared.ShortHandIf(dev && reverseProxy, proxy.HTTPMiddleware(log), nil),
		),
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

func newLogFile() (*os.File, error) {
	filename := fmt.Sprintf(
		"logs/debug-%s.log",
		time.Now().Format("2006-01-02_15-04-05"),
	)

	os.Mkdir("logs", 0755)

	return os.OpenFile(
		filename,
		os.O_CREATE|os.O_WRONLY,
		0644,
	)
}
