package db

import (
	"math"
	"sort"
	"strings"

	"gitlab.flexinfer.ai/services/tech-radar/pkg/types"
)

var ringPriority = map[string]int{
	"adopt":  4,
	"trial":  3,
	"assess": 2,
	"hold":   1,
}

var quadrantOrder = map[string]int{
	"languages":  0,
	"frameworks": 1,
	"tools":      2,
	"platforms":  3,
	"techniques": 4,
}

var aiMCPKeywords = []string{
	"mcp",
	"model context protocol",
	"modelcontextprotocol",
	"openai",
	"anthropic",
	"claude",
	"langchain",
	"langgraph",
	"llamaindex",
	"llama-index",
	"ollama",
	"vllm",
	"promptfoo",
	"litellm",
	"autogen",
	"crewai",
	"dspy",
}

var aiMCPSources = map[string]struct{}{
	".mcp.json":                  {},
	"mcp.json":                   {},
	"claude_desktop_config.json": {},
	"cline_mcp_settings.json":    {},
}

func buildInsights(radar *types.Radar, projects []types.Project) *types.Insights {
	insights := &types.Insights{
		ProjectsCount: len(projects),
	}

	if radar == nil {
		radar = &types.Radar{}
	}

	insights.TechCount = len(radar.Entries)

	entryByKey := make(map[string]types.Entry, len(radar.Entries))
	quadrantCounts := make(map[string]int)
	ringCounts := make(map[string]int)
	tagCounts := make(map[string]int)
	driftCounts := make(map[string]int)
	recentMoves := make([]types.RecentMove, 0, len(radar.Entries))

	for _, entry := range radar.Entries {
		key := strings.ToLower(entry.Name)
		entryByKey[key] = entry
		quadrantCounts[string(entry.Quadrant)]++
		ringCounts[string(entry.Ring)]++
		for _, tag := range entry.Tags {
			tagCounts[strings.TrimSpace(tag)]++
		}
		if entry.Ring == types.RingHold && len(entry.ProjectIDs) > 0 {
			driftCounts[entry.Name] = len(entry.ProjectIDs)
		}

		if move, ok := latestMove(entry); ok {
			recentMoves = append(recentMoves, move)
		}
	}

	usageCounts := map[string]int{}
	trackedCounts := map[string]int{}
	unchartedCounts := map[string]int{}
	aiMCPTrackedCounts := map[string]int{}
	aiMCPUnchartedCounts := map[string]int{}
	sourceCounts := map[string]int{}
	aiMCPProjects := 0

	projectHealth := make([]types.ProjectHealth, 0, len(projects))
	for _, project := range projects {
		seenProjectTechs := map[string]bool{}
		projectHasAIMCP := false
		health := types.ProjectHealth{
			ID:       project.ID,
			Name:     project.Name,
			LastScan: project.LastScan,
		}

		for _, tech := range project.Technologies {
			key := strings.ToLower(tech.Name)
			if !seenProjectTechs[key] {
				usageCounts[tech.Name]++
				seenProjectTechs[key] = true
			}

			sourceCounts[tech.Source]++
			health.TotalTechs++
			if len(health.TopTechs) < 6 {
				health.TopTechs = append(health.TopTechs, tech.Name)
			}

			entry, tracked := entryByKey[key]
			if !tracked {
				health.UntrackedTechs++
				unchartedCounts[tech.Name]++
				if isAIMCPSignal(tech.Name, tech.Source) {
					projectHasAIMCP = true
					aiMCPUnchartedCounts[tech.Name]++
				}
				continue
			}

			health.TrackedTechs++
			trackedCounts[entry.Name]++
			if isAIMCPSignal(entry.Name, tech.Source) || isAIMCPSignal(tech.Name, tech.Source) {
				projectHasAIMCP = true
				aiMCPTrackedCounts[entry.Name]++
			}

			switch entry.Ring {
			case types.RingAdopt:
				health.AdoptTechs++
			case types.RingTrial:
				health.TrialTechs++
			case types.RingAssess:
				health.AssessTechs++
			case types.RingHold:
				health.HoldTechs++
			}
		}

		health.CoveragePercent = percentage(health.TrackedTechs, health.TotalTechs)
		health.RadarAlignmentScore = clampInt(
			int(math.Round(health.CoveragePercent))+
				(health.AdoptTechs*4)-
				(health.UntrackedTechs*8)-
				(health.HoldTechs*15),
			0,
			100,
		)
		health.RiskLevel = riskLevel(health)
		projectHealth = append(projectHealth, health)
		if projectHasAIMCP {
			aiMCPProjects++
		}
	}

	sort.Slice(projectHealth, func(i, j int) bool {
		ri := riskOrder(projectHealth[i].RiskLevel)
		rj := riskOrder(projectHealth[j].RiskLevel)
		if ri != rj {
			return ri < rj
		}
		if projectHealth[i].RadarAlignmentScore == projectHealth[j].RadarAlignmentScore {
			if projectHealth[i].HoldTechs == projectHealth[j].HoldTechs {
				return projectHealth[i].Name < projectHealth[j].Name
			}
			return projectHealth[i].HoldTechs > projectHealth[j].HoldTechs
		}
		return projectHealth[i].RadarAlignmentScore < projectHealth[j].RadarAlignmentScore
	})
	if len(projectHealth) > 8 {
		projectHealth = projectHealth[:8]
	}

	sort.Slice(recentMoves, func(i, j int) bool {
		return recentMoves[i].Date.After(recentMoves[j].Date)
	})
	if len(recentMoves) > 8 {
		recentMoves = recentMoves[:8]
	}

	insights.TopUncharted = usageStatsFromMap(unchartedCounts, 8)
	insights.TopAdopted = usageStatsFromMap(trackedCounts, 8)
	insights.TopDrift = usageStatsFromMap(driftCounts, 8)
	insights.TopAIMCPUncharted = usageStatsFromMap(aiMCPUnchartedCounts, 8)
	insights.TopAIMCPTracked = usageStatsFromMap(aiMCPTrackedCounts, 8)
	insights.TopTags = usageStatsFromMap(tagCounts, 8)
	insights.QuadrantBreakdown = labelStatsFromMap(quadrantCounts, quadrantOrder)
	insights.RingBreakdown = labelStatsFromMap(ringCounts, map[string]int{
		"adopt":  0,
		"trial":  1,
		"assess": 2,
		"hold":   3,
	})
	insights.SourceBreakdown = labelStatsFromMap(sourceCounts, nil)
	insights.RecentMoves = recentMoves
	insights.ProjectHealth = projectHealth

	insights.Summary = types.InsightSummary{
		RadarCoveragePercent: percentage(len(trackedCounts), len(trackedCounts)+len(unchartedCounts)),
		TrackedUsageCount:    len(trackedCounts),
		UnchartedUsageCount:  len(unchartedCounts),
		ActiveDriftCount:     len(driftCounts),
		RecentDecisionCount:  len(recentMoves),
		AIMCPProjectCount:    aiMCPProjects,
		AIMCPUnchartedCount:  len(aiMCPUnchartedCounts),
	}

	return insights
}

func isAIMCPSignal(name, source string) bool {
	normalizedName := strings.ToLower(strings.TrimSpace(name))
	if normalizedName != "" {
		for _, keyword := range aiMCPKeywords {
			if strings.Contains(normalizedName, keyword) {
				return true
			}
		}
	}

	if _, ok := aiMCPSources[strings.ToLower(strings.TrimSpace(source))]; ok {
		return true
	}

	return false
}

func latestMove(entry types.Entry) (types.RecentMove, bool) {
	if len(entry.History) == 0 {
		return types.RecentMove{}, false
	}

	history := append([]types.HistoryEntry(nil), entry.History...)
	sort.Slice(history, func(i, j int) bool {
		return history[i].Date.After(history[j].Date)
	})

	move := types.RecentMove{
		Name:         entry.Name,
		ToRing:       string(history[0].Ring),
		Date:         history[0].Date,
		Reason:       history[0].Reason,
		Author:       history[0].Author,
		ProjectCount: len(entry.ProjectIDs),
	}
	if len(history) > 1 {
		move.FromRing = string(history[1].Ring)
	}
	return move, true
}

func usageStatsFromMap(values map[string]int, limit int) []types.UsageStat {
	stats := make([]types.UsageStat, 0, len(values))
	for name, count := range values {
		if name == "" || count <= 0 {
			continue
		}
		stats = append(stats, types.UsageStat{Name: name, Count: count})
	}

	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Count == stats[j].Count {
			return strings.ToLower(stats[i].Name) < strings.ToLower(stats[j].Name)
		}
		return stats[i].Count > stats[j].Count
	})

	if limit > 0 && len(stats) > limit {
		return stats[:limit]
	}
	return stats
}

func labelStatsFromMap(values map[string]int, order map[string]int) []types.LabelStat {
	stats := make([]types.LabelStat, 0, len(values))
	for label, count := range values {
		if label == "" || count <= 0 {
			continue
		}
		stats = append(stats, types.LabelStat{Label: label, Count: count})
	}

	sort.Slice(stats, func(i, j int) bool {
		if order != nil {
			oi, okI := order[stats[i].Label]
			oj, okJ := order[stats[j].Label]
			if okI && okJ && oi != oj {
				return oi < oj
			}
		}
		if stats[i].Count == stats[j].Count {
			return strings.ToLower(stats[i].Label) < strings.ToLower(stats[j].Label)
		}
		return stats[i].Count > stats[j].Count
	})

	return stats
}

func percentage(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return math.Round((float64(part)/float64(total))*1000) / 10
}

func riskLevel(health types.ProjectHealth) string {
	if health.HoldTechs > 0 || health.CoveragePercent < 45 {
		return "critical"
	}
	if health.UntrackedTechs > 0 || health.CoveragePercent < 75 {
		return "watch"
	}
	return "healthy"
}

func clampInt(value, minValue, maxValue int) int {
	return max(minValue, min(value, maxValue))
}

func riskOrder(level string) int {
	switch level {
	case "critical":
		return 0
	case "watch":
		return 1
	default:
		return 2
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
