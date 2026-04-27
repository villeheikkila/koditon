package main

import (
	"context"
	"fmt"
	"os"

	"koditon/internal/app"
)

func main() {
	if err := app.Run(context.Background(), os.Args, os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
