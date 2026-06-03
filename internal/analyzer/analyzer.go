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
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY_")
	openaiKey := os.Getenv("OPENAI_API_KEY_")
	ollamaModel := os.Getenv("OLLAMA_MODEL")

	if anthropicKey == "" && openaiKey == "" && ollamaModel == "" {
		return nil, fmt.Errorf("no API key found. Set ANTHROPIC_API_KEY or OPENAI_API_KEY or OLLAMA_MODEL")
	}

	var transcriptText strings.Builder
	for _, seg := range transcript.Segments {
		fmt.Fprintf(&transcriptText, "[%.2f - %.2f] %s\n", seg.Start, seg.End, seg.Text)
	}

	// Get total duration for context
	lastSeg := transcript.Segments[len(transcript.Segments)-1]
	totalDuration := lastSeg.End

	prompt := buildPrompt(transcriptText.String(), totalDuration)

	var responseText string
	var err error

	switch {
	case anthropicKey != "":
		fmt.Println("    🧠 Using Claude...")
		responseText, err = callAnthropic(anthropicKey, prompt)
	case openaiKey != "":
		fmt.Println("    🧠 Using OpenAI...")
		responseText, err = callOpenAI(openaiKey, prompt)
	default:
		if ollamaModel == "" {
			ollamaModel = "qwen2.5:7b"
		}
		fmt.Printf("    🧠 Using Ollama (%s)...\n", ollamaModel)

		truncated := truncateTranscript(transcriptText.String(), 12000)
		responseText, err = callOllama(ollamaModel, buildPromptForLocalModel(truncated, totalDuration))
	}

	if err != nil {
		return nil, err
	}

	var result AnalysisResult
	if err := json.Unmarshal([]byte(sanitizeJSON(responseText)), &result); err != nil {
		return nil, fmt.Errorf("failed to parse clip suggestions: %w\nRaw: %s", err, responseText)
	}

	result.Clips = validateClips(result.Clips, totalDuration)

	if len(result.Clips) == 0 && anthropicKey == "" && openaiKey == "" {
		fmt.Println("    🔄 Clips too short, retrying with stronger prompt...")
		truncated := truncateTranscript(transcriptText.String(), 12000)
		retryPrompt := fmt.Sprintf(`Your previous answer had clips shorter than 30 seconds. Try again.

MANDATORY: every clip must be AT LEAST 30 seconds. Calculate end minus start and make sure it is 30 or more.

The video is %.0f seconds long. Find 3 clips that are each 30-90 seconds.

ONLY output JSON: {"clips":[{"title":"x","start":0.0,"end":60.0,"why":"x","hook":"x"}]}

TRANSCRIPT:
%s`, totalDuration, truncated)

		responseText, err = callOllama(ollamaModel, retryPrompt)
		if err != nil {
			return nil, fmt.Errorf("retry failed: %w", err)
		}

		if err := json.Unmarshal([]byte(sanitizeJSON(responseText)), &result); err != nil {
			return nil, fmt.Errorf("failed to parse retry: %w\nRaw: %s", err, responseText)
		}

		result.Clips = validateClips(result.Clips, totalDuration)
	}

	if len(result.Clips) == 0 {
		return nil, fmt.Errorf("no valid clips found (all were too short or out of range)")
	}

	return &result, nil
}

func callAnthropic(apiKey, prompt string) (string, error) {
	reqBody := map[string]any{
		"model":      "claude-sonnet-4-6",
		"max_tokens": 2000,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Anthropic API error (%d): %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse API response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return apiResp.Content[0].Text, nil
}

func callOpenAI(apiKey, prompt string) (string, error) {
	reqBody := map[string]any{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": 2000,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("OpenAI API error (%d): %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse API response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return apiResp.Choices[0].Message.Content, nil
}

func callOllama(model, prompt string) (string, error) {
	ollamaURL := os.Getenv("OLLAMA_HOST")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	reqBody := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"stream": false,
		"options": map[string]any{
			"temperature": 0.3,
			"num_predict": 4000,
			"num_ctx":     16384,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", ollamaURL+"/api/chat", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Ollama request failed (is Ollama running? try: ollama serve): %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Ollama error (%d): %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse Ollama response: %w", err)
	}

	if apiResp.Message.Content == "" {
		return "", fmt.Errorf("empty response from Ollama")
	}

	return apiResp.Message.Content, nil
}

func buildPrompt(transcriptText string, totalDuration float64) string {
	return fmt.Sprintf(`I need you to find the best clips from this video transcript for TikTok/YouTube Shorts.

RULES (follow these exactly):
1. Find 3 to 5 clips
2. Each clip MUST be between 30 and 90 seconds long. Never shorter than 30 seconds.
3. Use the timestamps from the transcript. The video is %.0f seconds total.
4. Each clip must have a strong opening hook that grabs attention in the first 3 seconds
5. Each clip must contain a complete thought - never cut mid-sentence
6. Look for: emotional moments, funny moments, controversial opinions, surprising facts, high energy delivery

RESPOND WITH ONLY THIS JSON FORMAT, nothing else:
{"clips":[{"title":"catchy title","start":START_SECONDS,"end":END_SECONDS,"why":"reason this clip would go viral","hook":"the opening line"}]}

EXAMPLE (do not use these values, use real timestamps from the transcript):
{"clips":[{"title":"Example Title","start":45.0,"end":105.0,"why":"emotional moment","hook":"opening words"}]}

IMPORTANT: "start" and "end" are numbers in seconds. Each clip must be at least 30 seconds: end minus start >= 30.

TRANSCRIPT:
%s`, totalDuration, transcriptText)
}

func buildPromptForLocalModel(transcriptText string, totalDuration float64) string {
	return fmt.Sprintf(`Find 3-5 clips from this video transcript for TikTok/YouTube Shorts.

CRITICAL RULES:
- Each clip MUST be between 30 and 90 seconds long
- end minus start MUST be >= 30
- start and end are in seconds
- Pick moments with strong emotion, humor, drama, or controversy
- Each clip needs a complete thought, never cut mid-sentence

The video is %.0f seconds long. Timestamps go from 0 to %.0f.

RESPOND WITH ONLY THIS JSON:
{"clips":[{"title":"title","start":0.0,"end":60.0,"why":"reason","hook":"opening line"}]}

EXAMPLE with correct duration (notice end minus start = 60):
{"clips":[{"title":"The Big Reveal","start":19.0,"end":79.0,"why":"dramatic moment","hook":"You see when I called you last night"}]}

DO NOT make clips shorter than 30 seconds. This is the most important rule.

TRANSCRIPT:
%s`, totalDuration, totalDuration, transcriptText)
}

func validateClips(clips []Clip, totalDuration float64) []Clip {
	var valid []Clip
	for _, clip := range clips {
		duration := clip.End - clip.Start

		if duration < 15 {
			fmt.Printf("    ⚠️  Skipping '%s' (%.0fs too short)\n", clip.Title, duration)
			continue
		}

		if clip.Start < 0 {
			clip.Start = 0
		}
		if clip.End > totalDuration {
			clip.End = totalDuration
		}

		if clip.Start >= clip.End {
			continue
		}

		valid = append(valid, clip)
	}
	return valid
}

func sanitizeJSON(input string) string {
	start := strings.Index(input, "{")
	end := strings.LastIndex(input, "}")
	if start != -1 && end != -1 && start < end {
		return input[start : end+1]
	}
	return strings.TrimSpace(input)
}

func truncateTranscript(transcriptText string, maxChars int) string {
	if len(transcriptText) <= maxChars {
		return transcriptText
	}

	third := maxChars / 3
	head := transcriptText[:third]
	tail := transcriptText[len(transcriptText)-third:]

	return head + "\n\n[... middle section omitted for length ...]\n\n" + tail
}
