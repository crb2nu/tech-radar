package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gitlab.flexinfer.ai/services/tech-radar/internal/scanner"
	"gitlab.flexinfer.ai/services/tech-radar/pkg/types"
	"gopkg.in/yaml.v3"
)

type gitLabProject struct {
	ID                int    `json:"id"`
	PathWithNamespace string `json:"path_with_namespace"`
	HTTPURLToRepo     string `json:"http_url_to_repo"`
	DefaultBranch     string `json:"default_branch"`
}

func syncGitLabCmd() *cobra.Command {
	var (
		gitlabURL string
		token     string
		groups    string
		apiURL    string
		apiKey    string
		tmpDir    string
		limit     int
	)

	cmd := &cobra.Command{
		Use:   "sync-gitlab",
		Short: "Scan GitLab projects and publish results to the radar",
		RunE: func(cmd *cobra.Command, args []string) error {
			if gitlabURL == "" {
				return fmt.Errorf("--gitlab-url or GITLAB_URL is required")
			}
			if token == "" {
				return fmt.Errorf("--gitlab-token or GITLAB_TOKEN is required")
			}
			if apiURL == "" {
				return fmt.Errorf("--api-url or RADAR_API_URL is required")
			}

			var projects []gitLabProject
			if groups == "" {
				batch, err := listGitLabMembershipProjects(gitlabURL, token, limit)
				if err != nil {
					return err
				}
				projects = append(projects, batch...)
			} else {
				for _, group := range splitCSV(groups) {
					batch, err := listGitLabGroupProjects(gitlabURL, token, group, remainingLimit(limit, len(projects)))
					if err != nil {
						return err
					}
					projects = append(projects, batch...)
					if limit > 0 && len(projects) >= limit {
						projects = projects[:limit]
						break
					}
				}
			}
			if len(projects) == 0 {
				fmt.Println("No GitLab projects found")
				return nil
			}

			config, err := loadScanConfig()
			if err != nil {
				return err
			}
			s := scanner.New(config.Scan)

			if err := os.MkdirAll(tmpDir, 0755); err != nil {
				return err
			}
			defer func() {
				if err := os.RemoveAll(tmpDir); err != nil {
					fmt.Fprintf(os.Stderr, "warning: failed to clean %s: %v\n", tmpDir, err)
				}
			}()

			var failures int
			for _, project := range projects {
				projectDir := filepath.Join(tmpDir, strconv.Itoa(project.ID))
				fmt.Printf("Syncing %s...\n", project.PathWithNamespace)

				if err := cloneGitLabProject(project, token, projectDir); err != nil {
					failures++
					fmt.Fprintf(os.Stderr, "  clone failed: %v\n", err)
					continue
				}

				result, err := s.Scan(projectDir)
				if err != nil {
					failures++
					fmt.Fprintf(os.Stderr, "  scan failed: %v\n", err)
					continue
				}

				result.ProjectID = project.PathWithNamespace
				if result.Path == "" {
					result.Path = project.HTTPURLToRepo
				}

				if err := publishResult(apiURL, apiKey, *result); err != nil {
					failures++
					fmt.Fprintf(os.Stderr, "  publish failed: %v\n", err)
					continue
				}

				fmt.Printf("  published %d technologies from %d files\n", len(result.Technologies), len(result.Files))
			}

			if failures > 0 {
				return fmt.Errorf("gitlab sync completed with %d failure(s)", failures)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&gitlabURL, "gitlab-url", envOrDefault("GITLAB_URL", "https://gitlab.flexinfer.ai"), "GitLab base URL")
	cmd.Flags().StringVar(&token, "gitlab-token", os.Getenv("GITLAB_TOKEN"), "GitLab token with read_api and read_repository access")
	cmd.Flags().StringVar(&groups, "gitlab-group", envOrDefault("GITLAB_GROUPS", os.Getenv("GITLAB_GROUP")), "Optional comma-separated GitLab group paths or IDs to scan; empty scans token membership")
	cmd.Flags().StringVar(&apiURL, "api-url", envOrDefault("RADAR_API_URL", "http://tech-radar.tech-radar.svc.cluster.local/api/v1/ingest"), "Radar ingest API URL")
	cmd.Flags().StringVar(&apiKey, "api-key", os.Getenv("RADAR_API_KEY"), "Radar API key")
	cmd.Flags().StringVar(&tmpDir, "tmp-dir", envOrDefault("RADAR_SYNC_TMP_DIR", "/tmp/tech-radar-gitlab"), "Temporary clone directory")
	cmd.Flags().IntVar(&limit, "limit", envInt("GITLAB_PROJECT_LIMIT", 0), "Maximum projects to scan; 0 means all")

	return cmd
}

func listGitLabGroupProjects(gitlabURL, token, group string, limit int) ([]gitLabProject, error) {
	return listGitLabProjects(gitlabURL, token, "/api/v4/groups/"+url.PathEscape(group)+"/projects", map[string]string{
		"include_subgroups": "true",
		"archived":          "false",
		"simple":            "true",
	}, limit)
}

func listGitLabMembershipProjects(gitlabURL, token string, limit int) ([]gitLabProject, error) {
	return listGitLabProjects(gitlabURL, token, "/api/v4/projects", map[string]string{
		"membership": "true",
		"archived":   "false",
		"simple":     "true",
	}, limit)
}

func listGitLabProjects(gitlabURL, token, apiPath string, queryParams map[string]string, limit int) ([]gitLabProject, error) {
	base, err := url.Parse(strings.TrimRight(gitlabURL, "/"))
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	page := 1
	projects := []gitLabProject{}
	for {
		reqURL := *base
		reqURL.Path = strings.TrimRight(reqURL.Path, "/") + apiPath
		query := reqURL.Query()
		for key, value := range queryParams {
			query.Set(key, value)
		}
		query.Set("per_page", "100")
		query.Set("page", strconv.Itoa(page))
		reqURL.RawQuery = query.Encode()

		req, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("PRIVATE-TOKEN", token)

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		var batch []gitLabProject
		decodeErr := json.NewDecoder(resp.Body).Decode(&batch)
		closeErr := resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("gitlab returned status %s while listing projects", resp.Status)
		}
		if decodeErr != nil {
			return nil, decodeErr
		}
		if closeErr != nil {
			return nil, closeErr
		}

		projects = append(projects, batch...)
		if limit > 0 && len(projects) >= limit {
			return projects[:limit], nil
		}
		if resp.Header.Get("X-Next-Page") == "" || len(batch) == 0 {
			return projects, nil
		}
		page++
	}
}

func cloneGitLabProject(project gitLabProject, token, target string) error {
	if project.HTTPURLToRepo == "" {
		return fmt.Errorf("project has no HTTP clone URL")
	}
	if err := os.RemoveAll(target); err != nil {
		return err
	}

	cloneURL, err := authenticatedCloneURL(project.HTTPURLToRepo, token)
	if err != nil {
		return err
	}

	args := []string{"clone", "--depth=1", "--single-branch"}
	if project.DefaultBranch != "" {
		args = append(args, "--branch", project.DefaultBranch)
	}
	args = append(args, cloneURL, target)

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, sanitizeToken(string(output), token))
	}
	return nil
}

func authenticatedCloneURL(rawURL, token string) (string, error) {
	cloneURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	cloneURL.User = url.UserPassword("oauth2", token)
	return cloneURL.String(), nil
}

func loadScanConfig() (types.Config, error) {
	config := defaultScanConfig()
	if data, err := os.ReadFile(".tech-radar/config.yaml"); err == nil {
		if err := yaml.Unmarshal(data, &config); err != nil {
			return config, fmt.Errorf("failed to parse .tech-radar/config.yaml: %w", err)
		}
	}
	return config, nil
}

func defaultScanConfig() types.Config {
	return types.Config{
		Scan: types.ScanConfig{
			Files: []string{
				"package.json",
				"go.mod",
				"requirements.txt",
				"pyproject.toml",
				"Cargo.toml",
				".mcp.json",
				"mcp.json",
				"claude_desktop_config.json",
				"cline_mcp_settings.json",
			},
		},
	}
}

func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	var output []string
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			output = append(output, item)
		}
	}
	return output
}

func remainingLimit(limit, current int) int {
	if limit <= 0 {
		return 0
	}
	return limit - current
}

func sanitizeToken(value, token string) string {
	if token == "" {
		return value
	}
	return strings.ReplaceAll(value, token, "<redacted>")
}
