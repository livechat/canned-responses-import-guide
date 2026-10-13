package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const createCannedResponseURL = "https://api.livechatinc.com/v3.6/configuration/action/create_canned_response"

func validateAuth(auth string) error {
	scheme, encoded, ok := strings.Cut(auth, " ")
	if !ok || scheme != "Basic" || encoded == "" {
		return errors.New("TEXT_API_BASIC_AUTH must be 'Basic <base64(account_id:PAT)>'")
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return errors.New("TEXT_API_BASIC_AUTH contains invalid Base64")
	}

	accountID, token, ok := strings.Cut(string(decoded), ":")
	if !ok || accountID == "" || token == "" {
		return errors.New("TEXT_API_BASIC_AUTH must encode account_id:PAT")
	}

	return nil
}

type apiError struct {
	status int
	body   string
}

func (e *apiError) Error() string {
	if e.body == "" {
		return fmt.Sprintf("API returned HTTP %d", e.status)
	}

	return fmt.Sprintf("API returned HTTP %d: %s", e.status, e.body)
}

func createResponse(ctx context.Context, client *http.Client, endpoint, auth string, response cannedResponse) (int, error) {
	payload, err := json.Marshal(response)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	result, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer result.Body.Close()

	if result.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(result.Body, 1024))
		return 0, &apiError{status: result.StatusCode, body: strings.TrimSpace(string(body))}
	}

	var created struct {
		ID int `json:"id"`
	}

	if err := json.NewDecoder(result.Body).Decode(&created); err != nil || created.ID <= 0 {
		return 0, errors.New("API returned an invalid create result; check Text before retrying")
	}

	return created.ID, nil
}
