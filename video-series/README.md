# Process Compose video series

A ten part video course: presentation deck, voiceover scripts, and generated
narration audio. Every terminal image in the deck is a real capture from a real
run of `bin/process-compose`, not a mockup.

```
process-compose-course.html              the deck (open this)
process-compose-course.single-file.html  same deck, audio inlined, 20 MB
video/                                   rendered MP4s, one per episode, upload ready
SCRIPTS.md                               all ten scripts, with timings and cues
audio/                                   narration, one clip per beat + one per episode
content/ep01..ep10.json                  source of truth: narration + slides
demo/                                    the linkstash stack used throughout
tui/                                     captured ANSI frames + frames.json
scripts/                                 the toolchain
```

## The series

| # | episode | slides | runtime | file |
|---|---------|--------|---------|------|
| 1 | What Process Compose Is | 8 | 2:47 | `ep01-what-it-is.mp4` |
| 2 | Install It and Write Your First Config | 14 | 4:30 | `ep02-first-config.mp4` |
| 3 | Dependencies and Startup Order | 16 | 4:50 | `ep03-dependencies.mp4` |
| 4 | Health Checks That Mean Something | 12 | 4:08 | `ep04-health-checks.mp4` |
| 5 | Restart Policies and Clean Shutdown | 15 | 4:42 | `ep05-restart-shutdown.mp4` |
| 6 | The Terminal Interface in Depth | 18 | 5:12 | `ep06-tui.mp4` |
| 7 | Environment Variables and Templating | 20 | 6:15 | `ep07-config.mp4` |
| 8 | Namespaces, Replicas and Schedules | 16 | 4:19 | `ep08-namespaces-replicas.mp4` |
| 9 | The CLI and the REST API | 18 | 4:28 | `ep09-cli-api.mp4` |
| 10 | CI Pipelines and the MCP Server | 15 | 4:24 | `ep10-ci-and-mcp.mp4` |

152 slides, 44:27 of narration, 45:39 of finished video, 294 MB.

## Using the deck

Open `process-compose-course.html` in a browser. It needs the `audio/` folder
next to it. If you want one file with nothing beside it, use
`process-compose-course.single-file.html` instead, which has the audio embedded
as data URIs and is about 20 MB.

| key | |
|---|---|
| `space` / `→` | next slide |
| `←` | previous |
| `P` | play or pause the narration |
| `[` `]` | previous / next episode |
| `E` | episode picker |
| `S` | script panel, click any line to jump to that slide |
| `F` | full screen |

With playback on, slides advance automatically when each clip finishes, so the
deck plays through on its own. `#ep7.4` in the URL jumps straight to episode 7,
slide 4.

## Rendering the videos

```shell
python3 scripts/render-video.py            # all ten, into video/
python3 scripts/render-video.py 03 07      # only these two
python3 scripts/render-video.py --jobs 8   # more browsers at once
```

Needs `google-chrome`, `ffmpeg` and the `websockets` Python package. Each
episode produces three files:

```
video/ep03-dependencies.mp4         1920x1080, 30fps, H.264 High + AAC
video/ep03-dependencies.txt         YouTube description with chapter markers
video/ep03-dependencies-thumb.png   1280x720 title card
```

Rather than screen recording, the renderer drives the deck in headless Chrome
and steps every CSS animation by hand through the Web Animations API, one
screenshot per output frame at an exact animation time. Nothing depends on how
fast the capture happens to run, so the motion in the file is the motion in the
browser and two runs produce identical output.

Only the moving part of a slide is captured. Once the entrance animation has
settled, a slide with nothing looping on it is one still held for the rest of
the narration, and a slide that does have something looping gets a single two
second cycle repeated. That cycle is seamless because the renderer retimes the
looping animations so their periods all divide two seconds. About 12,000
screenshots cover roughly 83,000 output frames.

Audio is normalised to -14 LUFS with a true peak ceiling of -1.5 dBTP, which is
what YouTube normalises to, so the episodes will not play quiet next to
everything else on the platform. Chapters are embedded in the MP4 and repeated
as timestamps in the description file, which is where YouTube actually reads
them from.

`scripts/timing.py` holds the two numbers that set the pace: how long a slide is
on screen before its narration starts, and how long it stays after. `SCRIPTS.md`
timestamps are generated from the same constants, so they match the videos.

## Publishing

`video/UPLOAD.md` collects the titles, descriptions, thumbnails and shared
settings in series order, so uploading is copy and paste. Refresh it on its own
with `python3 scripts/render-video.py --notes`.

Upload through YouTube Studio rather than the Data API. Two things make the API
the slower route for a one time series: videos uploaded by an API project that
has not passed Google's compliance audit are locked to private and have to be
published by hand anyway, and a video insert costs 1600 units against a default
quota of 10,000 a day, which caps you at six of these ten per day.

## Text to speech

The audio was generated with [`edge-tts`](https://github.com/rany2/edge-tts),
which is free, needs no account and no API key. It uses the neural voices behind
Microsoft Edge's read aloud feature. Install it with:

```shell
uv tool install edge-tts     # or: pipx install edge-tts
```

The voice is `en-US-BrianMultilingualNeural`, chosen because it reads technical
prose as speech rather than as an announcement. Regenerate everything with:

```shell
python3 scripts/gen-audio.py            # all episodes
python3 scripts/gen-audio.py 03 07      # only these two
```

Clips are keyed by a hash of the voice, rate, pitch and text, so a re-run after
editing one paragraph only regenerates that paragraph.

To try a different voice or pace:

```shell
edge-tts --list-voices | grep en-US
PC_TTS_VOICE=en-US-AndrewMultilingualNeural PC_TTS_RATE=-6% python3 scripts/gen-audio.py
```

Worth auditioning: `en-US-AndrewMultilingualNeural` (warmer),
`en-US-ChristopherNeural` (more formal), `en-GB-RyanNeural` (British).

Other free options if you want to compare: Piper (fully offline, smaller
voices), Kokoro (offline, very good, needs Python and a model download), and the
free tiers at Murf or Play.ht. ElevenLabs has a free tier too, but it is capped
low and would not cover 44 minutes.

If you would rather record it yourself, `SCRIPTS.md` has every line with its
on screen cue and a running timestamp.

## Editing the scripts

`content/epNN.json` is the source. Each beat is one paragraph of narration plus
one slide:

```json
{
  "narration": "what gets spoken",
  "visual": { "type": "code", "lang": "yaml", "code": "...", "highlight": [3, 4] }
}
```

Slide types: `title`, `points`, `keys`, `table`, `conditions`, `callout`,
`quote`, `compare`, `code`, `cmd`, `tui`, `cmdframe`, `diagram`, `bigtext`.
`tui` and `cmdframe` reference a captured frame by name from `tui/`.

After editing:

```shell
python3 scripts/gen-audio.py      # regenerates only what changed
python3 scripts/export-scripts.py # refresh SCRIPTS.md
python3 scripts/build-deck.py     # rebuild the deck
python3 scripts/build-deck.py --embed   # ...and the single file version
```

## Re-capturing the terminal

`scripts/capture-tui.sh` starts the demo stack inside a detached tmux pane at a
fixed 120x34, drives the interface with real keystrokes, and captures each
screen with `tmux capture-pane -e`. CLI output is captured separately under
`script` so the commands see a tty and emit colour.

```shell
./scripts/capture-tui.sh              # writes tui/*.ansi
python3 scripts/ansi2html.py          # converts to tui/frames.json
python3 scripts/build-deck.py
```

One thing to know if you edit that script: every keystroke needs a delay after
it. Sending two keys back to back makes tcell merge the escape sequences and
the second key gets swallowed.

The capture also refuses to start if ports 8110, 8111 or 8099 are already
in use, which happens if a previous run was killed and left orphaned children.

## The demo stack

`demo/` is a small link shortener called linkstash. Every process is a real
program, so the health checks really pass and the dependency ordering really
happens.

| process | namespace | shows |
|---|---|---|
| `migrate` | backend | a one shot task, `process_completed_successfully` |
| `api` | backend | HTTP readiness and liveness probes, `restart: always` |
| `worker` | backend | `replicas: 2`, waits for `process_healthy` |
| `mailer` | backend | `ready_log_line` |
| `web` | frontend | depends on the API being healthy |
| `smoke-test` | tools | a dependent one shot check |
| `vacuum` | tools | `schedule.interval` |
| `docs` | tools | `disabled: true` |

```shell
cd demo && process-compose up
```

It needs nothing but Python 3 and sqlite3.

## Regenerating from scratch

```shell
make build                            # from the repo root
./video-series/scripts/capture-tui.sh
python3 video-series/scripts/ansi2html.py
python3 video-series/scripts/gen-audio.py
python3 video-series/scripts/export-scripts.py
python3 video-series/scripts/build-deck.py --embed
python3 video-series/scripts/render-video.py
```

## Accuracy

The content was written against this working copy at `v1.120.0`, and the
non-obvious claims were verified by running them rather than taken from the
documentation. Two places where that mattered:

- Episode 4 says a failing **readiness** probe is what stops and restarts a
  process, and that `availability` alone decides whether it comes back. That is
  what `onReadinessCheckEnd` in `src/app/process.go` does, and it differs from
  the Kubernetes model people expect.
- Episode 7 describes what the loader actually substitutes. `envsubst`'s
  scanner only treats `$` as a substitution when the very next character is
  `{`, so `${VAR}` is replaced at file read time and a bare `$VAR` is passed
  through untouched. That is survivable in `command`, which runs through a
  shell, and silently broken everywhere else: `working_dir: "$HOME/project"`
  fails with `stat $HOME/project: no such file or directory`, because nothing
  ever expands it.
