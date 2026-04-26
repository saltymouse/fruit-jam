package fetch

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const userAgent = "fruit-jam/0.1 (like Lynx/2.9.0)"

var client = &http.Client{Timeout: 10 * time.Second}

// Result holds the fetched page content and its final URL (after redirects).
type Result struct {
	URL  string
	Body string
}

func Get(rawURL string) (Result, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,*/*")
	return do(req)
}

func Post(rawURL string, values url.Values) (Result, error) {
	req, err := http.NewRequest(http.MethodPost, rawURL, strings.NewReader(values.Encode()))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,*/*")
	return do(req)
}

func do(req *http.Request) (Result, error) {
	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return Result{}, fmt.Errorf("HTTP %d %s", resp.StatusCode, resp.Status)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "" && !strings.Contains(ct, "text/html") && !strings.Contains(ct, "text/plain") {
		return Result{}, fmt.Errorf("unsupported content-type: %s", ct)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, err
	}

	return Result{URL: resp.Request.URL.String(), Body: string(body)}, nil
}
