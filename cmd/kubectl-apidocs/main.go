package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/hashmap-kz/kubectl-apidocs/pkg/apidocs"

	"github.com/hashmap-kz/kubectl-apidocs/pkg/cmd"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	// debug logger
	file, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		slog.Error("Failed to open log file", "error", err)
		return
	}
	defer file.Close()
	logger := apidocs.InitLogger(file)
	slog.SetDefault(logger)

	if err := cmd.NewCmdAPIDocs().Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
