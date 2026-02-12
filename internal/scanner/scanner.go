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
				techNames = append(techNames, t.Name)
				// Check mappings for quadrant hint
				if hint, ok := s.config.Mappings[strings.ToLower(t.Name)]; ok {
					t.Quadrant = hint.Quadrant
					if t.Name != hint.Name {
						t.Name = hint.Name // Use canonical name
					}
				}
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
	}
	return nil, nil
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
