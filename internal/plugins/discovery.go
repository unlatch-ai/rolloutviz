package plugins

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

const (
	DiscoverySchemaVersion  = 1
	DefaultDiscoveryDepth   = 2
	DefaultDiscoveryCount   = 128
	DefaultManifestBytes    = 256 << 10
	maxDirectoryEntries     = 4096
	maxDiscoveryDirectories = 512
)

// DiscoveryRoot is an explicitly allowed directory. Discovery never consults
// PATH and never executes, probes, or trusts a plugin.
type DiscoveryRoot struct {
	Path   string
	Source string
}

type DiscoveryOptions struct {
	Roots            []DiscoveryRoot
	TrustEntries     []TrustEntry
	MaxDepth         int
	MaxPlugins       int
	MaxManifestBytes int64
}

type DiscoveredPlugin struct {
	Rank         int      `json:"rank"`
	Source       string   `json:"source"`
	Path         string   `json:"path"`
	ManifestPath string   `json:"manifest_path"`
	Name         string   `json:"name,omitempty"`
	Kind         string   `json:"kind,omitempty"`
	APIVersion   string   `json:"api_version,omitempty"`
	Version      string   `json:"version,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Description  string   `json:"description,omitempty"`
	Status       string   `json:"status"`
	Digest       string   `json:"digest,omitempty"`
	Error        string   `json:"error,omitempty"`
}

type DiscoveryIssue struct {
	Root  string `json:"root"`
	Code  string `json:"code"`
	Error string `json:"error"`
}

type DiscoveryResult struct {
	SchemaVersion int                `json:"schema_version"`
	Plugins       []DiscoveredPlugin `json:"plugins"`
	Issues        []DiscoveryIssue   `json:"issues"`
}

// Discover inventories bounded manifest metadata under explicit roots. It
// intentionally does not call any plugin command. Content is hashed only for a
// path already present in the trust store, so untrusted trees are not walked.
func Discover(options DiscoveryOptions) DiscoveryResult {
	if options.MaxDepth <= 0 {
		options.MaxDepth = DefaultDiscoveryDepth
	}
	if options.MaxPlugins <= 0 {
		options.MaxPlugins = DefaultDiscoveryCount
	}
	if options.MaxManifestBytes <= 0 {
		options.MaxManifestBytes = DefaultManifestBytes
	}
	result := DiscoveryResult{SchemaVersion: DiscoverySchemaVersion, Plugins: []DiscoveredPlugin{}, Issues: []DiscoveryIssue{}}
	trusted := make(map[string]string, len(options.TrustEntries))
	for _, entry := range options.TrustEntries {
		trusted[cleanResolved(entry.Path)] = entry.Digest
	}
	seen := map[string]bool{}
	for _, root := range options.Roots {
		if len(result.Plugins) >= options.MaxPlugins {
			result.Issues = append(result.Issues, DiscoveryIssue{Root: root.Path, Code: "plugin_limit", Error: fmt.Sprintf("discovery is limited to %d manifests", options.MaxPlugins)})
			break
		}
		manifests, issue := discoverManifestPaths(root.Path, options.MaxDepth, options.MaxPlugins-len(result.Plugins))
		if issue != nil {
			result.Issues = append(result.Issues, *issue)
			continue
		}
		for _, manifestPath := range manifests {
			pluginPath := filepath.Dir(manifestPath)
			resolved := cleanResolved(pluginPath)
			if seen[resolved] {
				continue
			}
			seen[resolved] = true
			item := readDiscoveredManifest(root.Source, resolved, manifestPath, options.MaxManifestBytes)
			if digest, ok := trusted[resolved]; ok && item.Status != "invalid" {
				loaded, err := Load(resolved)
				if err != nil {
					item.Status, item.Error = "invalid", err.Error()
				} else {
					item.Digest = loaded.Digest
					if loaded.Digest == digest {
						item.Status = "trusted"
					} else {
						item.Status = "changed"
						item.Error = "plugin contents changed since trust was granted"
					}
				}
			}
			result.Plugins = append(result.Plugins, item)
		}
	}
	for index := range result.Plugins {
		result.Plugins[index].Rank = index + 1
	}
	return result
}

func discoverManifestPaths(root string, maxDepth, remaining int) ([]string, *DiscoveryIssue) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, &DiscoveryIssue{Root: root, Code: "root_invalid", Error: err.Error()}
	}
	info, err := os.Lstat(abs)
	if errors.Is(err, os.ErrNotExist) {
		return []string{}, nil
	}
	if err != nil {
		return nil, &DiscoveryIssue{Root: abs, Code: "root_unreadable", Error: err.Error()}
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, &DiscoveryIssue{Root: abs, Code: "root_invalid", Error: "discovery root must be a real directory, not a symlink"}
	}
	type queuedDir struct {
		path  string
		depth int
	}
	queue := []queuedDir{{abs, 0}}
	candidates := map[string]string{}
	visited := 0
	for len(queue) > 0 {
		visited++
		if visited > maxDiscoveryDirectories {
			return nil, &DiscoveryIssue{Root: abs, Code: "directory_limit", Error: fmt.Sprintf("discovery is limited to %d directories per root", maxDiscoveryDirectories)}
		}
		current := queue[0]
		queue = queue[1:]
		entries, err := boundedReadDir(current.path)
		if err != nil {
			return nil, &DiscoveryIssue{Root: abs, Code: "root_unreadable", Error: err.Error()}
		}
		for _, entry := range entries {
			path := filepath.Join(current.path, entry.Name())
			if entry.Type()&os.ModeSymlink != 0 {
				continue
			}
			if entry.IsDir() {
				if current.depth < maxDepth {
					queue = append(queue, queuedDir{path, current.depth + 1})
				}
				continue
			}
			if isManifestName(entry.Name()) && entry.Type().IsRegular() {
				dir := filepath.Dir(path)
				if previous, ok := candidates[dir]; !ok || manifestPriority(entry.Name()) < manifestPriority(filepath.Base(previous)) {
					candidates[dir] = path
				}
			}
		}
		sort.Slice(queue, func(i, j int) bool { return queue[i].path < queue[j].path })
	}
	manifests := make([]string, 0, len(candidates))
	for _, path := range candidates {
		manifests = append(manifests, path)
	}
	sort.Strings(manifests)
	if len(manifests) > remaining {
		return nil, &DiscoveryIssue{Root: abs, Code: "plugin_limit", Error: fmt.Sprintf("discovery is limited to %d additional manifests", remaining)}
	}
	return manifests, nil
}

func boundedReadDir(path string) ([]fs.DirEntry, error) {
	dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	entries, err := dir.ReadDir(maxDirectoryEntries + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(entries) > maxDirectoryEntries {
		return nil, fmt.Errorf("directory has more than %d entries", maxDirectoryEntries)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

func readDiscoveredManifest(source, pluginPath, manifestPath string, maxBytes int64) DiscoveredPlugin {
	item := DiscoveredPlugin{Source: source, Path: pluginPath, ManifestPath: manifestPath, Status: "untrusted"}
	info, err := os.Lstat(manifestPath)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		item.Status = "invalid"
		if err != nil {
			item.Error = err.Error()
		} else {
			item.Error = "manifest must be a regular file, not a symlink"
		}
		return item
	}
	file, err := os.Open(manifestPath)
	if err != nil {
		item.Status, item.Error = "invalid", err.Error()
		return item
	}
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	closeErr := file.Close()
	if err != nil || closeErr != nil {
		item.Status = "invalid"
		item.Error = errors.Join(err, closeErr).Error()
		return item
	}
	if int64(len(data)) > maxBytes {
		item.Status, item.Error = "invalid", fmt.Sprintf("manifest exceeds %d bytes", maxBytes)
		return item
	}
	manifest, err := ParseManifest(data)
	if err == nil {
		err = manifest.Validate()
	}
	if err != nil {
		item.Status, item.Error = "invalid", err.Error()
		return item
	}
	item.Name = manifest.Name
	item.Kind = manifest.Kind
	item.APIVersion = manifest.APIVersion
	item.Version = manifest.Version
	item.Capabilities = append([]string(nil), manifest.Capabilities...)
	item.Description = manifest.Description
	return item
}

func cleanResolved(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return filepath.Clean(abs)
}

func isManifestName(name string) bool {
	for _, candidate := range []string{ManifestName, "rlviz-plugin.yml", "rlviz-plugin.json", "plugin.yaml", "plugin.yml", "plugin.json"} {
		if name == candidate {
			return true
		}
	}
	return false
}

func manifestPriority(name string) int {
	for index, candidate := range []string{ManifestName, "rlviz-plugin.yml", "rlviz-plugin.json", "plugin.yaml", "plugin.yml", "plugin.json"} {
		if name == candidate {
			return index
		}
	}
	return 100
}
