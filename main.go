package main

import (
	"fmt"
	"os"

	"github.com/hashmap-kz/kubectl-docs/pkg/apidocs"
)

func main() {
	if err := apidocs.NewCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
