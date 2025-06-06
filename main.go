package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/hashmap-kz/kubectl-apidocs/internal/apidocs"

	"github.com/hashmap-kz/kubectl-apidocs/internal/cmd"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	// debug logger
	var logFile *os.File
	if os.Getenv("KUBECTL_APIDOCS_DEBUG_LOG") == "enable" {
		lg := filepath.Join(os.TempDir(), "kubectl-apidocs.log")
		logFile, err := os.OpenFile(lg, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o666)
		if err != nil {
			slog.Error("failed to open log file", "error", err)
			return
		}
		slog.SetDefault(apidocs.InitLogger(logFile))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	}

	// app
	if err := cmd.NewCmdAPIDocs().Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		_ = logFile.Close()
		os.Exit(1)
	}

	_ = logFile.Close()
}
