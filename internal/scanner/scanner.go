package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"gitlab.flexinfer.ai/services/tech-radar/pkg/types"
)

// Scanner scans directories for technologies.
type Scanner struct {
	config types.ScanConfig
}

// New creates a new Scanner.
func New(config types.ScanConfig) *Scanner {
	return &Scanner{
		config: config,
	}
}

// Scan scans a directory and returns discovered technologies.
func (s *Scanner) Scan(root string) (*types.ScanResult, error) {
	result := &types.ScanResult{
		Path:         root,
		Technologies: []types.DiscoveredTech{},
		Files:        make(map[string][]string),
	}
	seenTechs := make(map[string]int)

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && d.Name() != "." && d.Name() != ".." {
				return filepath.SkipDir
			}
			if d.Name() == "node_modules" || d.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		filename := d.Name()
		if !s.shouldScanPath(path, filename) {
			return nil
		}

		techs, err := s.parseFile(path, filename)
		if err != nil {
			return nil
		}

		if len(techs) == 0 {
			return nil
		}

		relPath, _ := filepath.Rel(root, path)
		var techNames []string
		for _, tech := range techs {
			tech = s.normalizeTech(tech)
			if tech.Name == "" {
				continue
			}
			if tech.EvidencePath == path {
				tech.EvidencePath = filepath.ToSlash(relPath)
			}
			techNames = appendIfMissing(techNames, tech.Name)

			key := strings.ToLower(strings.TrimSpace(tech.Name))
			if idx, ok := seenTechs[key]; ok {
				result.Technologies[idx] = mergeTech(result.Technologies[idx], tech)
				continue
			}

			seenTechs[key] = len(result.Technologies)
			result.Technologies = append(result.Technologies, tech)
		}
		result.Files[relPath] = techNames
		return nil
	})

	return result, err
}

func (s *Scanner) shouldScanPath(path, filename string) bool {
	for _, f := range s.config.Files {
		if f == filename {
			return true
		}
	}

	return isADRMarkdownPath(path)
}

func (s *Scanner) parseFile(path, filename string) ([]types.DiscoveredTech, error) {
	if isADRMarkdownPath(path) {
		return s.parseADRMarkdown(path)
	}

	switch filename {
	case "package.json":
		return s.parsePackageJSON(path)
	case "go.mod":
		return s.parseGoMod(path)
	case "requirements.txt":
		return s.parseRequirementsTxt(path)
	case "Cargo.toml":
		return s.parseCargoToml(path)
	case "pyproject.toml":
		return s.parsePyprojectToml(path)
	case ".mcp.json", "mcp.json", "claude_desktop_config.json", "cline_mcp_settings.json":
		return s.parseMCPConfigJSON(path, filename)
	}

	return nil, nil
}

func (s *Scanner) normalizeTech(tech types.DiscoveredTech) types.DiscoveredTech {
	if hint, ok := s.config.Mappings[strings.ToLower(strings.TrimSpace(tech.Name))]; ok {
		tech.Quadrant = hint.Quadrant
		if hint.Name != "" {
			tech.Name = hint.Name
		}
	}
	return tech
}

func (s *Scanner) parsePackageJSON(path string) ([]types.DiscoveredTech, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	var techs []types.DiscoveredTech
	for name, version := range pkg.Dependencies {
		techs = append(techs, types.DiscoveredTech{
			Name:    name,
			Version: version,
			Source:  "package.json",
		})
	}
	for name, version := range pkg.DevDependencies {
		techs = append(techs, types.DiscoveredTech{
			Name:    name,
			Version: version,
			Source:  "package.json",
		})
	}

	return techs, nil
}

func (s *Scanner) parseGoMod(path string) ([]types.DiscoveredTech, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var techs []types.DiscoveredTech
	inRequire := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		if strings.HasPrefix(line, "require (") {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}

		if inRequire {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				techs = append(techs, types.DiscoveredTech{
					Name:    parts[0],
					Version: parts[1],
					Source:  "go.mod",
				})
			}
		} else if strings.HasPrefix(line, "require ") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				techs = append(techs, types.DiscoveredTech{
					Name:    parts[1],
					Version: parts[2],
					Source:  "go.mod",
				})
			}
		}
	}

	return techs, nil
}

func (s *Scanner) parseRequirementsTxt(path string) ([]types.DiscoveredTech, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var techs []types.DiscoveredTech

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, "==")
		name := parts[0]
		version := ""
		if len(parts) > 1 {
			version = parts[1]
		} else {
			separators := []string{">=", "<=", ">", "<", "~="}
			for _, sep := range separators {
				if strings.Contains(name, sep) {
					subParts := strings.Split(name, sep)
					name = subParts[0]
					version = sep + subParts[1]
					break
				}
			}
		}
		name = strings.TrimSpace(name)

		if name != "" {
			techs = append(techs, types.DiscoveredTech{
				Name:    name,
				Version: version,
				Source:  "requirements.txt",
			})
		}
	}

	return techs, nil
}

func (s *Scanner) parseCargoToml(path string) ([]types.DiscoveredTech, error) {
	var cargo struct {
		Dependencies    map[string]interface{} `toml:"dependencies"`
		DevDependencies map[string]interface{} `toml:"dev-dependencies"`
		BuildDeps       map[string]interface{} `toml:"build-dependencies"`
	}

	if _, err := toml.DecodeFile(path, &cargo); err != nil {
		return nil, err
	}

	var techs []types.DiscoveredTech
	addDeps := func(deps map[string]interface{}) {
		for name, val := range deps {
			version := ""
			switch v := val.(type) {
			case string:
				version = v
			case map[string]interface{}:
				if ver, ok := v["version"].(string); ok {
					version = ver
				}
			}
			techs = append(techs, types.DiscoveredTech{
				Name:    name,
				Version: version,
				Source:  "Cargo.toml",
			})
		}
	}

	addDeps(cargo.Dependencies)
	addDeps(cargo.DevDependencies)
	addDeps(cargo.BuildDeps)

	return techs, nil
}

func (s *Scanner) parsePyprojectToml(path string) ([]types.DiscoveredTech, error) {
	var pyproject struct {
		Project struct {
			Dependencies []string `toml:"dependencies"`
		} `toml:"project"`
		Tool struct {
			Poetry struct {
				Dependencies    map[string]interface{} `toml:"dependencies"`
				DevDependencies map[string]interface{} `toml:"dev-dependencies"`
				Group           map[string]struct {
					Dependencies map[string]interface{} `toml:"dependencies"`
				} `toml:"group"`
			} `toml:"poetry"`
		} `toml:"tool"`
	}

	if _, err := toml.DecodeFile(path, &pyproject); err != nil {
		return nil, err
	}

	var techs []types.DiscoveredTech

	for _, dep := range pyproject.Project.Dependencies {
		name := dep
		version := ""
		for _, sep := range []string{"==", ">=", "<=", ">", "<", "~=", " "} {
			if idx := strings.Index(dep, sep); idx != -1 {
				name = strings.TrimSpace(dep[:idx])
				version = strings.TrimSpace(dep[idx:])
				break
			}
		}
		techs = append(techs, types.DiscoveredTech{
			Name:    name,
			Version: version,
			Source:  "pyproject.toml",
		})
	}

	addPoetryDeps := func(deps map[string]interface{}) {
		for name, val := range deps {
			if name == "python" {
				continue
			}
			version := ""
			switch v := val.(type) {
			case string:
				version = v
			case map[string]interface{}:
				if ver, ok := v["version"].(string); ok {
					version = ver
				}
			}
			techs = append(techs, types.DiscoveredTech{
				Name:    name,
				Version: version,
				Source:  "pyproject.toml",
			})
		}
	}

	addPoetryDeps(pyproject.Tool.Poetry.Dependencies)
	addPoetryDeps(pyproject.Tool.Poetry.DevDependencies)
	for _, group := range pyproject.Tool.Poetry.Group {
		addPoetryDeps(group.Dependencies)
	}

	return techs, nil
}

func (s *Scanner) parseADRMarkdown(path string) ([]types.DiscoveredTech, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	title := adrTitle(content, path)
	mentions := s.detectADRTechMentions(content)
	if len(mentions) == 0 {
		return nil, nil
	}

	techs := make([]types.DiscoveredTech, 0, len(mentions))
	for _, mention := range mentions {
		techs = append(techs, types.DiscoveredTech{
			Name:          mention.Name,
			Source:        "adr",
			Quadrant:      mention.Quadrant,
			EvidencePath:  path,
			EvidenceTitle: title,
		})
	}

	return techs, nil
}

func (s *Scanner) detectADRTechMentions(content string) []types.EntryHint {
	normalized := strings.ToLower(content)
	seen := make(map[string]bool)
	mentions := make([]types.EntryHint, 0)

	for key, hint := range s.config.Mappings {
		candidates := []string{strings.ToLower(strings.TrimSpace(key))}
		if hint.Name != "" {
			candidates = append(candidates, strings.ToLower(strings.TrimSpace(hint.Name)))
		}

		matched := false
		for _, candidate := range candidates {
			if candidate == "" {
				continue
			}
			if strings.Contains(normalized, candidate) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}

		name := hint.Name
		if name == "" {
			name = key
		}
		lookup := strings.ToLower(strings.TrimSpace(name))
		if seen[lookup] {
			continue
		}
		seen[lookup] = true
		mentions = append(mentions, types.EntryHint{Name: name, Quadrant: hint.Quadrant})
	}

	sort.Slice(mentions, func(i, j int) bool {
		return strings.ToLower(mentions[i].Name) < strings.ToLower(mentions[j].Name)
	})

	return mentions
}

func isADRMarkdownPath(path string) bool {
	lower := strings.ToLower(filepath.ToSlash(path))
	if !strings.HasSuffix(lower, ".md") {
		return false
	}

	segments := strings.Split(lower, "/")
	for _, segment := range segments {
		switch segment {
		case "adr", "adrs", "architecture-decisions", "decisions":
			return true
		}
	}

	return false
}

func adrTitle(content, path string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			return strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		}
	}

	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	base = strings.ReplaceAll(base, "-", " ")
	base = strings.ReplaceAll(base, "_", " ")
	return strings.TrimSpace(base)
}

func (s *Scanner) parseMCPConfigJSON(path, filename string) ([]types.DiscoveredTech, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	type mcpServerConfig struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
		URL     string   `json:"url"`
	}

	type mcpConfig struct {
		MCPServers map[string]mcpServerConfig `json:"mcpServers"`
		Servers    map[string]mcpServerConfig `json:"servers"`
	}

	var config mcpConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	servers := config.MCPServers
	if len(servers) == 0 {
		servers = config.Servers
	}
	if len(servers) == 0 {
		var rootServers map[string]mcpServerConfig
		if err := json.Unmarshal(data, &rootServers); err == nil {
			servers = rootServers
		}
	}
	if len(servers) == 0 {
		return nil, nil
	}

	techs := []types.DiscoveredTech{
		{
			Name:     "Model Context Protocol",
			Source:   filename,
			Quadrant: types.QuadrantTools,
		},
	}

	for _, server := range servers {
		for _, candidate := range extractMCPPackages(server.Command, server.Args, server.URL) {
			appendUniqueTech(&techs, types.DiscoveredTech{
				Name:   candidate,
				Source: filename,
			})
		}
	}

	return techs, nil
}

func extractMCPPackages(command string, args []string, url string) []string {
	tokens := append([]string{command}, args...)
	var packages []string

	for idx, token := range tokens {
		if token == "-m" && idx+1 < len(tokens) {
			appendUniqueString(&packages, strings.TrimSpace(tokens[idx+1]))
			continue
		}

		if candidate := mcpCandidateFromToken(token); candidate != "" {
			appendUniqueString(&packages, candidate)
		}
	}

	if candidate := mcpCandidateFromToken(url); candidate != "" {
		appendUniqueString(&packages, candidate)
	}

	return packages
}

func mcpCandidateFromToken(token string) string {
	trimmed := strings.Trim(strings.TrimSpace(token), `"'`)
	if trimmed == "" || strings.HasPrefix(trimmed, "-") {
		return ""
	}

	lower := strings.ToLower(trimmed)
	switch lower {
	case "npx", "pnpm", "pnpx", "yarn", "bunx", "uv", "uvx", "python", "python3", "node", "docker", "go":
		return ""
	}

	keywords := []string{
		"mcp",
		"modelcontextprotocol",
		"openai",
		"anthropic",
		"langchain",
		"langgraph",
		"ollama",
		"vllm",
		"promptfoo",
		"litellm",
		"llamaindex",
		"llama-index",
		"autogen",
		"crewai",
		"dspy",
	}
	for _, keyword := range keywords {
		if strings.Contains(lower, keyword) {
			return trimmed
		}
	}

	return ""
}

func mergeTech(existing, next types.DiscoveredTech) types.DiscoveredTech {
	if existing.Quadrant == "" {
		existing.Quadrant = next.Quadrant
	}
	if existing.Version == "" {
		existing.Version = next.Version
	}
	if next.Source == "adr" && existing.Source != "adr" {
		existing.Source = next.Source
	}
	if existing.EvidencePath == "" {
		existing.EvidencePath = next.EvidencePath
	}
	if existing.EvidenceTitle == "" {
		existing.EvidenceTitle = next.EvidenceTitle
	}
	return existing
}

func appendUniqueTech(techs *[]types.DiscoveredTech, tech types.DiscoveredTech) {
	key := strings.ToLower(strings.TrimSpace(tech.Name))
	for _, existing := range *techs {
		if strings.ToLower(strings.TrimSpace(existing.Name)) == key {
			return
		}
	}
	*techs = append(*techs, tech)
}

func appendIfMissing(values []string, value string) []string {
	for _, existing := range values {
		if strings.EqualFold(existing, value) {
			return values
		}
	}
	return append(values, value)
}

func appendUniqueString(values *[]string, value string) {
	*values = appendIfMissing(*values, value)
}
