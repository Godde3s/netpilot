// Package health performs HTTP(S) uptime checks with status classification.
package health

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/Godde3s/netpilot/internal/ui"
)

// Outcome is the result of one health check.
type Outcome struct {
	URL    string
	Status int
	Took   time.Duration
	Err    error
	Up     bool
}

// Client builds a tuned HTTP client for probing.
func Client(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: timeout,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
}

// Check issues a GET (or HEAD with -head) and classifies the response.
func Check(c *http.Client, rawURL string, headOnly bool) Outcome {
	start := time.Now()
	var resp *http.Response
	var err error
	if headOnly {
		resp, err = c.Head(rawURL)
	} else {
		resp, err = c.Get(rawURL)
	}
	took := time.Since(start)
	if err != nil {
		return Outcome{URL: rawURL, Err: err, Took: took, Up: false}
	}
	defer resp.Body.Close()
	up := resp.StatusCode >= 200 && resp.StatusCode < 400
	return Outcome{URL: rawURL, Status: resp.StatusCode, Took: took, Up: up}
}

// Report prints one outcome line with a color-coded verdict.
func Report(o Outcome) {
	if o.Err != nil {
		ui.Fail("%s — %s (%s)", o.URL, o.Err, o.Took.Round(time.Millisecond).String())
		return
	}
	verdict := ui.Green("UP")
	switch {
	case o.Status >= 500:
		verdict = ui.Red("SERVER-ERR")
	case o.Status >= 400:
		verdict = ui.Yellow("CLIENT-ERR")
	case o.Status >= 300:
		verdict = ui.Cyan("REDIRECT")
	}
	ui.Success("%s — %s %s", o.URL, verdict, ui.Gray(o.Took.Round(time.Millisecond).String()))
}
