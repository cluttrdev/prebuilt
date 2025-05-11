package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	if err := execute(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
