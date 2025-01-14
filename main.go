package main

import (
	"github.com/hashmap-kz/kubectl-docs/pkg/app"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	app.PrintTree()
}
