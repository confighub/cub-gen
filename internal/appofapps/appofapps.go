package appofapps

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	ModeAuthoritative = "authoritative"
	ModeObservedOnly  = "observed-only"
)

type Application struct {
	Path                 string
	Name                 string
	SourceRepo           string
	SourcePath           string
	DestinationNamespace string
}

type Root struct {
	Application
	ChildApplications []Application
}

type ChildApplication struct {
	Name                 string
	Path                 string
	SourceRepo           string
	SourcePath           string
	DestinationNamespace string
	Reason               string
}

type Analysis struct {
	RootApplicationPath   string
	RootSourcePath        string
	ChildApplicationPaths []string
	Mode                  string
	ModeReason            string
	GeneratedApplications []ChildApplication
}

func FindRoots(repo string) ([]Root, error) {
	apps, err := findApplications(repo)
	if err != nil {
		return nil, err
	}

	byPath := map[string]Application{}
	for _, app := range apps {
		byPath[app.Path] = app
	}

	roots := make([]Root, 0, len(apps))
	for _, app := range apps {
		if strings.TrimSpace(app.SourcePath) == "" {
			continue
		}
		childDir := filepath.Join(repo, filepath.FromSlash(app.SourcePath))
		st, err := os.Stat(childDir)
		if err != nil || !st.IsDir() {
			continue
		}

		children := make([]Application, 0, 4)
		for _, candidate := range apps {
			if candidate.Path == app.Path {
				continue
			}
			if pathIsUnderDir(candidate.Path, app.SourcePath) {
				children = append(children, candidate)
			}
		}
		if len(children) == 0 {
			continue
		}

		sortApplications(children)
		roots = append(roots, Root{Application: app, ChildApplications: children})
	}

	sort.Slice(roots, func(i, j int) bool {
		if roots[i].Path != roots[j].Path {
			return roots[i].Path < roots[j].Path
		}
		return roots[i].Name < roots[j].Name
	})
	return roots, nil
}

func Analyze(repo, rootPath string) (Analysis, error) {
	root, err := findRoot(repo, rootPath)
	if err != nil {
		return Analysis{}, err
	}

	children := make([]ChildApplication, 0, len(root.ChildApplications))
	childPaths := make([]string, 0, len(root.ChildApplications))
	for _, child := range root.ChildApplications {
		childPaths = append(childPaths, child.Path)
		children = append(children, ChildApplication{
			Name:                 child.Name,
			Path:                 child.Path,
			SourceRepo:           child.SourceRepo,
			SourcePath:           child.SourcePath,
			DestinationNamespace: child.DestinationNamespace,
			Reason:               fmt.Sprintf("included by root Application source path %s", root.SourcePath),
		})
	}
	childPaths = uniqueSorted(childPaths)

	mode := ModeObservedOnly
	reason := "Root Application source path did not resolve to deterministic child Application inputs."
	if len(children) > 0 {
		mode = ModeAuthoritative
		reason = fmt.Sprintf("Root Application %s selects %d child Application(s) from %s.", root.Name, len(children), root.SourcePath)
	}

	return Analysis{
		RootApplicationPath:   root.Path,
		RootSourcePath:        root.SourcePath,
		ChildApplicationPaths: childPaths,
		Mode:                  mode,
		ModeReason:            reason,
		GeneratedApplications: children,
	}, nil
}

func ParseApplicationFile(repo, path string) (Application, error) {
	content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
	if err != nil {
		return Application{}, err
	}

	type rawApplication struct {
		APIVersion string `yaml:"apiVersion"`
		Kind       string `yaml:"kind"`
		Metadata   struct {
			Name string `yaml:"name"`
		} `yaml:"metadata"`
		Spec struct {
			Source struct {
				RepoURL string `yaml:"repoURL"`
				Path    string `yaml:"path"`
			} `yaml:"source"`
			Destination struct {
				Namespace string `yaml:"namespace"`
			} `yaml:"destination"`
		} `yaml:"spec"`
	}

	var raw rawApplication
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return Application{}, err
	}
	if !isArgoApplication(raw.APIVersion, raw.Kind) {
		return Application{}, fmt.Errorf("%s is not an Argo Application", path)
	}

	name := strings.TrimSpace(raw.Metadata.Name)
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	return Application{
		Path:                 filepath.ToSlash(path),
		Name:                 name,
		SourceRepo:           strings.TrimSpace(raw.Spec.Source.RepoURL),
		SourcePath:           filepath.ToSlash(strings.TrimSpace(raw.Spec.Source.Path)),
		DestinationNamespace: strings.TrimSpace(raw.Spec.Destination.Namespace),
	}, nil
}

func IsApplicationFile(repo, path string) bool {
	_, err := ParseApplicationFile(repo, path)
	return err == nil
}

func findRoot(repo, rootPath string) (Root, error) {
	roots, err := FindRoots(repo)
	if err != nil {
		return Root{}, err
	}
	normalized := filepath.ToSlash(strings.TrimSpace(rootPath))
	for _, root := range roots {
		if root.Path == normalized {
			return root, nil
		}
	}
	return Root{}, fmt.Errorf("no app-of-apps root matched %s", rootPath)
}

func findApplications(repo string) ([]Application, error) {
	apps := make([]Application, 0, 8)
	err := filepath.WalkDir(repo, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() && shouldSkipDir(d.Name()) {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		rel, err := filepath.Rel(repo, path)
		if err != nil {
			return err
		}
		app, err := ParseApplicationFile(repo, filepath.ToSlash(rel))
		if err != nil {
			return nil
		}
		apps = append(apps, app)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sortApplications(apps)
	return apps, nil
}

func isArgoApplication(apiVersion, kind string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(apiVersion)), "argoproj.io") &&
		strings.EqualFold(strings.TrimSpace(kind), "Application")
}

func pathIsUnderDir(path, dir string) bool {
	path = filepath.ToSlash(strings.Trim(path, "/"))
	dir = filepath.ToSlash(strings.Trim(dir, "/"))
	if path == "" || dir == "" || path == dir {
		return false
	}
	return strings.HasPrefix(path, dir+"/")
}

func sortApplications(apps []Application) {
	sort.Slice(apps, func(i, j int) bool {
		if apps[i].Name != apps[j].Name {
			return apps[i].Name < apps[j].Name
		}
		return apps[i].Path < apps[j].Path
	})
}

func uniqueSorted(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(filepath.ToSlash(item))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", ".hg", ".svn", "node_modules", "vendor", ".terraform":
		return true
	default:
		return false
	}
}
