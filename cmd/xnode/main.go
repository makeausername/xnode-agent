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

	if len(os.Args) > 1 && os.Args[1] == "--check" {
		app, err := bootstrap.NewApp(Version)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		app.Logger = nil
		if err := app.SyncOnce(context.Background()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("xnode-agent check ok")
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "--once" {
		app, err := bootstrap.NewApp(Version)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := app.SyncOnce(context.Background()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("xnode-agent once ok")
		return
	}

	if err := bootstrap.Run(context.Background(), Version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
