package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	securityv1alpha1 "github.com/DavidHoenisch/coraza-operator/api/v1alpha1"
)

func TestValidateSourceGit(t *testing.T) {
	err := ValidateSource(securityv1alpha1.SourceSpec{Type: securityv1alpha1.SourceTypeGit})
	if err == nil {
		t.Fatal("expected validation error for missing git config")
	}
}

func TestValidateSourceHTTP(t *testing.T) {
	err := ValidateSource(securityv1alpha1.SourceSpec{
		Type: securityv1alpha1.SourceTypeHTTP,
		HTTP: &securityv1alpha1.HTTPSourceSpec{URL: "https://example.com/list.txt"},
	})
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestRegistryFetchAllMergesSources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("ETag", `"http-revision"`)
		_, _ = w.Write([]byte("203.0.113.5\n"))
	}))
	defer server.Close()

	registry := DefaultRegistry()
	result, err := registry.FetchAll(context.Background(), []securityv1alpha1.SourceSpec{
		{
			Type: securityv1alpha1.SourceTypeHTTP,
			HTTP: &securityv1alpha1.HTTPSourceSpec{URL: server.URL},
		},
		{
			Type: securityv1alpha1.SourceTypeHTTP,
			HTTP: &securityv1alpha1.HTTPSourceSpec{URL: server.URL},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.IPs) != 1 {
		t.Fatalf("expected deduplicated IPs, got %v", result.IPs)
	}
	if result.Revision == "" {
		t.Fatal("expected combined revision")
	}
}

func TestGitFetcherFetch(t *testing.T) {
	repoURL := initGitRepo(t, "192.0.2.1\n198.51.100.0/24\n")

	fetcher := NewGitFetcher()
	result, err := fetcher.Fetch(context.Background(), securityv1alpha1.SourceSpec{
		Type: securityv1alpha1.SourceTypeGit,
		Git: &securityv1alpha1.GitSourceSpec{
			URL:  repoURL,
			Path: "blocked-ips.txt",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.IPs) != 2 {
		t.Fatalf("expected 2 IPs, got %v", result.IPs)
	}
	if result.Revision == "" {
		t.Fatal("expected commit revision")
	}
}

func initGitRepo(t *testing.T, blocklist string) string {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "test")

	if err := os.WriteFile(filepath.Join(dir, "blocked-ips.txt"), []byte(blocklist), 0o644); err != nil {
		t.Fatalf("write blocklist: %v", err)
	}

	runGit(t, dir, "add", "blocked-ips.txt")
	runGit(t, dir, "commit", "-m", "add blocklist")
	runGit(t, dir, "branch", "-M", "main")

	absDir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("resolve repo path: %v", err)
	}

	return "file://" + filepath.ToSlash(absDir)
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}
