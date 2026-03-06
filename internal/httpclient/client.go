package httpclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/erlandas/ratatuile/internal/domain"
)

// Response holds the result of an HTTP request execution.
type Response struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Duration   time.Duration     `json:"duration"`
	Size       int               `json:"size"`
}

// Client wraps http.Client so we can inject timeouts and
// swap the transport in tests if needed.
type Client struct {
	http *http.Client
}

func NewClient() *Client {
	return &Client{
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

// Do executes a domain.Request and returns a Response.
func (c *Client) Do(ctx context.Context, req domain.Request) (Response, error) {
	var bodyReader io.Reader
	if req.Body != "" {
		bodyReader = strings.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, bodyReader)
	if err != nil {
		return Response{}, fmt.Errorf("creating request: %w", err)
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	q := httpReq.URL.Query()
	for k, v := range req.Params {
		q.Set(k, v)
	}
	httpReq.URL.RawQuery = q.Encode()

	start := time.Now()
	resp, err := c.http.Do(httpReq)
	duration := time.Since(start)
	if err != nil {
		return Response{}, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, fmt.Errorf("reading response body: %w", err)
	}

	headers := make(map[string]string)
	for k := range resp.Header {
		headers[k] = resp.Header.Get(k)
	}

	return Response{
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       string(body),
		Duration:   duration,
		Size:       len(body),
	}, nil
}
