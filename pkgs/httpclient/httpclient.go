// Package httpclient provides the shared HTTP client used for outbound requests.
package httpclient

import (
	"net/http"
	"time"
)

// DefaultTimeout bounds every request made through Shared so a hung upstream
// cannot block a serving goroutine indefinitely.
const DefaultTimeout = 30 * time.Second

// Shared is the HTTP client used for all outbound calls. Reusing a single
// client keeps connection pooling effective and guarantees every caller gets
// the same timeout.
var Shared = &http.Client{Timeout: DefaultTimeout}
