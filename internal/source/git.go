package source

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	securityv1alpha1 "github.com/DavidHoenisch/coraza-operator/api/v1alpha1"
)

const defaultGitBranch = "main"

// GitFetcher loads blocklists from a Git repository.
type GitFetcher struct{}

func NewGitFetcher() *GitFetcher {
	return &GitFetcher{}
}

func (f *GitFetcher) Type() securityv1alpha1.SourceType {
	return securityv1alpha1.SourceTypeGit
}

func (f *GitFetcher) Fetch(ctx context.Context, source securityv1alpha1.SourceSpec) (Result, error) {
	if source.Git == nil {
		return Result{}, fmt.Errorf("git config is required")
	}

	branch := source.Git.Branch
	if branch == "" {
		branch = defaultGitBranch
	}

	dir, err := os.MkdirTemp("", "coraza-blocklist-git-*")
	if err != nil {
		return Result{}, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	repo, err := git.PlainCloneContext(ctx, dir, false, &git.CloneOptions{
		URL:           source.Git.URL,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		Depth:         1,
		SingleBranch:  true,
	})
	if err != nil {
		return Result{}, fmt.Errorf("clone repository: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return Result{}, fmt.Errorf("resolve HEAD: %w", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(source.Git.Path)))
	if err != nil {
		return Result{}, fmt.Errorf("read blocklist file: %w", err)
	}

	return Result{
		IPs:      ParseBlocklist(content),
		Revision: head.Hash().String(),
	}, nil
}
