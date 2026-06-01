# Clipreator

Turn long YouTube videos into viral short-form clips with one command.

Clipreator downloads a YouTube video, transcribes it with Whisper, uses Claude AI to find the most engaging moments, then auto-edits them into vertical clips with captions — ready for TikTok, Reels, and Shorts.

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
3. **Finds the best clips** by sending the transcript to Claude, which identifies the 3-5 most viral-worthy 30-90 second segments
4. **Downloads the full video** and cuts at the AI-selected timestamps
5. **Crops to 9:16 vertical** — center crop optimized for short-form platforms
6. **Removes dead space** — detects silence gaps and cuts them out for tighter pacing
7. **Burns word-by-word captions** — animated highlighting synced to speech, TikTok-style

## Requirements

- **Go** 1.21+
- **Python** 3.8+ (for Whisper)
- **ffmpeg** with libass and libfreetype (for captions)
- **yt-dlp** (for downloading)
- **Anthropic API key**

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

### 2. Set your API key

```bash
export ANTHROPIC_API_KEY=your-key-here
```

Sign up at [console.anthropic.com](https://console.anthropic.com) if you don't have one.

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
┌──────────────┐
│  Claude AI   │  Pick best 3-5 viral moments
└──────┬───────┘
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
│  ffmpeg/ASS  │  Burn word-by-word captions
└──────┬───────┘
       │
       ▼
   Ready clips
```

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
│   │   └── analyzer.go      # Claude AI clip detection
│   ├── cutter/
│   │   └── cutter.go        # Video cutting, cropping, silence removal
│   └── captioner/
│       └── captioner.go     # ASS subtitle generation + burn
├── go.mod
└── README.md
```

## Tech Stack

- **Go** — orchestration and CLI
- **OpenAI Whisper** — local speech-to-text with word-level timestamps
- **Claude API** — AI-powered highlight detection
- **ffmpeg** — video processing, cropping, caption rendering
- **yt-dlp** — YouTube downloading

## Roadmap

- [ ] Ollama support (fully local, no API key needed)
- [ ] Custom caption styles (font, color, position)
- [ ] Face detection for smarter cropping
- [ ] Config file for editable settings
- [ ] Direct upload to TikTok/YouTube/Instagram

## License

MIT

## Author

Built by [Daniel Nwachukwu](https://github.com/Verifieddanny)
