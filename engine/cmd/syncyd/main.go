// Command syncyd is the Syncy synchronization daemon.
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
}
