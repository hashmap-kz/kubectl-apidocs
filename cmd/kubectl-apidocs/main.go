package main

import (
	"fmt"
	"os"

	"github.com/hashmap-kz/kubectl-apidocs/pkg/cmd"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	if err := cmd.NewCmd().Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
