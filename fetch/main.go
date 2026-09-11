package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const usage = `Usage: fetch [flags] URL
  -L                    Follow redirects
  -method METHOD        HTTP method (default GET)
  -body BODY            Request body
  -content-type TYPE    Content-Type header
`

type options struct {
	followRedirects bool
	method          string
	body            string
	contentType     string
	url             string
}

func main() {
	err := run(os.Args[1:], os.Stdout)

	if errors.Is(err, flag.ErrHelp) {
		fmt.Print(usage)
		return
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "Request failed:", err)
		os.Exit(1)
	}
}

func run(args []string, output io.Writer) error {
	opts, err := parseCLIFlags(args)
	if err != nil {
		return err
	}

	response, err := doHTTPRequest(opts)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if _, err := io.Copy(output, response.Body); err != nil {
		return fmt.Errorf("could not write response: %w", err)
	}

	return nil
}

func parseCLIFlags(args []string) (options, error) {
	var opts options

	flags := flag.NewFlagSet("fetch", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	flags.BoolVar(&opts.followRedirects, "L", false, "Follow redirects")
	flags.StringVar(&opts.method, "method", http.MethodGet, "HTTP method")
	flags.StringVar(&opts.body, "body", "", "Request body")
	flags.StringVar(&opts.contentType, "content-type", "", "Content type")

	if err := flags.Parse(args); err != nil {
		return options{}, err
	}

	opts.method = strings.ToUpper(opts.method)

	if err := validateHTTPMethod(opts.method); err != nil {
		return options{}, err
	}

	if (opts.method == http.MethodGet || opts.method == http.MethodHead) &&
		opts.body != "" {
		return options{}, fmt.Errorf("request body is not supported for GET or HEAD")
	}

	if opts.contentType != "" && opts.body == "" {
		return options{}, fmt.Errorf("-content-type requires a non-empty -body")
	}

	if flags.NArg() != 1 {
		return options{}, fmt.Errorf("expected exactly one URL\n%s", usage)
	}

	opts.url = flags.Arg(0)

	requestURL, err := url.Parse(opts.url)
	if err != nil {
		return options{}, fmt.Errorf("invalid URL: %w", err)
	}

	if (requestURL.Scheme != "http" && requestURL.Scheme != "https") ||
		requestURL.Hostname() == "" {
		return options{}, fmt.Errorf("URL must include http:// or https:// and a host")
	}

	return opts, nil
}

func doHTTPRequest(opts options) (*http.Response, error) {
	var body io.Reader
	if opts.body != "" {
		body = strings.NewReader(opts.body)
	}

	request, err := http.NewRequest(opts.method, opts.url, body)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}

	if opts.contentType != "" {
		request.Header.Set("Content-Type", opts.contentType)
	}

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	if !opts.followRedirects {
		client.CheckRedirect = func(
			req *http.Request,
			via []*http.Request,
		) error {
			return http.ErrUseLastResponse
		}
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("could not send request: %w", err)
	}

	return response, nil
}

func validateHTTPMethod(method string) error {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost,
		http.MethodPut, http.MethodPatch, http.MethodDelete,
		http.MethodConnect, http.MethodOptions, http.MethodTrace:
		return nil
	default:
		return fmt.Errorf("unsupported HTTP method: %q", method)
	}
}
