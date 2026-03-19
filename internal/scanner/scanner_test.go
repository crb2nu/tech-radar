package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"gitlab.flexinfer.ai/services/tech-radar/pkg/types"
)

func TestScanner_Scan(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "scanner_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("Failed to remove temp dir: %v", err)
		}
	})

	packageJSON := `{
		"dependencies": {
			"react": "^18.2.0",
			"axios": "^1.6.0"
		},
		"devDependencies": {
			"typescript": "^5.0.0"
		}
	}`
	if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	goMod := `module example.com/test

go 1.21

require (
	github.com/spf13/cobra v1.8.0
	gopkg.in/yaml.v3 v3.0.1
)`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	requirementsTxt := `
flask==3.0.0
requests>=2.31.0
numpy
`
	if err := os.WriteFile(filepath.Join(tempDir, "requirements.txt"), []byte(requirementsTxt), 0644); err != nil {
		t.Fatalf("Failed to write requirements.txt: %v", err)
	}

	config := types.ScanConfig{
		Files: []string{"package.json", "go.mod", "requirements.txt"},
		Mappings: map[string]types.EntryHint{
			"react": {Name: "React", Quadrant: types.QuadrantFrameworks},
		},
	}

	scanner := New(config)
	result, err := scanner.Scan(tempDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	expectedTechs := map[string]bool{
		"React":                  false,
		"axios":                  false,
		"typescript":             false,
		"github.com/spf13/cobra": false,
		"gopkg.in/yaml.v3":       false,
		"flask":                  false,
		"requests":               false,
		"numpy":                  false,
	}

	for _, tech := range result.Technologies {
		if _, ok := expectedTechs[tech.Name]; ok {
			expectedTechs[tech.Name] = true
		}
		if tech.Name == "React" && tech.Quadrant != types.QuadrantFrameworks {
			t.Errorf("Expected React quadrant to be frameworks, got %s", tech.Quadrant)
		}
	}

	for name, found := range expectedTechs {
		if !found {
			t.Errorf("Expected to find technology: %s", name)
		}
	}
}

func TestScanner_ScanMCPConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "scanner_mcp_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("Failed to remove temp dir: %v", err)
		}
	})

	packageJSON := `{
		"dependencies": {
			"openai": "^4.90.0"
		}
	}`
	if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	mcpConfig := `{
		"mcpServers": {
			"filesystem": {
				"command": "npx",
				"args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
			},
			"fetch": {
				"command": "uvx",
				"args": ["mcp-server-fetch"]
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(tempDir, ".mcp.json"), []byte(mcpConfig), 0644); err != nil {
		t.Fatalf("Failed to write .mcp.json: %v", err)
	}

	config := types.ScanConfig{
		Files: []string{"package.json", ".mcp.json"},
		Mappings: map[string]types.EntryHint{
			"openai": {Name: "OpenAI API", Quadrant: types.QuadrantPlatforms},
			"@modelcontextprotocol/server-filesystem": {Name: "Filesystem MCP Server", Quadrant: types.QuadrantTools},
			"mcp-server-fetch":                        {Name: "Fetch MCP Server", Quadrant: types.QuadrantTools},
		},
	}

	scanner := New(config)
	result, err := scanner.Scan(tempDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	expected := map[string]types.Quadrant{
		"OpenAI API":             types.QuadrantPlatforms,
		"Model Context Protocol": types.QuadrantTools,
		"Filesystem MCP Server":  types.QuadrantTools,
		"Fetch MCP Server":       types.QuadrantTools,
	}

	for name, quadrant := range expected {
		found := false
		for _, tech := range result.Technologies {
			if tech.Name != name {
				continue
			}
			found = true
			if tech.Quadrant != quadrant {
				t.Fatalf("expected %s quadrant to be %s, got %s", name, quadrant, tech.Quadrant)
			}
		}
		if !found {
			t.Fatalf("expected to find technology %s, got %#v", name, result.Technologies)
		}
	}
}

func TestScanner_ScanADRMarkdown(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "scanner_adr_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("Failed to remove temp dir: %v", err)
		}
	})

	adrDir := filepath.Join(tempDir, "docs", "adrs")
	if err := os.MkdirAll(adrDir, 0755); err != nil {
		t.Fatalf("Failed to create ADR dir: %v", err)
	}

	adr := `# ADR-0007 Adopt React and TypeScript

## Status
Accepted

We are standardizing on React and TypeScript for the frontend stack.
`
	adrPath := filepath.Join(adrDir, "0007-adopt-react-typescript.md")
	if err := os.WriteFile(adrPath, []byte(adr), 0644); err != nil {
		t.Fatalf("Failed to write ADR: %v", err)
	}

	config := types.ScanConfig{
		Mappings: map[string]types.EntryHint{
			"react":      {Name: "React", Quadrant: types.QuadrantFrameworks},
			"typescript": {Name: "TypeScript", Quadrant: types.QuadrantLanguages},
		},
	}

	scanner := New(config)
	result, err := scanner.Scan(tempDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(result.Technologies) != 2 {
		t.Fatalf("expected 2 ADR-discovered technologies, got %#v", result.Technologies)
	}

	expectedPath := "docs/adrs/0007-adopt-react-typescript.md"
	for _, tech := range result.Technologies {
		if tech.Source != "adr" {
			t.Fatalf("expected adr source, got %s", tech.Source)
		}
		if tech.EvidencePath != expectedPath {
			t.Fatalf("expected evidence path %s, got %s", expectedPath, tech.EvidencePath)
		}
		if tech.EvidenceTitle != "ADR-0007 Adopt React and TypeScript" {
			t.Fatalf("unexpected evidence title %q", tech.EvidenceTitle)
		}
	}
}

func TestScanner_ScanHiddenMCPConfigDir(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "scanner_hidden_mcp_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("Failed to remove temp dir: %v", err)
		}
	})

	cursorDir := filepath.Join(tempDir, ".cursor")
	if err := os.MkdirAll(cursorDir, 0755); err != nil {
		t.Fatalf("Failed to create hidden MCP config dir: %v", err)
	}

	mcpConfig := `{
		"mcpServers": {
			"filesystem": {
				"command": "npx",
				"args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(cursorDir, "mcp.json"), []byte(mcpConfig), 0644); err != nil {
		t.Fatalf("Failed to write hidden-dir mcp.json: %v", err)
	}

	config := types.ScanConfig{
		Files: []string{"mcp.json"},
		Mappings: map[string]types.EntryHint{
			"@modelcontextprotocol/server-filesystem": {Name: "Filesystem MCP Server", Quadrant: types.QuadrantTools},
		},
	}

	scanner := New(config)
	result, err := scanner.Scan(tempDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	expectedPath := ".cursor/mcp.json"
	foundProtocol := false
	foundFilesystem := false
	for _, tech := range result.Technologies {
		switch tech.Name {
		case "Model Context Protocol":
			foundProtocol = true
			if tech.Source != "mcp.json" {
				t.Fatalf("expected hidden-dir MCP source to stay mcp.json, got %s", tech.Source)
			}
			if tech.EvidencePath != expectedPath {
				t.Fatalf("expected evidence path %s, got %s", expectedPath, tech.EvidencePath)
			}
		case "Filesystem MCP Server":
			foundFilesystem = true
			if tech.Quadrant != types.QuadrantTools {
				t.Fatalf("expected Filesystem MCP Server quadrant to be tools, got %s", tech.Quadrant)
			}
			if tech.EvidencePath != expectedPath {
				t.Fatalf("expected evidence path %s, got %s", expectedPath, tech.EvidencePath)
			}
		}
	}

	if !foundProtocol || !foundFilesystem {
		t.Fatalf("expected hidden MCP config discoveries, got %#v", result.Technologies)
	}
}

func TestScanner_ScanPythonMCPSDKMappings(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "scanner_python_mcp_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("Failed to remove temp dir: %v", err)
		}
	})

	pyproject := `
[project]
dependencies = [
	"mcp>=1.6.0",
	"fastmcp==2.0.0",
	"anthropic>=0.49.0",
]
`
	if err := os.WriteFile(filepath.Join(tempDir, "pyproject.toml"), []byte(pyproject), 0644); err != nil {
		t.Fatalf("Failed to write pyproject.toml: %v", err)
	}

	config := types.ScanConfig{
		Files: []string{"pyproject.toml"},
		Mappings: map[string]types.EntryHint{
			"mcp":       {Name: "Model Context Protocol", Quadrant: types.QuadrantTools},
			"fastmcp":   {Name: "FastMCP", Quadrant: types.QuadrantTools},
			"anthropic": {Name: "Claude API", Quadrant: types.QuadrantPlatforms},
		},
	}

	scanner := New(config)
	result, err := scanner.Scan(tempDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	expected := map[string]types.Quadrant{
		"Model Context Protocol": types.QuadrantTools,
		"FastMCP":                types.QuadrantTools,
		"Claude API":             types.QuadrantPlatforms,
	}

	for name, quadrant := range expected {
		found := false
		for _, tech := range result.Technologies {
			if tech.Name != name {
				continue
			}
			found = true
			if tech.Quadrant != quadrant {
				t.Fatalf("expected %s quadrant to be %s, got %s", name, quadrant, tech.Quadrant)
			}
		}
		if !found {
			t.Fatalf("expected to find normalized technology %s, got %#v", name, result.Technologies)
		}
	}
}
