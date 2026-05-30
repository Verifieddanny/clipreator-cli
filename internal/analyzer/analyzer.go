package analyzer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/Verifieddanny/clipreator-cli/internal/transcriber"
)

type Clip struct {
	Title string  `json:"title"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Why   string  `json:"why"`
	Hook  string  `json:"hook"`
}

type AnalysisResult struct {
	Clips []Clip `json:"clips"`
}

func Analyze(transcript *transcriber.Transcript) (*AnalysisResult, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	// Build transcript text with timestamps for Claude
	var transcriptText strings.Builder
	for _, seg := range transcript.Segments {
		fmt.Fprintf(&transcriptText, "[%.2f - %.2f] %s\n", seg.Start, seg.End, seg.Text)
	}

	prompt := fmt.Sprintf(`You are a viral short-form video editor. Analyze this transcript and find the 3-5 best clips that would perform well as TikTok/Reels/YouTube Shorts.

Each clip should be 30-90 seconds long. Look for:
- Strong hooks (attention-grabbing opening lines)
- Emotional moments, funny moments, controversial takes
- Complete thoughts (don't cut mid-sentence)
- High energy or dramatic delivery

Return ONLY valid JSON, no other text:
{
  "clips": [
    {
      "title": "short catchy title for the clip",
      "start": start_timestamp_in_seconds,
      "end": end_timestamp_in_seconds,
      "why": "why this clip would go viral",
      "hook": "the opening line that grabs attention"
    }
  ]
}

TRANSCRIPT:
%s`, transcriptText.String())

	reqBody := map[string]interface{}{
		"model":      "claude-sonnet-4-6",
		"max_tokens": 2000,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	// Parse the Claude API response
	var apiResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("empty response from API")
	}

	// Parse Claude's JSON response into our struct
	var result AnalysisResult
	if err := json.Unmarshal([]byte(sanitizeJSON(apiResp.Content[0].Text)), &result); err != nil {
		return nil, fmt.Errorf("failed to parse clip suggestions: %w\nRaw: %s", err, apiResp.Content[0].Text)
	}

	return &result, nil
}

func sanitizeJSON(input string) string {
	// 1. Find the first occurrence of '{'
	start := strings.Index(input, "{")
	// 2. Find the last occurrence of '}'
	end := strings.LastIndex(input, "}")

	// 3. If both are found and start is before end, extract that slice
	if start != -1 && end != -1 && start < end {
		return input[start : end+1]
	}

	// Fallback to trimming if the markers aren't found
	return strings.TrimSpace(input)
}
