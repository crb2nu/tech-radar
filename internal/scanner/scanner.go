package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	seenTechs := make(map[string]bool)

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// Skip hidden directories like .git, .node_modules, etc.
			if strings.HasPrefix(d.Name(), ".") && d.Name() != "." && d.Name() != ".." {
				return filepath.SkipDir
			}
			if d.Name() == "node_modules" || d.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file is in the list of files to scan
		filename := d.Name()
		shouldScan := false
		for _, f := range s.config.Files {
			if f == filename {
				shouldScan = true
				break
			}
		}

		if !shouldScan {
			return nil
		}

		// Parse the file
		techs, err := s.parseFile(path, filename)
		if err != nil {
			return nil
		}

		if len(techs) > 0 {
			relPath, _ := filepath.Rel(root, path)
			var techNames []string
			for _, t := range techs {
				t = s.normalizeTech(t)
				if t.Name == "" {
					continue
				}
				techNames = appendIfMissing(techNames, t.Name)
				key := strings.ToLower(t.Name)
				if seenTechs[key] {
					continue
				}
				seenTechs[key] = true
				result.Technologies = append(result.Technologies, t)
			}
			result.Files[relPath] = techNames
		}

		return nil
	})

	return result, err
}

func (s *Scanner) parseFile(path, filename string) ([]types.DiscoveredTech, error) {
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

	// Standard PEP 621 dependencies
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
