package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseCLIFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want options
	}{
		{
			name: "defaults",
			args: []string{"https://example.com"},
			want: options{
				method: http.MethodGet,
				url:    "https://example.com",
			},
		},
		{
			name: "follow redirects",
			args: []string{"-L", "https://example.com"},
			want: options{
				followRedirects: true,
				method:          http.MethodGet,
				url:             "https://example.com",
			},
		},
		{
			name: "POST with JSON and lowercase method",
			args: []string{
				"-method", "post",
				"-body", `{"name":"Rex"}`,
				"-content-type", "application/json",
				"https://example.com",
			},
			want: options{
				method:      http.MethodPost,
				body:        `{"name":"Rex"}`,
				contentType: "application/json",
				url:         "https://example.com",
			},
		},
		{
			name: "POST without body",
			args: []string{"-method", "POST", "https://example.com"},
			want: options{
				method: http.MethodPost,
				url:    "https://example.com",
			},
		},
		{
			name: "body without content type",
			args: []string{
				"-method", "POST",
				"-body", "hello",
				"https://example.com",
			},
			want: options{
				method: http.MethodPost,
				body:   "hello",
				url:    "https://example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCLIFlags(tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseCLIFlagsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing URL",
			wantErr: "expected exactly one URL",
		},
		{
			name:    "multiple URLs",
			args:    []string{"https://example.com", "https://example.org"},
			wantErr: "expected exactly one URL",
		},
		{
			name:    "unknown flag",
			args:    []string{"-unknown", "https://example.com"},
			wantErr: "flag provided but not defined",
		},
		{
			name:    "missing flag value",
			args:    []string{"-method"},
			wantErr: "flag needs an argument",
		},
		{
			name:    "invalid boolean",
			args:    []string{"-L=maybe", "https://example.com"},
			wantErr: "invalid boolean value",
		},
		{
			name:    "unsupported method",
			args:    []string{"-method", "INVALID", "https://example.com"},
			wantErr: "unsupported HTTP method",
		},
		{
			name: "GET with body",
			args: []string{
				"-body", "hello", "https://example.com",
			},
			wantErr: "request body is not supported",
		},
		{
			name: "HEAD with body",
			args: []string{
				"-method", "HEAD",
				"-body", "hello",
				"https://example.com",
			},
			wantErr: "request body is not supported",
		},
		{
			name: "content type without body",
			args: []string{
				"-method", "POST",
				"-content-type", "application/json",
				"https://example.com",
			},
			wantErr: "-content-type requires",
		},
		{
			name:    "missing scheme",
			args:    []string{"example.com"},
			wantErr: "URL must include",
		},
		{
			name:    "unsupported scheme",
			args:    []string{"ftp://example.com"},
			wantErr: "URL must include",
		},
		{
			name:    "missing host",
			args:    []string{"https:///path"},
			wantErr: "URL must include",
		},
		{
			name:    "malformed URL",
			args:    []string{"https://example.com/%zz"},
			wantErr: "invalid URL",
		},
		{
			name:    "flag after URL",
			args:    []string{"https://example.com", "-L"},
			wantErr: "expected exactly one URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseCLIFlags(tt.args)
			if err == nil {
				t.Fatal("expected an error")
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseCLIFlagsHelp(t *testing.T) {
	for _, arg := range []string{"-h", "-help"} {
		t.Run(arg, func(t *testing.T) {
			_, err := parseCLIFlags([]string{arg})
			if !errors.Is(err, flag.ErrHelp) {
				t.Errorf("error = %v, want flag.ErrHelp", err)
			}
		})
	}
}

func TestValidateHTTPMethod(t *testing.T) {
	methods := []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			if err := validateHTTPMethod(method); err != nil {
				t.Errorf("valid method rejected: %v", err)
			}
		})
	}
}

// Exercises the full path: flags → request → server → output.
func TestRunRequests(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		method      string
		body        string
		contentType string
	}{
		{
			name:   "GET",
			method: http.MethodGet,
		},
		{
			name:   "HEAD",
			args:   []string{"-method", "HEAD"},
			method: http.MethodHead,
		},
		{
			name: "POST JSON",
			args: []string{
				"-method", "POST",
				"-body", `{"name":"Rex"}`,
				"-content-type", "application/json",
			},
			method:      http.MethodPost,
			body:        `{"name":"Rex"}`,
			contentType: "application/json",
		},
		{
			name: "POST text",
			args: []string{
				"-method", "POST",
				"-body", "Hello from Go!",
				"-content-type", "text/plain",
			},
			method:      http.MethodPost,
			body:        "Hello from Go!",
			contentType: "text/plain",
		},
		{
			name: "POST form",
			args: []string{
				"-method", "POST",
				"-body", "name=Rex&age=3",
				"-content-type", "application/x-www-form-urlencoded",
			},
			method:      http.MethodPost,
			body:        "name=Rex&age=3",
			contentType: "application/x-www-form-urlencoded",
		},
		{
			name:   "empty POST",
			args:   []string{"-method", "POST"},
			method: http.MethodPost,
		},
		{
			name: "PUT",
			args: []string{
				"-method", "PUT",
				"-body", "replacement",
			},
			method: http.MethodPut,
			body:   "replacement",
		},
		{
			name: "PATCH",
			args: []string{
				"-method", "PATCH",
				"-body", `{"age":4}`,
				"-content-type", "application/json",
			},
			method:      http.MethodPatch,
			body:        `{"age":4}`,
			contentType: "application/json",
		},
		{
			name:   "DELETE",
			args:   []string{"-method", "DELETE"},
			method: http.MethodDelete,
		},
		{
			name:   "OPTIONS",
			args:   []string{"-method", "OPTIONS"},
			method: http.MethodOptions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					if r.Method != tt.method {
						t.Errorf("method = %q, want %q", r.Method, tt.method)
					}

					if r.URL.Path != "/echo" || r.URL.Query().Get("name") != "Rex" {
						t.Errorf("unexpected request URL: %s", r.URL)
					}

					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Errorf("reading request body: %v", err)
						http.Error(w, "read failed", http.StatusInternalServerError)
						return
					}

					if string(body) != tt.body {
						t.Errorf("body = %q, want %q", body, tt.body)
					}

					if got := r.Header.Get("Content-Type"); got != tt.contentType {
						t.Errorf("Content-Type = %q, want %q", got, tt.contentType)
					}

					if r.Method != http.MethodHead {
						_, _ = io.WriteString(w, "response from server")
					}
				},
			))
			t.Cleanup(server.Close)

			args := append([]string{}, tt.args...)
			args = append(args, server.URL+"/echo?name=Rex")

			var output bytes.Buffer
			if err := run(args, &output); err != nil {
				t.Fatalf("run failed: %v", err)
			}

			want := "response from server"
			if tt.method == http.MethodHead {
				want = ""
			}

			if output.String() != want {
				t.Errorf("output = %q, want %q", output.String(), want)
			}
		})
	}
}

func TestRunRedirects(t *testing.T) {
	tests := []struct {
		name   string
		follow bool
		want   string
	}{
		{"disabled by default", false, "redirect response"},
		{"enabled with L", true, "destination response"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/start" {
						w.Header().Set("Location", "/destination")
						w.WriteHeader(http.StatusFound)
						_, _ = io.WriteString(w, "redirect response")
						return
					}

					if r.URL.Path != "/destination" {
						t.Errorf("unexpected path: %s", r.URL.Path)
					}

					if !tt.follow {
						t.Error("client followed a redirect without -L")
					}

					_, _ = io.WriteString(w, "destination response")
				},
			))
			t.Cleanup(server.Close)

			var args []string
			if tt.follow {
				args = append(args, "-L")
			}
			args = append(args, server.URL+"/start")

			var output bytes.Buffer
			if err := run(args, &output); err != nil {
				t.Fatalf("run failed: %v", err)
			}

			if output.String() != tt.want {
				t.Errorf("output = %q, want %q", output.String(), tt.want)
			}
		})
	}
}

func TestRunRedirectLoop(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "/loop")
			w.WriteHeader(http.StatusFound)
		},
	))
	t.Cleanup(server.Close)

	err := run([]string{"-L", server.URL + "/loop"}, io.Discard)
	if err == nil {
		t.Fatal("expected redirect-limit error")
	}
}

// HTTP error statuses are responses, not transport errors.
func TestRunHTTPErrorStatuses(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(status)
					_, _ = io.WriteString(w, "error response body")
				},
			))
			t.Cleanup(server.Close)

			var output bytes.Buffer
			if err := run([]string{server.URL}, &output); err != nil {
				t.Fatalf("HTTP status should not cause a Go error: %v", err)
			}

			if output.String() != "error response body" {
				t.Errorf("unexpected output: %q", output.String())
			}
		})
	}
}

func TestRunConnectionFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {},
	))
	server.Close()

	var output bytes.Buffer
	err := run([]string{server.URL}, &output)
	if err == nil {
		t.Fatal("expected connection error")
	}

	if output.Len() != 0 {
		t.Errorf("unexpected output: %q", output.String())
	}
}

type failingWriter struct {
	err error
}

func (w failingWriter) Write(p []byte) (int, error) {
	return 0, w.err
}

func TestRunOutputFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, "response body")
		},
	))
	t.Cleanup(server.Close)

	wantErr := errors.New("output unavailable")

	err := run([]string{server.URL}, failingWriter{err: wantErr})
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want wrapped output error", err)
	}
}

func TestRunInvalidArguments(t *testing.T) {
	var output bytes.Buffer

	err := run([]string{"-unknown"}, &output)
	if err == nil {
		t.Fatal("expected argument error")
	}

	if output.Len() != 0 {
		t.Errorf("unexpected output: %q", output.String())
	}
}
