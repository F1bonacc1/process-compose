#!/usr/bin/env python3
"""Assemble the standalone presentation.

Reads content/ep*.json, tui/frames.json, audio/manifest.json and the deck
sources, and writes video-series/process-compose-course.html with the CSS, the
JS, the terminal captures and the scripts all inlined. No network requests are
made by the finished page.

Audio is referenced relatively as audio/*.mp3. Pass --embed to additionally
write a single-file version with the audio inlined as data URIs.
"""
import base64
import glob
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.normpath(os.path.join(HERE, ".."))
CONTENT = os.path.join(ROOT, "content")
DECK = os.path.join(HERE, "deck")

SHELL = """<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Process Compose · video series</title>
<style>
__CSS__
</style>
</head>
<body>

<div id="app">
  <div id="stagewrap"><div class="stage" id="stage"></div></div>

  <div id="bar">
    <div class="group">
      <button id="prev" title="Left arrow">←</button>
      <button id="play" title="P">▶ Play</button>
      <button id="next" title="Space or right arrow">→</button>
    </div>

    <div id="meta">
      <div id="metatop">
        <b id="epname"></b>
        <span>
          <kbd class="hint">space</kbd> next
          <kbd class="hint">P</kbd> play
          <kbd class="hint">E</kbd> episodes
          <kbd class="hint">S</kbd> script
          <kbd class="hint">F</kbd> full screen
          <kbd class="hint">[ ]</kbd> episode
        </span>
        <span id="eptime"></span>
      </div>
      <div id="track"></div>
    </div>

    <div class="group">
      <button id="eps" title="E">Episodes</button>
      <button id="scr" title="S">Script</button>
      <button id="full" title="F">⬜</button>
    </div>
  </div>
</div>

<div class="drawer" id="epdrawer">
  <h2>Process Compose · video series</h2>
  <div class="hint">__SERIESMETA__ · press E or Escape to close</div>
  <div class="eplist" id="eplist"></div>
</div>

<div class="drawer" id="scriptdrawer">
  <div id="script-outer" style="max-width:900px;margin:0 auto">
    <h2 id="scripthead"></h2>
    <div class="hint" id="scriptmeta"></div>
    <div id="script"></div>
  </div>
</div>

<script>
const COURSE = __COURSE__;
const FRAMES = __FRAMES__;
const MANIFEST = __MANIFEST__;
</script>
<script>
__JS__
</script>
</body>
</html>
"""


def read(path, default=None):
    if not os.path.exists(path):
        if default is None:
            raise SystemExit("missing required file: %s" % path)
        return default
    with open(path, encoding="utf-8") as fh:
        return fh.read()


def fmt(sec):
    sec = int(round(sec))
    return "%d:%02d" % (sec // 60, sec % 60)


def main():
    episodes = []
    for path in sorted(glob.glob(os.path.join(CONTENT, "ep*.json"))):
        with open(path, encoding="utf-8") as fh:
            episodes.append(json.load(fh))
    if not episodes:
        raise SystemExit("no episodes found in %s" % CONTENT)

    frames = json.loads(read(os.path.join(ROOT, "tui", "frames.json")))
    manifest = json.loads(read(os.path.join(ROOT, "audio", "manifest.json"),
                               '{"episodes":{}}'))

    course = {"episodes": episodes}
    beats = sum(len(e["beats"]) for e in episodes)
    words = sum(len(b["narration"].split()) for e in episodes for b in e["beats"])
    runtime = sum(v["duration"] for v in manifest.get("episodes", {}).values())

    meta = "%d episodes · %d beats · %d words · %s narration · %d real terminal captures" % (
        len(episodes), beats, words, fmt(runtime), len(frames))

    html = (SHELL
            .replace("__CSS__", read(os.path.join(DECK, "deck.css")))
            .replace("__JS__", read(os.path.join(DECK, "deck.js")))
            .replace("__SERIESMETA__", meta)
            .replace("__COURSE__", json.dumps(course, ensure_ascii=False))
            .replace("__FRAMES__", json.dumps(frames, ensure_ascii=False))
            .replace("__MANIFEST__", json.dumps(manifest, ensure_ascii=False)))

    dest = os.path.join(ROOT, "process-compose-course.html")
    with open(dest, "w", encoding="utf-8") as fh:
        fh.write(html)
    print("wrote %s  (%.1f KB)" % (dest, os.path.getsize(dest) / 1024))
    print("  %s" % meta)

    if "--embed" in sys.argv:
        embedded = embed_audio(html, manifest)
        dest2 = os.path.join(ROOT, "process-compose-course.single-file.html")
        with open(dest2, "w", encoding="utf-8") as fh:
            fh.write(embedded)
        print("wrote %s  (%.1f MB, audio inlined)" %
              (dest2, os.path.getsize(dest2) / 1048576))
    return 0


def embed_audio(html, manifest):
    """Replace the audio/*.mp3 references with data: URIs."""
    table = {}
    audio_dir = os.path.join(ROOT, "audio")
    for ep in manifest.get("episodes", {}).values():
        for b in ep["beats"]:
            p = os.path.join(audio_dir, b["file"])
            if os.path.exists(p):
                with open(p, "rb") as fh:
                    table[b["file"]] = "data:audio/mpeg;base64," + \
                        base64.b64encode(fh.read()).decode("ascii")
    inject = ("\nconst AUDIO_DATA = " + json.dumps(table) + ";\n")
    # deck.js builds paths as "audio/" + file; redirect that through the table.
    html = html.replace(
        'return "audio/" + m.beats[b].file;',
        'const f = m.beats[b].file;\n'
        '    return (typeof AUDIO_DATA !== "undefined" && AUDIO_DATA[f]) '
        '|| ("audio/" + f);')
    html = html.replace(
        'if (!audio.src.endsWith(src)) audio.src = src;',
        'if (audio.src !== src) audio.src = src;')
    return html.replace("const MANIFEST = ", inject + "const MANIFEST = ")


if __name__ == "__main__":
    sys.exit(main())
