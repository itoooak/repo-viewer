package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"repo-viewer/internal/git"
	"repo-viewer/internal/view"
)

type Handler struct {
	repoManager *git.RepoManager
	renderer    *view.Renderer
}

type PathSegment struct {
	Name string
	Path string
}

func New(repoDir string) (*Handler, error) {
	repoManager, err := git.NewRepoManager(repoDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize RepoManager: %w", err)
	}

	renderer, err := view.NewRenderer("templates")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Renderer: %w", err)
	}

	return &Handler{
		repoManager: repoManager,
		renderer:    renderer,
	}, nil
}

func (h *Handler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	repos, err := h.repoManager.ListRepositories()
	if err != nil {
		h.renderer.RenderError(w, http.StatusInternalServerError,
			fmt.Sprintf("Failed to list repositories: %v", err))
		return
	}

	data := map[string]interface{}{
		"Title": "Repositories",
		"Repos": repos,
	}
	if err := h.renderer.Render(w, "index", data); err != nil {
		h.renderer.RenderError(w, http.StatusInternalServerError,
			fmt.Sprintf("Failed to render page: %v", err))
	}
}

func (h *Handler) HandleRepo(w http.ResponseWriter, r *http.Request) {
	repoName := r.PathValue("repo")

	repo, err := h.repoManager.OpenRepository(repoName)
	if err != nil {
		h.renderer.RenderError(w, http.StatusNotFound,
			fmt.Sprintf("Repository not found: %v", err))
		return
	}

	head, err := git.GetHeadBranch(repo)
	if err != nil {
		h.renderer.RenderError(w, http.StatusInternalServerError,
			fmt.Sprintf("Failed to get HEAD: %v", err))
		return
	}

	branches, err := git.GetBranches(repo)
	if err != nil {
		h.renderer.RenderError(w, http.StatusInternalServerError,
			fmt.Sprintf("Failed to get branches: %v", err))
		return
	}

	tags, err := git.GetTags(repo)
	if err != nil {
		h.renderer.RenderError(w, http.StatusInternalServerError,
			fmt.Sprintf("Failed to get tags: %v", err))
		return
	}

	var readmeContent string
	hash, err := git.ResolveRef(repo, head)
	if err == nil {
		for _, readmeName := range []string{"README.md", "README", "readme.md", "readme"} {
			content, _, isBinary, err := git.GetBlob(repo, hash, readmeName)
			if err == nil && !isBinary {
				readmeContent = content
				break
			}
		}
	}

	data := map[string]interface{}{
		"Title":         repoName + " - repo-viewer",
		"RepoName":      repoName,
		"Head":          head,
		"Branches":      branches,
		"Tags":          tags,
		"ReadmeContent": readmeContent,
	}

	if err := h.renderer.Render(w, "repo", data); err != nil {
		h.renderer.RenderError(w, http.StatusInternalServerError,
			fmt.Sprintf("Failed to render page: %v", err))
	}
}

func (h *Handler) HandleTree(w http.ResponseWriter, r *http.Request) {
	repoName := r.PathValue("repo")
	ref := r.PathValue("ref")
	treePath := r.PathValue("path")

	repo, err := h.repoManager.OpenRepository(repoName)
	if err != nil {
		h.renderer.RenderError(w, http.StatusNotFound,
			fmt.Sprintf("Repository not found: %v", err))
		return
	}

	hash, err := git.ResolveRef(repo, ref)
	if err != nil {
		h.renderer.RenderError(w, http.StatusNotFound,
			fmt.Sprintf("Reference not found: %v", err))
		return
	}

	entries, err := git.GetTree(repo, hash, treePath)
	if err != nil {
		h.renderer.RenderError(w, http.StatusNotFound,
			fmt.Sprintf("Failed to get tree: %v", err))
		return
	}

	pathParts := []string{}
	if treePath != "" {
		pathParts = strings.Split(treePath, "/")
	}

	parentPath := ""
	if len(pathParts) > 1 {
		parentPath = "/" + strings.Join(pathParts[:len(pathParts)-1], "/")
	}

	pathSegments := make([]PathSegment, 0, len(pathParts))
	for i := 0; i < len(pathParts)-1; i++ {
		fullPath := strings.Join(pathParts[:i+1], "/")
		pathSegments = append(pathSegments, PathSegment{
			Name: pathParts[i],
			Path: fullPath,
		})
	}

	currentName := ""
	if len(pathParts) > 0 {
		currentName = pathParts[len(pathParts)-1]
	}

	data := map[string]interface{}{
		"Title":        repoName + " / " + ref + " - repo-viewer",
		"RepoName":     repoName,
		"Ref":          ref,
		"Path":         treePath,
		"PathSegments": pathSegments,
		"CurrentName":  currentName,
		"ParentPath":   parentPath,
		"Entries":      entries,
	}

	if err := h.renderer.Render(w, "tree", data); err != nil {
		h.renderer.RenderError(w, http.StatusInternalServerError,
			fmt.Sprintf("Failed to render page: %v", err))
	}
}

func (h *Handler) HandleBlob(w http.ResponseWriter, r *http.Request) {
	repoName := r.PathValue("repo")
	ref := r.PathValue("ref")
	blobPath := r.PathValue("path")

	repo, err := h.repoManager.OpenRepository(repoName)
	if err != nil {
		h.renderer.RenderError(w, http.StatusNotFound,
			fmt.Sprintf("Repository not found: %v", err))
		return
	}

	hash, err := git.ResolveRef(repo, ref)
	if err != nil {
		h.renderer.RenderError(w, http.StatusNotFound,
			fmt.Sprintf("Reference not found: %v", err))
		return
	}

	content, size, isBinary, err := git.GetBlob(repo, hash, blobPath)
	if err != nil {
		h.renderer.RenderError(w, http.StatusNotFound,
			fmt.Sprintf("Failed to get file: %v", err))
		return
	}

	pathParts := strings.Split(blobPath, "/")

	pathSegments := make([]PathSegment, 0, len(pathParts))
	for i := 0; i < len(pathParts)-1; i++ {
		fullPath := strings.Join(pathParts[:i+1], "/")
		pathSegments = append(pathSegments, PathSegment{
			Name: pathParts[i],
			Path: fullPath,
		})
	}

	currentName := ""
	if len(pathParts) > 0 {
		currentName = pathParts[len(pathParts)-1]
	}

	data := map[string]interface{}{
		"Title":        blobPath + " - repo-viewer",
		"RepoName":     repoName,
		"Ref":          ref,
		"Path":         blobPath,
		"PathSegments": pathSegments,
		"CurrentName":  currentName,
		"Content":      content,
		"Size":         size,
		"IsBinary":     isBinary,
	}

	if err := h.renderer.Render(w, "blob", data); err != nil {
		h.renderer.RenderError(w, http.StatusInternalServerError,
			fmt.Sprintf("Failed to render page: %v", err))
	}
}

func (h *Handler) HandleNotFound(w http.ResponseWriter, r *http.Request) {
	h.renderer.RenderError(w, http.StatusNotFound, "Page not found")
}

func (h *Handler) HandleFileListAPI(w http.ResponseWriter, r *http.Request) {
	repoName := r.PathValue("repo")
	ref := r.URL.Query().Get("ref")
	if ref == "" {
		http.Error(w, "ref parameter is required", http.StatusBadRequest)
		return
	}

	repo, err := h.repoManager.OpenRepository(repoName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Repository not found: %v", err), http.StatusNotFound)
		return
	}

	hash, err := git.ResolveRef(repo, ref)
	if err != nil {
		http.Error(w, fmt.Sprintf("Reference not found: %v", err), http.StatusNotFound)
		return
	}

	files, err := git.GetFileList(repo, hash)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get file list: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"ref":   ref,
		"files": files,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}
