---
date:
    created: 2026-08-12
tags:
    - video
    - tutorial
    - workflow
---

# A Ten Part Video Course for Process Compose

There is now a video course for Process Compose. Ten episodes, about 45 minutes
in total, and no single episode runs longer than six.

[Watch the playlist](https://www.youtube.com/playlist?list=PLY3UbHll3zOI)

<!-- more -->

## Why a course

These docs are organised by feature, which is the right shape once you know
which feature you need. A course can go in the other order. Each episode starts
from something you are trying to do, introduces the part of Process Compose that
does it, and then shows it running.

## The episodes

| # | episode | runtime |
|---|---------|---------|
| 1 | [What Process Compose Is](https://www.youtube.com/watch?v=52Wfz4I9jdY&list=PLY3UbHll3zOI) | 2:48 |
| 2 | [Install It and Write Your First Config](https://www.youtube.com/watch?v=7peoWOViunY&list=PLY3UbHll3zOI) | 4:31 |
| 3 | [Dependencies and Startup Order](https://www.youtube.com/watch?v=4QN7YfdHVGw&list=PLY3UbHll3zOI) | 4:51 |
| 4 | [Health Checks That Mean Something](https://www.youtube.com/watch?v=j60ObpVd6Mg&list=PLY3UbHll3zOI) | 4:09 |
| 5 | [Restart Policies and Clean Shutdown](https://www.youtube.com/watch?v=Q74xr5LMiD8&list=PLY3UbHll3zOI) | 4:43 |
| 6 | [The Terminal Interface in Depth](https://www.youtube.com/watch?v=K54l1oVFsQA&list=PLY3UbHll3zOI) | 5:13 |
| 7 | [Environment Variables and Templating](https://www.youtube.com/watch?v=gURvcOjf5Qs&list=PLY3UbHll3zOI) | 6:16 |
| 8 | [Namespaces, Replicas and Schedules](https://www.youtube.com/watch?v=Ivz2Ow8p820&list=PLY3UbHll3zOI) | 4:20 |
| 9 | [The CLI and the REST API](https://www.youtube.com/watch?v=Fgi7CVgjY1o&list=PLY3UbHll3zOI) | 4:29 |
| 10 | [CI Pipelines and the MCP Server](https://www.youtube.com/watch?v=qyb5bTRuRHk&list=PLY3UbHll3zOI) | 4:25 |

Every episode has chapter markers, so the playlist doubles as something to jump
into when you need one specific answer.

## Built around the questions people ask

Most of the running time goes to questions that have come up again and again in
issues and discussions over the years, and to the small gotchas that are easy to
read past the first time.

Episode 4 walks through the probe model end to end: which probe acts when a
check starts failing, and how the `availability` block decides what happens
next. Useful if you are arriving from Kubernetes, where the division of labour
between liveness and readiness is drawn differently.

Episode 7 puts the two substitution systems side by side and shows when each one
runs. `${VAR}` is resolved by the loader when the file is read, from your `.env`
and the environment Process Compose was launched with. `$${VAR}` is passed
through so the shell expands it at process start, with the process environment in
place. The episode runs both and prints the results next to each other, which is
the quickest way to see which one you want.

Episode 3 does the same for the four dependency conditions, and episode 5 for
restart policies, backoff, exit codes and shutting a project down cleanly.

## How the videos were made

The whole pipeline is scripted, so correcting one sentence regenerates one
narration clip and one video instead of asking for a re-record.

The slides are a single standalone HTML file, hand written, with no presentation
framework. Layout is in CSS container query units so it scales to any output
resolution, and the diagrams are inline SVG.

The terminal frames come from a running instance rather than a mockup. A script
starts the demo stack in a detached tmux pane at a fixed size, drives the TUI
with real keystrokes, and captures each screen as ANSI:

```shell
tmux new-session -d -s pcshots -x 120 -y 34 -c demo \
  "TERM=tmux-256color process-compose up -p 8099"
tmux send-keys -t pcshots F3     # the process info dialog
tmux capture-pane -p -e -t pcshots > tui/05-process-info.ansi
```

A converter turns that ANSI into HTML the deck embeds, so the terminal in the
video is real text with real colours rather than a screenshot. One thing worth
knowing if you try this: every keystroke needs a settle delay after it. Send two
keys back to back and tcell merges the escape sequences, so the second key gets
swallowed.

The narration is text to speech, using [edge-tts](https://github.com/rany2/edge-tts),
which is free and needs no account or API key:

```shell
uv tool install edge-tts
edge-tts --voice en-US-BrianMultilingualNeural --rate=-3% \
         --text "Process Compose runs several programs at once on one machine." \
         --write-media ep01-00.mp3
```

Write `--rate=-3%` with the equals sign. Passing `--rate -3%` makes argparse read
the value as another flag and the command fails.

Rendering is headless Chrome plus ffmpeg. Instead of screen recording the deck,
the renderer drives it over the Chrome DevTools Protocol, pauses every CSS
animation, and steps each one to an exact time through the Web Animations API,
taking one screenshot per output frame. Capture speed then has no bearing on
playback speed, and two runs produce identical files. ffmpeg muxes the frames
against the narration and normalises the audio to -14 LUFS, which is what YouTube
normalises to, so the episodes do not play quiet next to everything else.

Five commands rebuild the series from scratch:

```shell
./scripts/capture-tui.sh          # drive the demo stack, capture the TUI
python3 scripts/ansi2html.py      # ANSI into embeddable HTML
python3 scripts/gen-audio.py      # narration, one clip per paragraph
python3 scripts/build-deck.py     # assemble the deck
python3 scripts/render-video.py   # deck into ten MP4s
```

## Scope

The course is also clear about where Process Compose is not the answer. It is a
single machine orchestrator aimed at local development, integration tests in CI,
and running a handful of services on one box. It does not schedule across hosts
and it is not trying to replace systemd for long lived production services. If
you need a cluster, you need something else.

The demo stack used throughout is a small link shortener with a migration step,
an API, two workers, a mail relay and a static frontend, spread across three
namespaces. It runs on nothing but Python 3 and sqlite3.
