# Clipreator

Turn long YouTube videos into viral short-form clips with one command.

Clipreator downloads a YouTube video, transcribes it with Whisper, uses AI to find the most engaging moments, then auto-edits them into vertical clips with captions — ready for TikTok, Reels, and Shorts.

## Demo

```bash
clipreator -url="https://youtu.be/your-video-id"
```

```
📥 Downloading audio...
✅ Audio saved to: .clipreator_tmp/audio.mp3

🎙️ Transcribing with Whisper...
✅ Transcription complete! 312 segments found

🧠 Analyzing transcript for viral clips...
    🧠 Using Claude...
🎬 Found 5 potential clips:

  Clip 1: We're Here To Take The Trophy From Them
  ⏱️  07:11 → 08:02 (51s)
  🎣 Hook: They are defending the trophy... and we are here to take that away from them.
  🔥 Why: Confident, aggressive take showing elite competitor mentality.

📥 Downloading full video...
✂️  Cutting clips...
💬 Adding captions...

🎉 Done! 5 clips ready:
  📎 .clipreator_tmp/clips/clip_1.mp4
  📎 .clipreator_tmp/clips/clip_2.mp4
  📎 .clipreator_tmp/clips/clip_3.mp4
  📎 .clipreator_tmp/clips/clip_4.mp4
  📎 .clipreator_tmp/clips/clip_5.mp4
```

## What It Does

1. **Downloads** the audio from any YouTube video using `yt-dlp`
2. **Transcribes** with OpenAI Whisper running locally — word-level timestamps included
3. **Finds the best clips** using AI (Claude, OpenAI, or Ollama) to identify the 3-5 most viral-worthy 30-90 second segments
4. **Downloads the full video** and cuts at the AI-selected timestamps
5. **Crops to 9:16 vertical** — center crop optimized for short-form platforms
6. **Removes dead space** — detects silence gaps and cuts them out for tighter pacing
7. **Burns word-by-word captions** — animated highlighting synced to speech, TikTok-style

## Requirements

- **Go** 1.21+
- **Python** 3.8+ (for Whisper)
- **ffmpeg** with libass and libfreetype (for captions)
- **yt-dlp** (for downloading)
- **AI backend** — choose one:
  - Ollama (free, local, no API key)
  - OpenAI API key
  - Anthropic API key (Claude)

## Installation

### 1. Install dependencies

```bash
# macOS
brew install yt-dlp
brew tap homebrew-ffmpeg/ffmpeg
brew install homebrew-ffmpeg/ffmpeg/ffmpeg

# Install Whisper
python3 -m venv .venv
source .venv/bin/activate
pip install openai-whisper
```

### 2. Choose your AI backend

**Option A: Free with Ollama (no API key needed)**

```bash
# Install Ollama
brew install ollama
# Or download from https://ollama.com/download

# Pull a model (Qwen 2.5 recommended for best local results)
ollama pull qwen2.5:7b

# Set the model (optional, defaults to qwen2.5:7b)
export OLLAMA_MODEL=qwen2.5:7b
```

That's it. No signup, no credit card. Runs entirely on your machine.

**Option B: OpenAI**

```bash
export OPENAI_API_KEY=your-key-here
```

Sign up at [platform.openai.com](https://platform.openai.com).

**Option C: Claude (best quality)**

```bash
export ANTHROPIC_API_KEY=your-key-here
```

Sign up at [console.anthropic.com](https://console.anthropic.com).

> **Priority:** If multiple keys are set, Clipreator uses Claude > OpenAI > Ollama automatically.

### 3. Build

```bash
git clone https://github.com/Verifieddanny/clipreator-cli.git
cd clipreator-cli
go build -o bin/clipreator ./cmd/clipreator/
```

## Usage

```bash
# Activate the Python environment (needed for Whisper)
source .venv/bin/activate

# Run on any YouTube video
./bin/clipreator -url="https://youtu.be/your-video-id"
```

Clips are saved to `.clipreator_tmp/clips/`.

### Environment Variables

| Variable | Description | Default |
|---|---|---|
| `ANTHROPIC_API_KEY` | Claude API key (best quality) | — |
| `OPENAI_API_KEY` | OpenAI API key | — |
| `OLLAMA_MODEL` | Ollama model name | `qwen2.5:7b` |
| `OLLAMA_HOST` | Ollama server URL | `http://localhost:11434` |

## How It Works

```
YouTube URL
    │
    ▼
┌──────────────┐
│  yt-dlp      │  Download audio
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Whisper     │  Transcribe with word timestamps
└──────┬───────┘
       │
       ▼
┌─────────────────────────┐
│  Claude / OpenAI / Ollama│  Pick best 3-5 viral moments
└──────┬──────────────────┘
       │
       ▼
┌──────────────┐
│  yt-dlp      │  Download full video
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  ffmpeg      │  Cut → Crop 9:16 → Remove silence
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  ffmpeg/ASS  │  Burn synced word-by-word captions
└──────┬───────┘
       │
       ▼
   Ready clips
```

## Local Models: What Works

Not all local models handle this task equally. Our testing:

| Model | Quality | Notes |
|---|---|---|
| **Qwen 2.5 7B** | ⭐⭐⭐⭐ | Best local option. Follows JSON format well, picks good timestamps |
| **Llama 3.1 8B** | ⭐⭐ | Often produces clips under 10 seconds. Struggles with timestamp math |
| **Mistral 7B** | ⭐⭐⭐ | Decent, occasionally misses the JSON format |

For best results, use Claude or OpenAI. For free local processing, Qwen 2.5 7B is the recommended choice.

## Project Structure

```
clipreator/
├── cmd/
│   └── clipreator/
│       └── main.go          # CLI entry point
├── internal/
│   ├── transcriber/
│   │   └── transcriber.go   # Audio download + Whisper transcription
│   ├── analyzer/
│   │   └── analyzer.go      # AI clip detection (Claude + OpenAI + Ollama)
│   ├── cutter/
│   │   └── cutter.go        # Video cutting, cropping, silence removal
│   └── captioner/
│       └── captioner.go     # ASS subtitle generation + burn
├── go.mod
├── Makefile
└── README.md
```

## Tech Stack

- **Go** — orchestration and CLI
- **OpenAI Whisper** — local speech-to-text with word-level timestamps
- **Claude API / OpenAI API / Ollama** — AI-powered highlight detection
- **ffmpeg** — video processing, cropping, caption rendering
- **yt-dlp** — YouTube downloading

## Roadmap

- [x] OpenAI API support
- [x] Ollama support (fully local, no API key needed)
- [x] Caption sync with silence removal
- [x] Transcript truncation for local models
- [x] Auto-retry for local models
- [ ] Custom caption styles (font, color, position, animation)
- [ ] Config file for editable settings
- [ ] Face detection for smarter cropping
- [ ] Direct upload to TikTok/YouTube/Instagram

## Disclaimer

This tool is for personal and educational use. Users are responsible for ensuring they have the right to download and edit any content processed with Clipreator. Respect copyright laws and content creators' rights. Do not use this tool to repost others' content without permission.

## License

MIT

## Author

Built by [DevDanny](https://github.com/Verifieddanny)