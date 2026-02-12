package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"gitlab.flexinfer.ai/services/tech-radar/pkg/types"
)

func TestScanner_Scan(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "scanner_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a package.json file
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

	// Create a go.mod file
	goMod := `module example.com/test

go 1.21

require (
	github.com/spf13/cobra v1.8.0
	gopkg.in/yaml.v3 v3.0.1
)`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Create a requirements.txt file
	requirementsTxt := `
flask==3.0.0
requests>=2.31.0
numpy
`
	if err := os.WriteFile(filepath.Join(tempDir, "requirements.txt"), []byte(requirementsTxt), 0644); err != nil {
		t.Fatalf("Failed to write requirements.txt: %v", err)
	}

	// Create scanner config
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

	// Verify results
	expectedTechs := map[string]bool{
		"react":                    false,
		"axios":                    false,
		"typescript":               false,
		"github.com/spf13/cobra":   false,
		"gopkg.in/yaml.v3":         false,
		"flask":                    false,
		"requests":                 false,
		"numpy":                    false,
	}

	for _, tech := range result.Technologies {
		if _, ok := expectedTechs[tech.Name]; ok {
			expectedTechs[tech.Name] = true
		}
		// Check mapping for React
		if tech.Name == "React" { // Mapped name
			expectedTechs["react"] = true
			if tech.Quadrant != types.QuadrantFrameworks {
				t.Errorf("Expected React quadrant to be frameworks, got %s", tech.Quadrant)
			}
		}
	}

	for name, found := range expectedTechs {
		if !found {
			t.Errorf("Expected to find technology: %s", name)
		}
	}
}
