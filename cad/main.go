package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		_, _ = fmt.Fprintln(os.Stderr, "No filename provided")
		os.Exit(1)
	}

	filePath := os.Args[1]

	file, err := os.Open(filePath)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Could not open file at", filePath, "error:", err)
		os.Exit(1)
	}

	defer file.Close()

	_, err = io.Copy(os.Stdout, file)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Could not read file at", filePath, "error:", err)
		os.Exit(1)
	}
}
