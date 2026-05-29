package sync

import (
	"context"
	"strings"

	securityv1alpha1 "github.com/DavidHoenisch/coraza-operator/api/v1alpha1"
	"github.com/DavidHoenisch/coraza-operator/internal/source"
)

// Result holds the outcome of a blocklist sync.
type Result struct {
	IPs       []string
	CommitSHA string
}

// Syncer loads blocked IPs for an IPBlockList spec.
type Syncer interface {
	Fetch(ctx context.Context, spec securityv1alpha1.IPBlockListSpec) (Result, error)
}

// DefaultSyncer fetches from all configured sources using a source registry.
type DefaultSyncer struct {
	Registry *source.Registry
}

func NewDefaultSyncer(registry *source.Registry) *DefaultSyncer {
	if registry == nil {
		registry = source.DefaultRegistry()
	}
	return &DefaultSyncer{Registry: registry}
}

func (s *DefaultSyncer) Fetch(ctx context.Context, spec securityv1alpha1.IPBlockListSpec) (Result, error) {
	result, err := s.Registry.FetchAll(ctx, spec.Sources)
	if err != nil {
		return Result{}, err
	}

	return Result{
		IPs:       result.IPs,
		CommitSHA: result.Revision,
	}, nil
}

// StaticSyncer is useful in tests to avoid network access.
type StaticSyncer struct {
	Result Result
	Err    error
}

func (s StaticSyncer) Fetch(_ context.Context, _ securityv1alpha1.IPBlockListSpec) (Result, error) {
	return s.Result, s.Err
}

// FormatBlocklist renders IPs as a newline-delimited blocklist file.
func FormatBlocklist(ips []string) string {
	return strings.Join(ips, "\n")
}
