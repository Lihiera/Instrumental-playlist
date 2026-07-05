package main

import (
	"fmt"
	"os"

	"instrumental-playlist/internal/app"
)

// main starts the instrumental-playlist application and reports startup errors to stderr.
func main() {
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
