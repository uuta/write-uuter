package main

import (
	"fmt"
	"os"

	"github.com/uuta/write-uuter/internal/captureadapter"
)

func main() {
	if err := captureadapter.RunCloudflare(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
