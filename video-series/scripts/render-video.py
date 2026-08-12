#!/usr/bin/env python3
"""Render each episode of the course to a YouTube ready MP4.

The presentation is driven in headless Chrome. Rather than screenshotting in
wall clock time, which would play the CSS animations back at whatever rate the
capture happens to run at, every animation on the slide is paused and stepped
by hand through the Web Animations API. One screenshot per output frame, at an
exact animation time, so the motion in the file is the motion in the browser.

Only the moving part of a slide is captured. Once the entrance animation has
settled, a slide with nothing looping on it is a single still held for the rest
of the narration. A slide that does have something looping gets one two second
cycle captured and repeated, which is seamless because the loop lengths are
normalised to divide two seconds. That turns roughly 83,000 output frames into
roughly 12,000 screenshots.

  python3 scripts/render-video.py                 # all episodes
  python3 scripts/render-video.py 03 07           # just those two
  python3 scripts/render-video.py --jobs 2        # fewer browsers at once

Needs: google-chrome, ffmpeg, and the python websockets package.
Reads:  process-compose-course.html, content/ep*.json, audio/*.mp3
Writes: video/epNN-<slug>.mp4 plus a description and a thumbnail for each.
"""

from __future__ import annotations

import argparse
import asyncio
import base64
import glob
import json
import math
import os
import shutil
import sys
import time
import urllib.request

try:
    import websockets
except ImportError:  # pragma: no cover
    sys.exit("needs the websockets package: pip install websockets")

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.normpath(os.path.join(HERE, ".."))
sys.path.insert(0, HERE)
from timing import LEAD_S, TAIL_S  # noqa: E402

DECK = os.path.join(ROOT, "process-compose-course.html")
AUDIO = os.path.join(ROOT, "audio")
OUTDIR = os.path.join(ROOT, "video")

FPS = 30
WIDTH, HEIGHT = 1920, 1080
CRF = 17
X264_PRESET = "slow"
AUDIO_BITRATE = "192k"
LOUDNESS_I = -14.0          # what YouTube normalises to
LOUDNESS_TP = -1.5
LOUDNESS_LRA = 11.0
FADE_IN_S = 0.5
FADE_OUT_S = 0.8
AMBIENT_PERIOD_MS = 2000    # every looping animation is retimed to divide this
MAX_ENTRANCE_MS = 6000      # a safety net, no real slide comes close
MIN_CHAPTER_S = 25.0        # YouTube rejects chapters shorter than 10s
REPO = "https://github.com/F1bonacc1/process-compose"

# Render mode. The player chrome comes off and the stage is pinned to exactly
# 1920x1080 so the slide fills the frame with no letterboxing.
#
# The animation-duration overrides matter more than they look. march is 1.1s
# and blink is 1.05s in the deck, which share no useful common multiple with
# the 2s pulse. Retimed to 1s and 2s they all line up every two seconds, which
# is what makes the captured tail loop seamlessly. The speed change is a tenth
# of a second on a decorative dash, invisible in playback.
RENDER_CSS = """
#bar { display: none !important; }
#app { grid-template-rows: 1fr !important; }
#stagewrap { padding: 0 !important; }
.stage {
  width: %dpx !important; height: %dpx !important;
  max-width: none !important; aspect-ratio: auto !important;
  border: 0 !important; border-radius: 0 !important; box-shadow: none !important;
}
html, body { background: #0c0d14 !important; overflow: hidden !important; }
.drawer { display: none !important; }
.cursor { animation-duration: 1s !important; }
.dash   { animation-duration: 1s !important; }
.pulse  { animation-duration: 2s !important; }

/* The slide entrance is a transition in the deck, and the deck adds the "on"
   class inside a requestAnimationFrame, which runs before the frame's first
   style recalculation. The element therefore goes straight to its end state
   with no start value to transition from, and slides with no per item
   animation of their own (a title card, a pull quote) would be captured as a
   single still. Restating it as a keyframe animation always fires, and it
   animates the same two properties over the same duration and curve. */
.slide { transition: none !important; }
.slide.on { animation: vr-enter .34s cubic-bezier(.2, .7, .3, 1) both !important; }
@keyframes vr-enter {
  from { opacity: 0; transform: translateY(1.4cqw); }
  to   { opacity: 1; transform: none; }
}
""" % (WIDTH, HEIGHT)

# Page side helper. prep() lands on a slide and freezes every animation it
# started; seek() moves them all to an exact time. getAnimations() is collected
# once, after the deck has applied its "on" class, which is the point at which
# the entrance transition and all the staggered item animations exist.
RENDER_JS = """
window.__vr = {
  css: function (text) {
    var s = document.createElement("style");
    s.textContent = text;
    document.head.appendChild(s);
    return true;
  },

  prep: async function (hash) {
    if (location.hash === hash) location.hash = "#reset";
    location.hash = hash;
    await new Promise(function (r) { setTimeout(r, 0); });
    await new Promise(function (r) { requestAnimationFrame(r); });
    await new Promise(function (r) { requestAnimationFrame(r); });

    var all = document.getAnimations();
    window.__A = all;
    var entrance = 0, ambient = false;
    for (var i = 0; i < all.length; i++) {
      var t = all[i].effect.getTiming();
      var d = typeof t.duration === "number" ? t.duration : 0;
      if (t.iterations === Infinity) { ambient = true; continue; }
      var end = (t.delay || 0) + d * (t.iterations || 1) + (t.endDelay || 0);
      if (end > entrance) entrance = end;
    }
    for (var j = 0; j < all.length; j++) {
      try { all[j].pause(); all[j].currentTime = 0; } catch (e) {}
    }
    return { count: all.length, entrance: entrance, ambient: ambient,
             type: (document.querySelector(".slide") || {}).className || "" };
  },

  seek: function (ms) {
    var a = window.__A || [];
    for (var i = 0; i < a.length; i++) {
      try { a[i].currentTime = ms; } catch (e) {}
    }
    return true;
  }
};
true
"""


# ------------------------------------------------------------------ helpers --

def fmt_clock(sec: float) -> str:
    sec = int(sec)
    h, m, s = sec // 3600, (sec % 3600) // 60, sec % 60
    return "%d:%02d:%02d" % (h, m, s) if h else "%d:%02d" % (m, s)


async def run(*args: str, stderr: bool = False) -> str:
    """Run a tool to completion. ffmpeg reports on stderr, hence the flag."""
    p = await asyncio.create_subprocess_exec(
        *args, stdout=asyncio.subprocess.PIPE, stderr=asyncio.subprocess.PIPE)
    out, err = await p.communicate()
    if p.returncode:
        raise RuntimeError("%s failed (%d)\n%s" %
                           (args[0], p.returncode, err.decode()[-3000:]))
    return err.decode() if stderr else out.decode()


async def ffprobe_duration(path: str) -> float:
    out = await run("ffprobe", "-v", "error", "-show_entries", "format=duration",
                    "-of", "default=nokey=1:noprint_wrappers=1", path)
    return float(out.strip())


# ------------------------------------------------------------------- chrome --

class Chrome:
    """A minimal Chrome DevTools Protocol client, enough to drive the deck."""

    def __init__(self, port: int, profile: str):
        self.port, self.profile = port, profile
        self.proc = self.ws = None
        self._id = 0
        self._pending: dict[int, asyncio.Future] = {}
        self._waiters: dict[str, list[asyncio.Future]] = {}
        self._pump = None

    async def __aenter__(self):
        shutil.rmtree(self.profile, ignore_errors=True)
        self.proc = await asyncio.create_subprocess_exec(
            "google-chrome",
            "--headless=new", "--remote-debugging-port=%d" % self.port,
            "--user-data-dir=%s" % self.profile,
            "--no-sandbox", "--no-first-run", "--no-default-browser-check",
            "--disable-extensions", "--disable-background-networking",
            "--disable-sync", "--disable-component-update", "--mute-audio",
            "--hide-scrollbars", "--disable-gpu", "--disable-dev-shm-usage",
            "--window-size=%d,%d" % (WIDTH, HEIGHT),
            "--force-device-scale-factor=1", "--allow-file-access-from-files",
            "--font-render-hinting=none", "about:blank",
            stdout=asyncio.subprocess.DEVNULL, stderr=asyncio.subprocess.DEVNULL)

        target = None
        for _ in range(120):
            try:
                raw = await asyncio.to_thread(
                    lambda: urllib.request.urlopen(
                        "http://127.0.0.1:%d/json/list" % self.port, timeout=2).read())
                for t in json.loads(raw):
                    if t.get("type") == "page" and t.get("webSocketDebuggerUrl"):
                        target = t
                        break
            except Exception:
                pass
            if target:
                break
            await asyncio.sleep(0.25)
        if not target:
            await self.__aexit__(None, None, None)
            raise RuntimeError("chrome did not expose a page target on port %d" % self.port)

        # max_size off: a 1920x1080 png screenshot is well past the 1 MiB default.
        self.ws = await websockets.connect(target["webSocketDebuggerUrl"],
                                           max_size=None, ping_interval=None,
                                           close_timeout=2)
        self._pump = asyncio.create_task(self._read())
        return self

    async def __aexit__(self, *exc):
        if self._pump:
            self._pump.cancel()
        if self.ws:
            try:
                await self.ws.close()
            except Exception:
                pass
        if self.proc and self.proc.returncode is None:
            self.proc.kill()
            await self.proc.wait()
        shutil.rmtree(self.profile, ignore_errors=True)

    async def _read(self):
        try:
            async for raw in self.ws:
                m = json.loads(raw)
                if "id" in m:
                    fut = self._pending.pop(m["id"], None)
                    if fut and not fut.done():
                        if "error" in m:
                            fut.set_exception(RuntimeError(m["error"].get("message")))
                        else:
                            fut.set_result(m.get("result", {}))
                else:
                    for fut in self._waiters.pop(m.get("method"), []):
                        if not fut.done():
                            fut.set_result(m.get("params"))
        except asyncio.CancelledError:
            raise
        except Exception as exc:
            for fut in list(self._pending.values()):
                if not fut.done():
                    fut.set_exception(exc)

    async def send(self, method: str, **params):
        self._id += 1
        fut = asyncio.get_running_loop().create_future()
        self._pending[self._id] = fut
        await self.ws.send(json.dumps({"id": self._id, "method": method,
                                       "params": params}))
        return await asyncio.wait_for(fut, timeout=120)

    def on(self, method: str) -> asyncio.Future:
        fut = asyncio.get_running_loop().create_future()
        self._waiters.setdefault(method, []).append(fut)
        return fut

    async def evaluate(self, expression: str):
        r = await self.send("Runtime.evaluate", expression=expression,
                            awaitPromise=True, returnByValue=True)
        if "exceptionDetails" in r:
            d = r["exceptionDetails"]
            raise RuntimeError("page JS: %s" % (
                d.get("exception", {}).get("description") or d.get("text")))
        return r["result"].get("value")

    async def screenshot(self) -> bytes:
        r = await self.send("Page.captureScreenshot", format="png",
                            optimizeForSpeed=True, captureBeyondViewport=False)
        return base64.b64decode(r["data"])


# -------------------------------------------------------------------- audio --

async def build_audio(ep: dict, meta: dict, work: str) -> tuple[str, list[float]]:
    """Pad every narration clip, glue them together, normalise the result.

    Returns the episode wav and the exact on screen duration of each beat.
    Durations come from the padded wavs rather than the manifest so the video
    timeline is locked to the samples that actually exist.
    """
    parts, durations = [], []
    for i, beat in enumerate(meta["beats"]):
        src = os.path.join(AUDIO, beat["file"])
        if not os.path.exists(src):
            raise RuntimeError("missing narration clip: %s" % src)
        dest = os.path.join(work, "a%03d.wav" % i)
        await run("ffmpeg", "-y", "-v", "error", "-i", src,
                  "-af", "adelay=%d:all=1,apad=pad_dur=%s" % (
                      round(LEAD_S * 1000), TAIL_S),
                  "-ar", "48000", "-ac", "2", "-c:a", "pcm_s16le", dest)
        parts.append(dest)
        durations.append(await ffprobe_duration(dest))

    listing = os.path.join(work, "audio.txt")
    with open(listing, "w", encoding="utf-8") as fh:
        for p in parts:
            fh.write("file '%s'\n" % p)
    joined = os.path.join(work, "joined.wav")
    await run("ffmpeg", "-y", "-v", "error", "-f", "concat", "-safe", "0",
              "-i", listing, "-c", "copy", joined)

    # Two pass loudnorm. The measured pass lets the second one apply a flat
    # gain (linear=true) instead of riding the level, which would breathe on a
    # voice this consistent and would also change the length.
    probe = await run("ffmpeg", "-hide_banner", "-nostats", "-i", joined,
                      "-af", "loudnorm=I=%s:TP=%s:LRA=%s:print_format=json" % (
                          LOUDNESS_I, LOUDNESS_TP, LOUDNESS_LRA),
                      "-f", "null", "-", stderr=True)
    stats = _last_json(probe)
    normalised = os.path.join(work, "audio.wav")
    await run(
        "ffmpeg", "-y", "-v", "error", "-i", joined,
        "-af", "loudnorm=I=%s:TP=%s:LRA=%s:measured_I=%s:measured_TP=%s:"
               "measured_LRA=%s:measured_thresh=%s:offset=%s:linear=true:"
               "print_format=summary" % (
                   LOUDNESS_I, LOUDNESS_TP, LOUDNESS_LRA,
                   stats["input_i"], stats["input_tp"], stats["input_lra"],
                   stats["input_thresh"], stats["target_offset"]),
        "-ar", "48000", "-ac", "2", "-c:a", "pcm_s16le", normalised)
    return normalised, durations


def _last_json(text: str) -> dict:
    """Pull loudnorm's json report out of ffmpeg's stderr chatter."""
    start = text.rfind("{")
    while start != -1:
        try:
            return json.loads(text[start:text.rfind("}") + 1])
        except json.JSONDecodeError:
            start = text.rfind("{", 0, start)
    raise RuntimeError("could not read the loudnorm report")


# ------------------------------------------------------------------ capture --

def plan_frames(entrance_ms: float, ambient: bool) -> tuple[list[float], int, int]:
    """Decide which animation times to screenshot for one slide.

    Returns the times to capture, the index the loop starts at, and its length.
    A loop length of zero means the last frame is simply held.
    """
    step = 1000.0 / FPS
    entrance = min(max(entrance_ms, 0.0), MAX_ENTRANCE_MS)
    if ambient:
        # Start the loop on a two second boundary at or after the entrance, so
        # by the time it begins every one shot animation has finished and the
        # frame at loop start is identical to the frame one period later.
        loop_start_ms = math.ceil(entrance / AMBIENT_PERIOD_MS) * AMBIENT_PERIOD_MS
        loop_len = int(round(AMBIENT_PERIOD_MS / step))
        loop_start = int(round(loop_start_ms / step))
        count = loop_start + loop_len
    else:
        count = int(math.ceil(entrance / step)) + 1
        loop_start, loop_len = count - 1, 0
    return [i * step for i in range(count)], loop_start, loop_len


async def capture_episode(ep: dict, frames_dir: str, port: int, profile: str,
                          log) -> list[dict]:
    """Screenshot every slide of one episode. Returns a plan per beat."""
    os.makedirs(frames_dir, exist_ok=True)
    plans = []
    async with Chrome(port, profile) as br:
        await br.send("Page.enable")
        await br.send("Runtime.enable")
        await br.send("Emulation.setDeviceMetricsOverride",
                      width=WIDTH, height=HEIGHT, deviceScaleFactor=1, mobile=False)
        loaded = br.on("Page.loadEventFired")
        await br.send("Page.navigate", url="file://" + DECK)
        await asyncio.wait_for(loaded, timeout=60)
        await asyncio.sleep(1.0)          # webfonts and the first layout
        await br.evaluate(RENDER_JS)
        await br.evaluate("window.__vr.css(%s)" % json.dumps(RENDER_CSS))
        await asyncio.sleep(0.3)

        shots = 0
        for b in range(len(ep["beats"])):
            info = await br.evaluate(
                "window.__vr.prep(%s)" % json.dumps("#ep%d.%d" % (ep["number"], b + 1)))
            times, loop_start, loop_len = plan_frames(info["entrance"], info["ambient"])
            files = []
            for i, t in enumerate(times):
                await br.evaluate("window.__vr.seek(%.4f)" % t)
                path = os.path.join(frames_dir, "b%03d_%04d.png" % (b, i))
                with open(path, "wb") as fh:
                    fh.write(await br.screenshot())
                files.append(path)
            shots += len(files)
            plans.append({"files": files, "loop_start": loop_start,
                          "loop_len": loop_len})
        log("captured %d frames over %d slides" % (shots, len(plans)))
    return plans


def write_concat(plans: list[dict], durations: list[float], path: str) -> float:
    """Write the ffmpeg concat list for a whole episode.

    Frame counts are worked out against a running total rather than per beat,
    so rounding to the frame grid cannot drift the picture away from the voice
    over the course of the episode.
    """
    step = 1.0 / FPS
    lines, elapsed, emitted = [], 0.0, 0
    for plan, dur in zip(plans, durations):
        elapsed += dur
        want = int(round(elapsed * FPS))
        files, start, length = plan["files"], plan["loop_start"], plan["loop_len"]
        for k in range(emitted, want):
            i = k - emitted
            if i < len(files):
                idx = i
            elif length:
                idx = start + (i - len(files)) % length
            else:
                idx = len(files) - 1
            lines.append("file '%s'\nduration %.6f" % (files[idx], step))
        emitted = want
    lines.append("file '%s'" % plans[-1]["files"][-1])   # concat ignores the last duration
    with open(path, "w", encoding="utf-8") as fh:
        fh.write("\n".join(lines) + "\n")
    return emitted * step


# ----------------------------------------------------------- youtube extras --

def chapter_title(v: dict, ep: dict) -> str:
    t = v["type"]
    if t == "title":
        return "Intro"
    if t in ("points", "keys", "table"):
        return v.get("heading", ep["title"])
    if t == "callout":
        return v.get("heading", "Watch out")
    if t == "code":
        return v.get("caption") or v.get("filename", "Configuration")
    if t == "cmd":
        return v["command"].split("\n")[0]
    if t == "cmdframe":
        return v["command"].split("\n")[0]
    if t == "tui":
        return v.get("caption", "In the terminal")
    if t == "diagram":
        return {
            "tabs": "Five terminal tabs",
            "tuilayout": "How the screen is laid out",
            "states": "The process lifecycle",
            "readygap": "Started is not ready",
            "probeflow": "What a failing probe does",
            "signals": "Shutdown signals",
            "twophase": "Two substitution passes",
            "clientserver": "Client and server",
        }.get(v["name"], v["name"])
    if t == "compare":
        return "%s vs %s" % (v["before"]["title"], v["after"]["title"])
    if t == "conditions":
        return "Dependency conditions"
    if t == "quote":
        return "Why not containers"
    if t == "bigtext":
        return "Up next"
    return t


def names_itself(v: dict) -> bool:
    """Whether this slide carries wording good enough to title a chapter."""
    t = v["type"]
    if t in ("code", "tui"):
        return bool(v.get("caption"))
    if t == "bigtext":
        return False
    return t in ("points", "keys", "table", "callout", "diagram", "compare",
                 "conditions", "cmd", "cmdframe", "quote")


def build_chapters(ep: dict, durations: list[float]) -> list[tuple[float, str]]:
    """Group beats into chapters at least MIN_CHAPTER_S apart.

    YouTube wants the first at 0:00, at least three of them, and none shorter
    than ten seconds. A beat that would only give a weak chapter name, a bare
    file name say, is passed over in favour of the next one that reads well,
    unless waiting would leave the chapter far too long.
    """
    chapters, clock = [], 0.0
    for beat, dur in zip(ep["beats"], durations):
        gap = clock - chapters[-1][0] if chapters else 0.0
        if not chapters:
            chapters.append((0.0, "Intro"))
        elif gap >= MIN_CHAPTER_S and (names_itself(beat["visual"])
                                       or gap >= MIN_CHAPTER_S * 2.5):
            title = chapter_title(beat["visual"], ep).strip()
            chapters.append((clock, (title[:57] + "...") if len(title) > 60 else title))
        clock += dur
    if len(chapters) > 1 and clock - chapters[-1][0] < 10.0:
        chapters.pop()
    return chapters if len(chapters) >= 3 else []


def write_description(ep: dict, chapters, total: float, dest: str) -> None:
    lines = ["Process Compose, episode %d: %s" % (ep["number"], ep["title"]), ""]
    lines.append(ep["subtitle"] + ".")
    lines.append("")
    lines.append(ep["beats"][0]["narration"])
    lines.append("")
    lines.append("Part of a ten episode series on Process Compose, a process "
                 "orchestrator for programs that are not in containers.")
    lines.append("")
    lines.append("Project and documentation: %s" % REPO)
    lines.append("")
    if chapters:
        lines.append("Chapters")
        for start, title in chapters:
            lines.append("%s %s" % (fmt_clock(start), title))
        lines.append("")
    lines.append("Runtime %s." % fmt_clock(total))
    with open(dest, "w", encoding="utf-8") as fh:
        fh.write("\n".join(lines) + "\n")


TAGS = ("process compose, process-compose, process orchestrator, devops, "
        "local development, developer tools, docker compose alternative, yaml, "
        "golang, cli, tui, ci pipeline, mcp server")


def write_upload_notes() -> str:
    """Collect everything an upload needs into one file, in series order.

    Reads the per episode description files rather than the render state, so it
    can be refreshed on its own with --notes.
    """
    out = ["# Uploading the series", "",
           "One section per episode, in the order they should go up. In each",
           "`video/*.txt` the first line is the title and everything after the",
           "blank line is the description, chapters included.", "",
           "Settings that apply to all ten:", "",
           "| field | value |", "|---|---|",
           "| category | Science & Technology |",
           "| language | English |",
           "| audience | not made for kids |",
           "| licence | standard YouTube licence |",
           "| playlist | Process Compose, add in episode order |",
           "| visibility | upload private, review, then publish |", "",
           "Tags:", "", "```", TAGS, "```", ""]

    files = sorted(glob.glob(os.path.join(OUTDIR, "ep*.txt")))
    for txt in files:
        stem = os.path.basename(txt)[:-4]
        lines = open(txt, encoding="utf-8").read().rstrip("\n").split("\n")
        title, body = lines[0], "\n".join(lines[1:]).strip("\n")
        out += ["---", "", "## %s" % title, "",
                "- video: `video/%s.mp4`" % stem,
                "- thumbnail: `video/%s-thumb.png`" % stem, "",
                "Description:", "", "```", body, "```", ""]

    dest = os.path.join(OUTDIR, "UPLOAD.md")
    with open(dest, "w", encoding="utf-8") as fh:
        fh.write("\n".join(out) + "\n")
    return dest


def write_ffmetadata(ep: dict, chapters, total: float, dest: str) -> None:
    out = [";FFMETADATA1",
           "title=Process Compose %02d. %s" % (ep["number"], ep["title"]),
           "artist=Process Compose", "genre=Education", ""]
    bounds = [c[0] for c in chapters] + [total]
    for i, (start, title) in enumerate(chapters):
        out += ["[CHAPTER]", "TIMEBASE=1/1000",
                "START=%d" % round(start * 1000),
                "END=%d" % round(bounds[i + 1] * 1000),
                "title=%s" % title, ""]
    with open(dest, "w", encoding="utf-8") as fh:
        fh.write("\n".join(out) + "\n")


# ------------------------------------------------------------------ episode --

async def render_episode(ep: dict, meta: dict, scratch: str, port: int,
                         keep: bool) -> dict:
    key = "%02d" % ep["number"]
    tag = "ep%s" % key

    def log(msg):
        print("  [%s] %s" % (tag, msg), flush=True)

    work = os.path.join(scratch, tag)
    frames_dir = os.path.join(work, "frames")
    shutil.rmtree(work, ignore_errors=True)
    os.makedirs(frames_dir, exist_ok=True)
    started = time.time()

    audio_path, durations = await build_audio(ep, meta, work)
    total = sum(durations)
    log("audio %s, %d clips" % (fmt_clock(total), len(durations)))

    plans = await capture_episode(ep, frames_dir, port,
                                  os.path.join(work, "profile"), log)

    concat = os.path.join(work, "frames.txt")
    video_len = write_concat(plans, durations, concat)

    chapters = build_chapters(ep, durations)
    os.makedirs(OUTDIR, exist_ok=True)
    stem = "%s-%s" % (tag, ep["slug"])
    mp4 = os.path.join(OUTDIR, stem + ".mp4")
    metafile = os.path.join(work, "meta.txt")
    write_ffmetadata(ep, chapters, total, metafile)
    write_description(ep, chapters, total, os.path.join(OUTDIR, stem + ".txt"))

    fade_out = max(0.0, total - FADE_OUT_S)
    await run(
        "ffmpeg", "-y", "-v", "error",
        "-f", "concat", "-safe", "0", "-i", concat,
        "-i", audio_path,
        "-i", metafile,
        "-map", "0:v:0", "-map", "1:a:0", "-map_metadata", "2",
        "-vf", "fps=%d,fade=t=in:st=0:d=%s,fade=t=out:st=%.3f:d=%s,format=yuv420p"
               % (FPS, FADE_IN_S, fade_out, FADE_OUT_S),
        "-af", "afade=t=out:st=%.3f:d=%s" % (fade_out, FADE_OUT_S),
        "-c:v", "libx264", "-preset", X264_PRESET, "-crf", str(CRF),
        "-profile:v", "high", "-level", "4.2",
        "-x264-params", "keyint=%d:min-keyint=%d:scenecut=0" % (FPS * 2, FPS * 2),
        "-color_primaries", "bt709", "-color_trc", "bt709", "-colorspace", "bt709",
        "-c:a", "aac", "-b:a", AUDIO_BITRATE, "-ar", "48000", "-ac", "2",
        "-movflags", "+faststart", "-shortest", "-threads", "4", mp4)

    thumb = os.path.join(OUTDIR, stem + "-thumb.png")
    await run("ffmpeg", "-y", "-v", "error", "-i", plans[0]["files"][-1],
              "-vf", "scale=1280:720:flags=lanczos", thumb)

    size = os.path.getsize(mp4) / 1048576
    if not keep:
        shutil.rmtree(work, ignore_errors=True)
    log("wrote %s  (%.1f MB, %s, %d chapters) in %s"
        % (os.path.basename(mp4), size, fmt_clock(total), len(chapters),
           fmt_clock(time.time() - started)))
    return {"file": mp4, "seconds": total, "megabytes": size,
            "chapters": len(chapters), "video_len": video_len}


async def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("episodes", nargs="*", help="episode numbers, default all")
    ap.add_argument("--jobs", type=int, default=4, help="browsers at once")
    ap.add_argument("--keep-frames", action="store_true",
                    help="leave the captured png frames on disk")
    ap.add_argument("--notes", action="store_true",
                    help="only rebuild video/UPLOAD.md from what is already there")
    args = ap.parse_args()

    if args.notes:
        print("wrote %s" % write_upload_notes())
        return 0

    if not os.path.exists(DECK):
        return _fail("build the deck first: python3 scripts/build-deck.py")
    for tool in ("google-chrome", "ffmpeg", "ffprobe"):
        if not shutil.which(tool):
            return _fail("%s is not on PATH" % tool)

    with open(os.path.join(AUDIO, "manifest.json"), encoding="utf-8") as fh:
        manifest = json.load(fh)["episodes"]

    episodes = []
    for path in sorted(glob.glob(os.path.join(ROOT, "content", "ep*.json"))):
        with open(path, encoding="utf-8") as fh:
            ep = json.load(fh)
        key = "%02d" % ep["number"]
        if args.episodes and key not in ["%02d" % int(e) for e in args.episodes]:
            continue
        if key not in manifest:
            return _fail("no narration for episode %s, run gen-audio.py first" % key)
        if len(manifest[key]["beats"]) != len(ep["beats"]):
            return _fail("episode %s has %d slides but %d clips, rerun gen-audio.py"
                         % (key, len(ep["beats"]), len(manifest[key]["beats"])))
        episodes.append(ep)
    if not episodes:
        return _fail("no episodes matched")

    scratch = os.environ.get("PC_RENDER_SCRATCH") or os.path.join(ROOT, ".render")
    os.makedirs(scratch, exist_ok=True)
    print("rendering %d episode(s) at %dx%d %dfps, %d at a time"
          % (len(episodes), WIDTH, HEIGHT, FPS, args.jobs), flush=True)

    gate = asyncio.Semaphore(args.jobs)
    started = time.time()

    async def one(i, ep):
        async with gate:
            return await render_episode(ep, manifest["%02d" % ep["number"]],
                                        scratch, 9400 + i, args.keep_frames)

    results = await asyncio.gather(*[one(i, ep) for i, ep in enumerate(episodes)],
                                   return_exceptions=True)

    failed = 0
    total_s = total_mb = 0.0
    for ep, r in zip(episodes, results):
        if isinstance(r, Exception):
            failed += 1
            print("  ep%02d FAILED: %s" % (ep["number"], r), file=sys.stderr)
        else:
            total_s += r["seconds"]
            total_mb += r["megabytes"]
    if not args.keep_frames:
        shutil.rmtree(scratch, ignore_errors=True)

    if failed < len(episodes):
        write_upload_notes()
    print("\n%d/%d episodes rendered to %s" %
          (len(episodes) - failed, len(episodes), OUTDIR))
    print("  %s of video, %.0f MB, built in %s"
          % (fmt_clock(total_s), total_mb, fmt_clock(time.time() - started)))
    return 1 if failed else 0


def _fail(msg: str) -> int:
    print("error: %s" % msg, file=sys.stderr)
    return 2


if __name__ == "__main__":
    sys.exit(asyncio.run(main()))
