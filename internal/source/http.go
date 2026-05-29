package source

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	securityv1alpha1 "github.com/DavidHoenisch/coraza-operator/api/v1alpha1"
)

// HTTPFetcher loads blocklists from a remote HTTP endpoint.
type HTTPFetcher struct {
	Client *http.Client
}

func NewHTTPFetcher() *HTTPFetcher {
	return &HTTPFetcher{Client: http.DefaultClient}
}

func (f *HTTPFetcher) Type() securityv1alpha1.SourceType {
	return securityv1alpha1.SourceTypeHTTP
}

func (f *HTTPFetcher) Fetch(ctx context.Context, source securityv1alpha1.SourceSpec) (Result, error) {
	if source.HTTP == nil {
		return Result{}, fmt.Errorf("http config is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.HTTP.URL, nil)
	if err != nil {
		return Result{}, fmt.Errorf("create request: %w", err)
	}

	resp, err := f.Client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("fetch blocklist: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("fetch blocklist: unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, fmt.Errorf("read response body: %w", err)
	}

	revision := strings.TrimSpace(resp.Header.Get("ETag"))
	if revision == "" {
		revision = strings.TrimSpace(resp.Header.Get("Last-Modified"))
	}
	if revision == "" {
		revision = resp.Header.Get("Date")
	}

	return Result{
		IPs:      ParseBlocklist(body),
		Revision: revision,
	}, nil
}
