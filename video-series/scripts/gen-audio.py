#!/usr/bin/env python3
"""Generate voiceover audio for every beat with edge-tts.

edge-tts is free and needs no account. It uses the neural voices behind
Microsoft Edge's read-aloud feature.

Output goes to video-series/audio/:
  epNN-BB.mp3      one file per beat
  epNN.mp3         the whole episode, with short gaps between beats
  manifest.json    durations and hashes, consumed by build-deck.py

Beats are keyed by a hash of (voice, rate, pitch, text), so re-running after a
script edit only regenerates what changed.
"""
import glob
import hashlib
import json
import os
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.normpath(os.path.join(HERE, ".."))
CONTENT = os.path.join(ROOT, "content")
AUDIO = os.path.join(ROOT, "audio")

VOICE = os.environ.get("PC_TTS_VOICE", "en-US-BrianMultilingualNeural")
RATE = os.environ.get("PC_TTS_RATE", "-3%")
PITCH = os.environ.get("PC_TTS_PITCH", "+0Hz")
GAP = 0.45  # seconds of silence between beats in the joined episode file


def run(cmd, **kw):
    return subprocess.run(cmd, check=True, capture_output=True, text=True, **kw)


def duration(path):
    out = run(["ffprobe", "-v", "error", "-show_entries", "format=duration",
               "-of", "csv=p=0", path]).stdout.strip()
    return round(float(out), 3)


def beat_hash(text):
    h = hashlib.sha256()
    h.update("\x00".join([VOICE, RATE, PITCH, text]).encode("utf-8"))
    return h.hexdigest()[:16]


def synth(text, dest):
    # --rate=-3% must use the equals form; argparse reads a bare "-3%" as a flag.
    run(["edge-tts", "--voice", VOICE, "--rate=" + RATE, "--pitch=" + PITCH,
         "--text", text, "--write-media", dest])


def join_episode(files, dest):
    """Concatenate beat files with a short silence between them."""
    listfile = dest + ".txt"
    silence = os.path.join(AUDIO, "_gap.mp3")
    if not os.path.exists(silence):
        run(["ffmpeg", "-y", "-f", "lavfi", "-i",
             "anullsrc=r=24000:cl=mono", "-t", str(GAP),
             "-q:a", "9", silence])
    with open(listfile, "w", encoding="utf-8") as fh:
        for i, f in enumerate(files):
            if i:
                fh.write("file '%s'\n" % os.path.basename(silence))
            fh.write("file '%s'\n" % os.path.basename(f))
    run(["ffmpeg", "-y", "-f", "concat", "-safe", "0", "-i", listfile,
         "-c", "copy", dest], cwd=AUDIO)
    os.remove(listfile)


def main():
    os.makedirs(AUDIO, exist_ok=True)
    manifest_path = os.path.join(AUDIO, "manifest.json")
    old = {}
    if os.path.exists(manifest_path):
        with open(manifest_path, encoding="utf-8") as fh:
            old = json.load(fh).get("episodes", {})

    only = sys.argv[1:] or None
    manifest = {"voice": VOICE, "rate": RATE, "pitch": PITCH, "episodes": {}}
    made = skipped = 0

    for path in sorted(glob.glob(os.path.join(CONTENT, "ep*.json"))):
        with open(path, encoding="utf-8") as fh:
            ep = json.load(fh)
        num = "%02d" % ep["number"]
        if only and num not in only and str(ep["number"]) not in only:
            if num in old:
                manifest["episodes"][num] = old[num]
            continue

        prev = {b["hash"]: b for b in old.get(num, {}).get("beats", [])}
        beats = []
        files = []
        print("episode %s  %s" % (num, ep["title"]))

        for i, beat in enumerate(ep["beats"]):
            text = beat["narration"].strip()
            h = beat_hash(text)
            fname = "ep%s-%02d.mp3" % (num, i)
            dest = os.path.join(AUDIO, fname)

            if h in prev and os.path.exists(dest) and prev[h]["file"] == fname:
                dur = prev[h]["duration"]
                skipped += 1
            else:
                synth(text, dest)
                dur = duration(dest)
                made += 1
                print("    [%02d] %5.1fs  %s" % (i, dur, text[:58].replace("\n", " ")))

            beats.append({"index": i, "file": fname, "duration": dur,
                          "hash": h, "words": len(text.split())})
            files.append(dest)

        total = sum(b["duration"] for b in beats) + GAP * (len(beats) - 1)
        joined = "ep%s.mp3" % num
        join_episode(files, os.path.join(AUDIO, joined))
        manifest["episodes"][num] = {
            "number": ep["number"], "title": ep["title"], "slug": ep["slug"],
            "file": joined, "duration": round(total, 2), "beats": beats,
        }
        print("    total %s  (%d beats, %d words)\n" %
              (fmt(total), len(beats), sum(b["words"] for b in beats)))

    for num in sorted(old):
        manifest["episodes"].setdefault(num, old[num])

    with open(manifest_path, "w", encoding="utf-8") as fh:
        json.dump(manifest, fh, indent=1)

    grand = sum(e["duration"] for e in manifest["episodes"].values())
    print("generated %d clips, reused %d" % (made, skipped))
    print("series runtime: %s across %d episodes" %
          (fmt(grand), len(manifest["episodes"])))
    return 0


def fmt(sec):
    return "%d:%02d" % (int(sec) // 60, int(sec) % 60)


if __name__ == "__main__":
    sys.exit(main())
