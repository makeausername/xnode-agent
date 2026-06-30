package main

import (
	"context"
	"fmt"
	"os"

	"github.com/makeausername/xnode-agent/internal/bootstrap"
)

var Version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println("xnode-agent", Version)
		return
	}

	if err := bootstrap.Run(context.Background(), Version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
