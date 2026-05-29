package sync

import (
	"context"
	"testing"

	securityv1alpha1 "github.com/DavidHoenisch/coraza-operator/api/v1alpha1"
)

func TestFormatBlocklist(t *testing.T) {
	got := FormatBlocklist([]string{"192.0.2.1", "198.51.100.0/24"})
	want := "192.0.2.1\n198.51.100.0/24"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestStaticSyncer(t *testing.T) {
	syncer := StaticSyncer{
		Result: Result{
			IPs:       []string{"192.0.2.1"},
			CommitSHA: "static",
		},
	}

	result, err := syncer.Fetch(context.Background(), securityv1alpha1.IPBlockListSpec{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.IPs) != 1 {
		t.Fatalf("expected one IP, got %v", result.IPs)
	}
}
