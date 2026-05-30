package transcriber

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Word struct {
	Text  string  `json:"word"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

type Segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
	Words []Word  `json:"words"`
}

type Transcript struct {
	Segments []Segment `json:"segments"`
	Text     string    `json:"text"`
}

func DownloadAudio(url, outputDir string) (string, error) {
	os.MkdirAll(outputDir, 0755)
	outputPath := filepath.Join(outputDir, "audio.mp3")

	cmd := exec.Command("yt-dlp",
		"-x", "--audio-format", "mp3",
		"--no-warnings",
		"-o", outputPath,
		url,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to download audio: %w", err)
	}
	return outputPath, nil
}

func Transcribe(audioPath, outputDir string) (*Transcript, error) {
	cmd := exec.Command("whisper",
		audioPath,
		"--model", "base",
		"--output_format", "json",
		"--output_dir", outputDir,
		"--word_timestamps", "True",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("whisper failed: %w", err)
	}

	jsonPath := filepath.Join(outputDir, "audio.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read transcript: %w", err)
	}

	var transcript Transcript
	if err := json.Unmarshal(data, &transcript); err != nil {
		return nil, fmt.Errorf("failed to parse transcript: %w", err)
	}

	return &transcript, nil
}