package main

import (
	"fmt"
	"os"

	"github.com/ethersphere/beekeeper/cmd/beekeeper/cmd"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
