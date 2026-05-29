package source

import (
	"context"
	"fmt"
	"sort"
	"strings"

	securityv1alpha1 "github.com/DavidHoenisch/coraza-operator/api/v1alpha1"
)

// Result holds the outcome of fetching a single source.
type Result struct {
	IPs      []string
	Revision string
}

// Fetcher loads blocked IPs from one configured source.
type Fetcher interface {
	Type() securityv1alpha1.SourceType
	Fetch(ctx context.Context, source securityv1alpha1.SourceSpec) (Result, error)
}

// Registry routes source specs to registered fetchers.
type Registry struct {
	fetchers map[securityv1alpha1.SourceType]Fetcher
}

// NewRegistry builds a registry from one or more fetchers.
func NewRegistry(fetchers ...Fetcher) *Registry {
	registry := &Registry{fetchers: make(map[securityv1alpha1.SourceType]Fetcher, len(fetchers))}
	for _, fetcher := range fetchers {
		registry.fetchers[fetcher.Type()] = fetcher
	}
	return registry
}

// DefaultRegistry returns the built-in fetcher registry.
func DefaultRegistry() *Registry {
	return NewRegistry(
		NewGitFetcher(),
		NewHTTPFetcher(),
	)
}

// FetchAll loads and merges IPs from every configured source.
func (r *Registry) FetchAll(ctx context.Context, sources []securityv1alpha1.SourceSpec) (Result, error) {
	if len(sources) == 0 {
		return Result{}, fmt.Errorf("spec.sources must contain at least one source")
	}

	merged := make([]string, 0)
	seen := make(map[string]struct{})
	revisions := make([]string, 0, len(sources))

	for i, sourceSpec := range sources {
		if err := ValidateSource(sourceSpec); err != nil {
			return Result{}, fmt.Errorf("source[%d]: %w", i, err)
		}

		fetcher, ok := r.fetchers[sourceSpec.Type]
		if !ok {
			return Result{}, fmt.Errorf("source[%d]: unsupported source type %q", i, sourceSpec.Type)
		}

		result, err := fetcher.Fetch(ctx, sourceSpec)
		if err != nil {
			return Result{}, fmt.Errorf("source[%d]: %w", i, err)
		}

		for _, ip := range result.IPs {
			if _, exists := seen[ip]; exists {
				continue
			}
			seen[ip] = struct{}{}
			merged = append(merged, ip)
		}

		if result.Revision != "" {
			revisions = append(revisions, fmt.Sprintf("%s:%s", strings.ToLower(string(sourceSpec.Type)), result.Revision))
		}
	}

	sort.Strings(revisions)

	return Result{
		IPs:      merged,
		Revision: strings.Join(revisions, ","),
	}, nil
}

// ValidateSource checks that a source spec matches its declared type.
func ValidateSource(source securityv1alpha1.SourceSpec) error {
	switch source.Type {
	case securityv1alpha1.SourceTypeGit:
		if source.Git == nil {
			return fmt.Errorf("git config is required for Git sources")
		}
		if source.Git.URL == "" {
			return fmt.Errorf("git.url is required")
		}
		if source.Git.Path == "" {
			return fmt.Errorf("git.path is required")
		}
	case securityv1alpha1.SourceTypeHTTP:
		if source.HTTP == nil {
			return fmt.Errorf("http config is required for HTTP sources")
		}
		if source.HTTP.URL == "" {
			return fmt.Errorf("http.url is required")
		}
	default:
		return fmt.Errorf("unsupported source type %q", source.Type)
	}

	return nil
}
