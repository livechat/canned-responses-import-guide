package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const header = "text,tags,group_id,is_private\n"

func testAuth() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte("123:example-token"))
}

func testCSV(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "responses.csv")
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadResponses(t *testing.T) {
	input := "\ufeff" + header + "\"Hello,\nfriend\",\" greeting | support \",0,false\nPrivate,internal,19,TRUE\n"
	got, err := readResponses(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	want := []cannedResponse{
		{Text: "Hello,\nfriend", Tags: []string{"greeting", "support"}, GroupID: 0, IsPrivate: false, line: 2},
		{Text: "Private", Tags: []string{"internal"}, GroupID: 19, IsPrivate: true, line: 4},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("readResponses() = %#v, want %#v", got, want)
	}
}

func TestReadResponsesRejectsInvalidCSV(t *testing.T) {
	tests := []struct {
		name, csv, message string
	}{
		{"empty", "", "CSV is empty"},
		{"wrong header", "text,group_id,tags,is_private\nHello,0,,false\n", "CSV header must be"},
		{"no rows", header, "CSV has no response rows"},
		{"missing column", header + "Hello,,0\n", "line 2: expected 4 columns"},
		{"blank text", header + "  ,tag,0,false\n", "line 2: text must not be empty"},
		{"missing tags", header + "Hello,,0,false\n", "line 2: tags must contain at least one tag"},
		{"blank tags", header + "Hello,   ,0,false\n", "line 2: tags must contain at least one tag"},
		{"empty tag", header + "Hello,tag||support,0,false\n", "line 2: tags must not contain empty entries"},
		{"bad group", header + "Hello,tag,-1,false\n", "line 2: group_id must be a non-negative integer"},
		{"bad privacy", header + "Hello,tag,0,\n", "line 2: is_private must be true or false"},
		{"invalid UTF-8", header + "\xff,tag,0,false\n", "line 2: CSV must be UTF-8"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := readResponses(strings.NewReader(test.csv))
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("readResponses() error = %v, want %q", err, test.message)
			}
		})
	}
}

func TestValidateAuth(t *testing.T) {
	if err := validateAuth(testAuth()); err != nil {
		t.Fatalf("valid Basic auth rejected: %v", err)
	}
	for _, auth := range []string{"", "Bearer abc", "Basic not-base64!", "Basic " + base64.StdEncoding.EncodeToString([]byte("missing-colon"))} {
		if err := validateAuth(auth); err == nil {
			t.Errorf("validateAuth(%q) should fail", auth)
		}
	}
}

func TestImportFileValidatesBeforeSending(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprint(w, `{"id":1}`)
	}))
	defer server.Close()
	path := testCSV(t, header+"Valid,greeting,0,false\nInvalid,greeting,0,maybe\n")
	var out, errOut bytes.Buffer
	code := importFile(context.Background(), path, testAuth(), server.Client(), server.URL, &out, &errOut)
	if code != 1 || calls != 0 || !strings.Contains(errOut.String(), "line 3: is_private") {
		t.Errorf("code=%d, calls=%d, error=%q", code, calls, errOut.String())
	}
}

func TestImportFileRejectsMissingTagsBeforeSending(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprint(w, `{"id":1}`)
	}))
	defer server.Close()
	path := testCSV(t, header+"Valid,greeting,0,false\nMissing,,0,false\n")
	var out, errOut bytes.Buffer
	code := importFile(context.Background(), path, testAuth(), server.Client(), server.URL, &out, &errOut)
	if code != 1 || calls != 0 || !strings.Contains(errOut.String(), "line 3: tags must contain at least one tag") {
		t.Errorf("code=%d, calls=%d, error=%q", code, calls, errOut.String())
	}
}

func TestImportFileSendsEachRowAndReportsPartialSuccess(t *testing.T) {
	var requests []cannedResponse
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/action/create_canned_response" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != testAuth() || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("missing headers: %v", r.Header)
		}
		var response cannedResponse
		if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
			t.Errorf("decode request: %v", err)
		}
		requests = append(requests, response)
		if len(requests) == 2 {
			http.Error(w, "invalid response text", http.StatusBadRequest)
			return
		}
		fmt.Fprintf(w, `{"id":%d}`, len(requests))
	}))
	defer server.Close()
	path := testCSV(t, header+"First,hello|welcome,0,false\nSecond,follow-up,19,true\nThird,closing,0,false\n")
	var out, errOut bytes.Buffer
	code := importFile(context.Background(), path, testAuth(), server.Client(), server.URL+"/action/create_canned_response", &out, &errOut)
	if code != 1 || out.String() != "Created 2 of 3 canned responses.\n" || !strings.Contains(errOut.String(), "line 3: API returned HTTP 400") {
		t.Errorf("code=%d, output=%q, error=%q", code, out.String(), errOut.String())
	}
	want := []cannedResponse{
		{Text: "First", Tags: []string{"hello", "welcome"}, GroupID: 0},
		{Text: "Second", Tags: []string{"follow-up"}, GroupID: 19, IsPrivate: true},
		{Text: "Third", Tags: []string{"closing"}, GroupID: 0},
	}
	if !reflect.DeepEqual(requests, want) {
		t.Errorf("requests = %#v, want %#v", requests, want)
	}
}

func TestImportFileReportsFullSuccess(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprintf(w, `{"id":%d}`, calls)
	}))
	defer server.Close()
	path := testCSV(t, header+"First,greeting,0,false\nSecond,closing,0,false\n")
	var out, errOut bytes.Buffer
	code := importFile(context.Background(), path, testAuth(), server.Client(), server.URL, &out, &errOut)
	if code != 0 || calls != 2 || out.String() != "Created 2 of 2 canned responses.\n" || errOut.Len() != 0 {
		t.Errorf("code=%d, calls=%d, output=%q, error=%q", code, calls, out.String(), errOut.String())
	}
}

func TestImportFileStopsAfterServerFailure(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 2 {
			http.Error(w, "try later", http.StatusServiceUnavailable)
			return
		}
		fmt.Fprintf(w, `{"id":%d}`, calls)
	}))
	defer server.Close()
	path := testCSV(t, header+"First,greeting,0,false\nSecond,update,0,false\nThird,closing,0,false\n")
	var out, errOut bytes.Buffer
	code := importFile(context.Background(), path, testAuth(), server.Client(), server.URL, &out, &errOut)
	if code != 1 || calls != 2 || out.String() != "Created 1 of 3 canned responses.\n" || !strings.Contains(errOut.String(), "Import stopped") {
		t.Errorf("code=%d, calls=%d, output=%q, error=%q", code, calls, out.String(), errOut.String())
	}
}

func TestImportFileStopsWhenCreateResultIsUncertain(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprint(w, `{}`)
	}))
	defer server.Close()
	path := testCSV(t, header+"First,greeting,0,false\nSecond,closing,0,false\n")
	var out, errOut bytes.Buffer
	code := importFile(context.Background(), path, testAuth(), server.Client(), server.URL, &out, &errOut)
	if code != 1 || calls != 1 || out.String() != "Created 0 of 2 canned responses.\n" || !strings.Contains(errOut.String(), "check Text before retrying") {
		t.Errorf("code=%d, calls=%d, output=%q, error=%q", code, calls, out.String(), errOut.String())
	}
}
