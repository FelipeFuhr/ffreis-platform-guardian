package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const claudeAPIURL = "https://api.anthropic.com/v1/messages"
const claudeModel = "claude-haiku-4-5-20251001"

type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []claudeMessage `json:"messages"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// EnhanceSuggestions calls the Claude API to enrich fix suggestions with
// context-aware remediation advice.  It is a no-op when apiKey is empty.
//
// The API key is read from apiKey; if empty, ANTHROPIC_API_KEY is tried.
// If no key is available the suggestions are returned unchanged.
func EnhanceSuggestions(ctx context.Context, suggestions []Suggestion, apiKey string) ([]Suggestion, error) {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		return suggestions, nil
	}

	// Build a compact prompt summarising the top suggestions.
	prompt := buildPrompt(suggestions)

	body, err := callClaude(ctx, apiKey, prompt)
	if err != nil {
		return suggestions, fmt.Errorf("claude API: %w", err)
	}

	// Parse the response — Claude returns one line per suggestion, prefixed "N. "
	enhanced := parseEnhanced(body, len(suggestions))
	for i, text := range enhanced {
		if i < len(suggestions) && text != "" {
			suggestions[i].Enhanced = text
		}
	}
	return suggestions, nil
}

func buildPrompt(suggestions []Suggestion) string {
	var buf bytes.Buffer
	buf.WriteString("You are a platform engineering assistant. For each rule violation below, write a single concise fix instruction (1-2 sentences) that a developer can act on immediately. Reply with numbered lines only (no extra text).\n\n")
	for i, s := range suggestions {
		fmt.Fprintf(&buf, "%d. Rule: %s\n", i+1, s.RuleID)
		if s.Remediation != "" {
			fmt.Fprintf(&buf, "   Existing guidance: %s\n", s.Remediation)
		}
		fmt.Fprintf(&buf, "   Repos affected: %d\n", len(s.AffectedRepos))
	}
	return buf.String()
}

func callClaude(ctx context.Context, apiKey, prompt string) (string, error) {
	req := claudeRequest{
		Model:     claudeModel,
		MaxTokens: 1024,
		Messages:  []claudeMessage{{Role: "user", Content: prompt}},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, claudeAPIURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var cr claudeResponse
	if err := json.Unmarshal(respBody, &cr); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	if cr.Error != nil {
		return "", fmt.Errorf("%s: %s", cr.Error.Type, cr.Error.Message)
	}

	for _, block := range cr.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}
	return "", nil
}

// parseEnhanced splits a numbered response into per-suggestion lines.
func parseEnhanced(text string, count int) []string {
	out := make([]string, count)
	if text == "" {
		return out
	}

	// Split on newlines; lines starting with "N. " map to suggestion index N-1.
	lines := bytes.Split([]byte(text), []byte("\n"))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		// Find "N. " prefix
		dot := bytes.Index(line, []byte(". "))
		if dot < 1 {
			continue
		}
		idx := 0
		for _, c := range line[:dot] {
			if c < '0' || c > '9' {
				idx = -1
				break
			}
			idx = idx*10 + int(c-'0')
		}
		if idx < 1 || idx > count {
			continue
		}
		out[idx-1] = string(bytes.TrimSpace(line[dot+2:]))
	}
	return out
}
