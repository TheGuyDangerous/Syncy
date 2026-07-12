// Command syncyd is the Syncy synchronization daemon.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/TheGuyDangerous/Syncy/engine/internal/buildinfo"
	"github.com/TheGuyDangerous/Syncy/engine/internal/daemon"
)

func main() {
	showVersion := flag.Bool("version", false, "print version information and exit")
	dataDir := flag.String("data-dir", "", "data directory (default: user config dir + /syncy)")
	listen := flag.String("listen", ":22067", "QUIC listen address")
	apiAddr := flag.String("api", "127.0.0.1:22068", "control API address")
	flag.Parse()

	if *showVersion {
		fmt.Println(buildinfo.String())
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	d, err := daemon.New(daemon.Config{DataDir: *dataDir, ListenAddr: *listen, APIAddr: *apiAddr})
	if err != nil {
		logger.Error("failed to initialize", "error", err)
		os.Exit(1)
	}
	defer d.Close()

	logger.Info("syncyd starting",
		"version", buildinfo.Version,
		"device", d.DeviceID(),
		"data_dir", d.DataDir(),
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := d.Run(ctx); err != nil {
		logger.Error("daemon stopped with error", "error", err)
		os.Exit(1)
	}
	logger.Info("syncyd stopped")
}
