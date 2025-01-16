package main

import (
	"fmt"
	"os"

	"github.com/hashmap-kz/kubectl-apidocs/pkg/apidocs"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	if err := apidocs.NewCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
