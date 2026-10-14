package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

type cannedResponse struct {
	Text      string   `json:"text"`
	Shortcuts []string `json:"tags"`
	GroupID   int      `json:"group_id"`
	IsPrivate bool     `json:"is_private"`
	line      int
}

func readResponses(input io.Reader) ([]cannedResponse, error) {
	reader := csv.NewReader(input)
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if errors.Is(err, io.EOF) {
		return nil, errors.New("CSV is empty")
	}
	if err != nil {
		return nil, fmt.Errorf("read CSV header: %w", err)
	}

	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], "\ufeff")
	}

	want := []string{"text", "shortcuts", "group_id", "is_private"}
	if len(header) != len(want) {
		return nil, fmt.Errorf("CSV header must be %s", strings.Join(want, ","))
	}

	for i := range want {
		if header[i] != want[i] {
			return nil, fmt.Errorf("CSV header must be %s", strings.Join(want, ","))
		}
	}

	var responses []cannedResponse
	for {
		fields, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read CSV: %w", err)
		}

		line, _ := reader.FieldPos(0)
		if len(fields) != len(want) {
			return nil, fmt.Errorf("line %d: expected %d columns, got %d", line, len(want), len(fields))
		}

		for _, field := range fields {
			if !utf8.ValidString(field) {
				return nil, fmt.Errorf("line %d: CSV must be UTF-8", line)
			}
		}

		if strings.TrimSpace(fields[0]) == "" {
			return nil, fmt.Errorf("line %d: text must not be empty", line)
		}

		if strings.TrimSpace(fields[1]) == "" {
			return nil, fmt.Errorf("line %d: shortcuts must contain at least one shortcut", line)
		}
		shortcuts := make([]string, 0)
		for _, value := range strings.Split(fields[1], "|") {
			shortcut := strings.TrimSpace(value)
			if shortcut == "" {
				return nil, fmt.Errorf("line %d: shortcuts must not contain empty entries", line)
			}
			shortcuts = append(shortcuts, shortcut)
		}

		groupID, err := strconv.Atoi(strings.TrimSpace(fields[2]))
		if err != nil || groupID < 0 {
			return nil, fmt.Errorf("line %d: group_id must be a non-negative integer", line)
		}

		var isPrivate bool
		switch strings.ToLower(strings.TrimSpace(fields[3])) {
		case "true":
			isPrivate = true
		case "false":
		default:
			return nil, fmt.Errorf("line %d: is_private must be true or false", line)
		}

		responses = append(responses, cannedResponse{
			Text:      fields[0],
			Shortcuts: shortcuts,
			GroupID:   groupID,
			IsPrivate: isPrivate,
			line:      line,
		})
	}

	if len(responses) == 0 {
		return nil, errors.New("CSV has no response rows")
	}

	return responses, nil
}
