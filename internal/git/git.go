package git

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type RepoManager struct {
	BaseDir string
}

func NewRepoManager(baseDir string) (*RepoManager, error) {
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("repository directory does not exist: %s", baseDir)
	}

	absPath, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	return &RepoManager{BaseDir: absPath}, nil
}

type RepoInfo struct {
	Name string
	Path string
}

func (rm *RepoManager) ListRepositories() ([]RepoInfo, error) {
	entries, err := os.ReadDir(rm.BaseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var repos []RepoInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		repoPath := filepath.Join(rm.BaseDir, entry.Name())
		if _, err := git.PlainOpen(repoPath); err == nil {
			repos = append(repos, RepoInfo{
				Name: entry.Name(),
				Path: repoPath,
			})
		}
	}

	return repos, nil
}

func (rm *RepoManager) OpenRepository(name string) (*git.Repository, error) {
	if strings.Contains(name, "..") || strings.Contains(name, "/") {
		return nil, fmt.Errorf("invalid repository name: %s", name)
	}

	repoPath := filepath.Join(rm.BaseDir, name)
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	return repo, nil
}

type BranchInfo struct {
	Name string
	Hash string
}

type TagInfo struct {
	Name string
	Hash string
}

func GetHeadBranch(repo *git.Repository) (string, error) {
	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}
	return head.Name().Short(), nil
}

func GetBranches(repo *git.Repository) ([]BranchInfo, error) {
	var branches []BranchInfo

	branchIter, err := repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("failed to get branches: %w", err)
	}

	err = branchIter.ForEach(func(ref *plumbing.Reference) error {
		branches = append(branches, BranchInfo{
			Name: ref.Name().Short(),
			Hash: ref.Hash().String(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return branches, nil
}

func GetTags(repo *git.Repository) ([]TagInfo, error) {
	var tags []TagInfo

	tagIter, err := repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}

	err = tagIter.ForEach(func(ref *plumbing.Reference) error {
		tags = append(tags, TagInfo{
			Name: ref.Name().Short(),
			Hash: ref.Hash().String(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return tags, nil
}

func ResolveRef(repo *git.Repository, refName string) (string, error) {
	if plumbing.IsHash(refName) {
		hash := plumbing.NewHash(refName)
		_, err := repo.CommitObject(hash)
		if err == nil {
			return hash.String(), nil
		}
	}

	ref, err := repo.Reference(plumbing.NewBranchReferenceName(refName), true)
	if err == nil {
		return ref.Hash().String(), nil
	}

	ref, err = repo.Reference(plumbing.NewTagReferenceName(refName), true)
	if err == nil {
		return ref.Hash().String(), nil
	}

	return "", fmt.Errorf("reference not found: %s", refName)
}

type TreeEntry struct {
	Name  string
	IsDir bool
	Size  int64
	Mode  string
}

func GetTree(repo *git.Repository, hashStr, path string) ([]TreeEntry, error) {
	hash := plumbing.NewHash(hashStr)
	commit, err := repo.CommitObject(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	if path != "" && path != "/" {
		path = strings.Trim(path, "/")
		tree, err = tree.Tree(path)
		if err != nil {
			return nil, fmt.Errorf("failed to get subtree: %w", err)
		}
	}

	var entries []TreeEntry
	for _, entry := range tree.Entries {
		te := TreeEntry{
			Name:  entry.Name,
			IsDir: entry.Mode == filemode.Dir,
			Mode:  entry.Mode.String(),
		}

		if !te.IsDir {
			file, err := tree.File(entry.Name)
			if err == nil {
				te.Size = file.Size
			}
		}

		entries = append(entries, te)
	}

	return entries, nil
}

func GetFileList(repo *git.Repository, hashStr string) ([]string, error) {
	hash := plumbing.NewHash(hashStr)
	commit, err := repo.CommitObject(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	var files []string
	err = tree.Files().ForEach(func(file *object.File) error {
		files = append(files, file.Name)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate files: %w", err)
	}

	return files, nil
}

func GetBlob(repo *git.Repository, hashStr, path string) (string, int64, bool, error) {
	hash := plumbing.NewHash(hashStr)
	commit, err := repo.CommitObject(hash)
	if err != nil {
		return "", 0, false, fmt.Errorf("failed to get commit: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return "", 0, false, fmt.Errorf("failed to get tree: %w", err)
	}

	path = strings.Trim(path, "/")
	file, err := tree.File(path)
	if err != nil {
		return "", 0, false, fmt.Errorf("failed to get file: %w", err)
	}

	reader, err := file.Reader()
	if err != nil {
		return "", 0, false, fmt.Errorf("failed to read file: %w", err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return "", 0, false, fmt.Errorf("failed to read file content: %w", err)
	}

	// TODO: Improve binary file detection
	// Current implementation: simple NUL byte check
	// Consider: MIME type detection, file extension, or more sophisticated heuristics
	isBinary := false
	for _, b := range content {
		if b == 0 {
			isBinary = true
			break
		}
	}

	return string(content), file.Size, isBinary, nil
}
