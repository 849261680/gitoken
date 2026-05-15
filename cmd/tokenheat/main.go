package main

import (
	"fmt"
	"os"

	"github.com/849261680/token-heatmap/internal/tokenheat"
)

func main() {
	if err := tokenheat.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
