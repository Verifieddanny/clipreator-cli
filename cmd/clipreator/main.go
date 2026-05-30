package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Verifieddanny/clipreator-cli/internal/analyzer"
	"github.com/Verifieddanny/clipreator-cli/internal/captioner"
	"github.com/Verifieddanny/clipreator-cli/internal/cutter"

	"github.com/Verifieddanny/clipreator-cli/internal/transcriber"
)

func main() {
	var url string
	flag.StringVar(&url, "url", "", "YouTube URL to process")
	flag.Parse()

	if url == "" {
		fmt.Println("URL is required")
		os.Exit(1)
	}

	workDir := ".clipreator_tmp"

	fmt.Println("📥 Downloading audio...")
	audioPath, err := transcriber.DownloadAudio(url, workDir)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Audio saved to: %s\n\n", audioPath)

	fmt.Println("🎙️ Transcribing with Whisper...")
	transcript, err := transcriber.Transcribe(audioPath, workDir)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ Transcription complete! %d segments found\n\n", len(transcript.Segments))

	for i, seg := range transcript.Segments {
		if i < 5 {
			fmt.Printf("[%06.2f -> %06.2f] %s\n", seg.Start, seg.End, seg.Text)
		}
	}
	if len(transcript.Segments) > 5 {
		fmt.Printf("\n... and %d more segments\n", len(transcript.Segments)-5)
	}

	fmt.Println("🧠 Analyzing transcript for viral clips...")
	result, err := analyzer.Analyze(transcript)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🎬 Found %d potential clips:\n\n", len(result.Clips))
	for i, clip := range result.Clips {
		minutes := int(clip.Start) / 60
		seconds := int(clip.Start) % 60
		endMin := int(clip.End) / 60
		endSec := int(clip.End) % 60
		duration := clip.End - clip.Start

		fmt.Printf("  Clip %d: %s\n", i+1, clip.Title)
		fmt.Printf("  ⏱️  %02d:%02d → %02d:%02d (%.0fs)\n", minutes, seconds, endMin, endSec, duration)
		fmt.Printf("  🎣 Hook: %s\n", clip.Hook)
		fmt.Printf("  🔥 Why: %s\n\n", clip.Why)
	}

	fmt.Println("📥 Downloading full video...")
	videoPath, err := cutter.DownloadVideo(url, workDir)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Video saved to: %s\n\n", videoPath)

	fmt.Println("✂️  Cutting clips...")
	clipPaths, err := cutter.CutClips(videoPath, workDir, result.Clips)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Step 4: Burn captions onto each clip
	fmt.Println("\n💬 Adding captions...")
	clipsDir := filepath.Join(workDir, "clips")
	var finalPaths []string

	for i, clipPath := range clipPaths {
		clip := result.Clips[i]
		fmt.Printf("  💬 Captioning clip %d: %s\n", i+1, clip.Title)

		words := captioner.ExtractWords(transcript, clip.Start, clip.End)
		if len(words) == 0 {
			fmt.Printf("    ⚠️  No word timestamps found, skipping captions\n")
			finalPaths = append(finalPaths, clipPath)
			continue
		}
		fmt.Printf("    📝 %d words with timestamps\n", len(words))

		captionedPath := filepath.Join(clipsDir, fmt.Sprintf("final_%d.mp4", i+1))
		if err := captioner.BurnCaptions(clipPath, words, captionedPath); err != nil {
			fmt.Printf("    ⚠️  Failed to burn captions: %v\n", err)
			finalPaths = append(finalPaths, clipPath)
			continue
		}

		os.Remove(clipPath)
		os.Rename(captionedPath, clipPath)
		fmt.Printf("    ✅ Captions burned!\n")
		finalPaths = append(finalPaths, clipPath)
	}

	fmt.Printf("\n🎉 Done! %d clips ready:\n", len(clipPaths))
	for _, p := range clipPaths {
		fmt.Printf("  📎 %s\n", p)
	}

}
