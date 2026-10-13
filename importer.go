package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

func importFile(ctx context.Context, path, auth string, client *http.Client, endpoint string, out, errOut io.Writer) int {
	if err := validateAuth(auth); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}

	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(errOut, "open CSV: %v\n", err)
		return 1
	}
	defer file.Close()

	responses, err := readResponses(file)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}

	var (
		created = 0
		failed  = false
	)

	for _, response := range responses {
		_, err := createResponse(ctx, client, endpoint, auth, response)
		if err == nil {
			created++
			continue
		}

		failed = true
		fmt.Fprintf(errOut, "line %d: %v\n", response.line, err)
		var apiErr *apiError
		if !errors.As(err, &apiErr) || apiErr.status < 400 || apiErr.status == http.StatusUnauthorized || apiErr.status == http.StatusForbidden || apiErr.status == http.StatusNotFound || apiErr.status == http.StatusRequestTimeout || apiErr.status == http.StatusTooManyRequests || apiErr.status >= 500 {
			fmt.Fprintln(errOut, "Import stopped; check Text before retrying. A rerun creates every row again.")
			break
		}
	}

	fmt.Fprintf(out, "Created %d of %d canned responses.\n", created, len(responses))
	if failed {
		return 1
	}

	return 0
}
