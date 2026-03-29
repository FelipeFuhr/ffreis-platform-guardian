package reporter

import (
	"encoding/json"
	"io"

	"github.com/ffreis/platform-guardian/internal/engine"
	"github.com/ffreis/platform-guardian/internal/rule"
)

type SARIFReporter struct {
	w io.Writer
}

type sarifOutput struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name    string      `json:"name"`
	Version string      `json:"version"`
	Rules   []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	ShortDescription sarifMessage           `json:"shortDescription"`
	Properties       map[string]interface{} `json:"properties,omitempty"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

func (r *SARIFReporter) Report(report *engine.ScanReport) error {
	// Collect unique rules
	ruleMap := make(map[string]*engine.RuleResult)
	for i := range report.Results {
		result := &report.Results[i]
		if result.Rule != nil {
			ruleMap[result.Rule.ID] = result
		}
	}

	var sarifRules []sarifRule
	for _, result := range ruleMap {
		if result.Rule == nil {
			continue
		}
		sarifRules = append(sarifRules, sarifRule{
			ID:   result.Rule.ID,
			Name: result.Rule.Name,
			ShortDescription: sarifMessage{
				Text: result.Rule.Name,
			},
			Properties: map[string]interface{}{
				"severity": string(result.Rule.Severity),
				"tags":     result.Rule.Tags,
			},
		})
	}

	var sarifResults []sarifResult
	for _, result := range report.Results {
		if result.Status != engine.StatusFail {
			continue
		}
		if result.Rule == nil {
			continue
		}

		level := "error"
		if result.Rule.Severity == rule.SeverityWarning {
			level = "warning"
		} else if result.Rule.Severity == rule.SeverityInfo {
			level = "note"
		}

		sarifResults = append(sarifResults, sarifResult{
			RuleID:  result.Rule.ID,
			Level:   level,
			Message: sarifMessage{Text: result.Message},
			Locations: []sarifLocation{
				{
					PhysicalLocation: sarifPhysicalLocation{
						ArtifactLocation: sarifArtifactLocation{URI: "."},
					},
				},
			},
		})
	}

	output := sarifOutput{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{
			{
				Tool: sarifTool{
					Driver: sarifDriver{
						Name:    "platform-guardian",
						Version: "1.0.0",
						Rules:   sarifRules,
					},
				},
				Results: sarifResults,
			},
		},
	}

	enc := json.NewEncoder(r.w)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}
