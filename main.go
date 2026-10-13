package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	file := flag.String("file", "", "path to the canned responses CSV file")
	flag.Parse()
	if *file == "" || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}

	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "load .env: %v\n", err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	os.Exit(importFile(context.Background(), *file, os.Getenv("TEXT_API_BASIC_AUTH"), client, createCannedResponseURL, os.Stdout, os.Stderr))
}
