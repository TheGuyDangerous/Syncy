// Command syncyd is the Syncy synchronization daemon.
//
// It embeds the sync engine and, in later milestones, exposes a local control
// API over loopback for the desktop and mobile clients to drive. For now it is
// a thin entry point that reports build information; the engine, discovery and
// control API are wired up in subsequent milestones.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/TheGuyDangerous/Syncy/engine/internal/buildinfo"
)

func main() {
	showVersion := flag.Bool("version", false, "print version information and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(buildinfo.String())
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("syncyd starting",
		"version", buildinfo.Version,
		"commit", buildinfo.Commit,
	)
	logger.Info("engine is not wired up yet; this is an early development build")
	// Later milestones start the engine, device discovery and the local
	// control API here, then block until a shutdown signal is received.
}
