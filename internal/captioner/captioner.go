package captioner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Verifieddanny/clipreator-cli/internal/transcriber"
)

type CaptionWord struct {
	Text  string
	Start float64
	End   float64
}

// ExtractWords pulls word-level timestamps for a clip's time range
func ExtractWords(transcript *transcriber.Transcript, clipStart, clipEnd float64) []CaptionWord {
	var words []CaptionWord
	for _, seg := range transcript.Segments {
		if seg.End < clipStart || seg.Start > clipEnd {
			continue
		}
		for _, w := range seg.Words {
			if w.Start >= clipStart && w.End <= clipEnd {
				words = append(words, CaptionWord{
					Text:  strings.TrimSpace(w.Text),
					Start: w.Start - clipStart,
					End:   w.End - clipStart,
				})
			}
		}
	}
	return words
}

// groupWords splits words into display chunks of 3-4 words,
// breaking on natural pauses
func groupWords(words []CaptionWord) [][]CaptionWord {
	var groups [][]CaptionWord
	var current []CaptionWord

	for i, w := range words {
		current = append(current, w)

		isLast := i == len(words)-1
		hasGap := !isLast && words[i+1].Start-w.End > 0.4
		isFull := len(current) >= 4

		if isLast || hasGap || isFull {
			group := make([]CaptionWord, len(current))
			copy(group, current)
			groups = append(groups, group)
			current = nil
		}
	}
	return groups
}

func formatASSTime(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	h := int(seconds) / 3600
	m := (int(seconds) % 3600) / 60
	s := int(seconds) % 60
	cs := int((seconds - float64(int(seconds))) * 100)
	return fmt.Sprintf("%d:%02d:%02d.%02d", h, m, s, cs)
}

// GenerateASS creates an ASS subtitle file with word-by-word highlighting
func GenerateASS(words []CaptionWord, outputPath string) error {
	header := `[Script Info]
ScriptType: v4.00+
PlayResX: 1080
PlayResY: 1920
WrapStyle: 0

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Default,Arial Black,68,&H00FFFFFF,&H000000FF,&H00000000,&H80000000,-1,0,0,0,100,100,2,0,1,5,0,2,40,40,180,1
Style: Highlight,Arial Black,68,&H0000FFFF,&H000000FF,&H00000000,&H80000000,-1,0,0,0,100,100,2,0,1,5,0,2,40,40,180,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
`

	groups := groupWords(words)
	var events strings.Builder
	events.WriteString(header)

	for _, group := range groups {
		if len(group) == 0 {
			continue
		}
		// groupStart := group[0].Start
		groupEnd := group[len(group)-1].End

		// For each word in the group, create an event highlighting just that word
		for wi, word := range group {
			var line strings.Builder
			for wj, w := range group {
				if wj > 0 {
					line.WriteString(" ")
				}
				if wj == wi {
					// Highlighted word (yellow)
					fmt.Fprintf(&line, `{\c&H00FFFF&\b1}%s{\c&HFFFFFF&\b1}`, strings.ToUpper(w.Text))
				} else {
					line.WriteString(strings.ToUpper(w.Text))
				}
			}

			start := word.Start
			var end float64
			if wi < len(group)-1 {
				end = group[wi+1].Start
			} else {
				end = groupEnd
			}

			// Clamp minimum duration
			if end-start < 0.1 {
				end = start + 0.1
			}

			fmt.Fprintf(&events,
				"Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n",
				formatASSTime(start),
				formatASSTime(end),
				line.String(),
			)
		}

		// Small gap between groups for readability
		_ = groupEnd
	}

	return os.WriteFile(outputPath, []byte(events.String()), 0644)
}

// BurnCaptions overlays captions onto a video using drawtext filters
// BurnCaptions overlays ASS subtitles onto a video
func BurnCaptions(videoPath string, words []CaptionWord, outputPath string) error {
	// Generate ASS file
	assPath := outputPath + ".ass"
	if err := GenerateASS(words, assPath); err != nil {
		return fmt.Errorf("failed to generate ASS: %w", err)
	}
	defer os.Remove(assPath)

	absASS, err := filepath.Abs(assPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Escape for ffmpeg filter: colons, brackets, single quotes
	escaped := strings.ReplaceAll(absASS, "\\", "/")
	escaped = strings.ReplaceAll(escaped, ":", "\\:")
	escaped = strings.ReplaceAll(escaped, "'", "'\\''")
	escaped = strings.ReplaceAll(escaped, "[", "\\[")
	escaped = strings.ReplaceAll(escaped, "]", "\\]")

	cmd := exec.Command("ffmpeg", "-y",
		"-i", videoPath,
		"-vf", "ass="+escaped,
		"-c:v", "libx264", "-preset", "fast", "-crf", "23",
		"-c:a", "copy",
		outputPath,
	)
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
