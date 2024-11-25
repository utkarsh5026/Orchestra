package utils

import (
	"net/http"
	"time"
)

type HttpRetryOptions struct {
	MaxRetries int
	WaitTime   time.Duration
}

func DefaultHttpRetryOptions() HttpRetryOptions {
	return HttpRetryOptions{
		MaxRetries: 10,
		WaitTime:   5 * time.Second,
	}
}

// HTTPWithRetry executes an HTTP request with automatic retries on failure.
//
// Parameters:
//   - f: The HTTP request function to execute, taking a URL string and returning a response and error
//   - url: The URL to make the request to
//   - options: Optional retry configuration. If nil, default options will be used
//
// Returns:
//   - *http.Response: The HTTP response if successful
//   - error: Any error that occurred after all retries were exhausted
func HTTPWithRetry(f func(string) (*http.Response, error), url string, options *HttpRetryOptions) (*http.Response, error) {
	if options == nil {
		def := DefaultHttpRetryOptions()
		options = &def
	}

	var resp *http.Response
	var err error
	for i := 0; i < options.MaxRetries; i++ {
		resp, err = f(url)
		if err != nil {
			time.Sleep(options.WaitTime)
		} else {
			break
		}
	}
	return resp, err
}
