#!/usr/bin/env python3
"""Write SCRIPTS.md: every episode's narration as a readable shooting script.

Use this to record the voiceover yourself, hand it to a narrator, or paste
chapters into a YouTube description.
"""
import glob
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.normpath(os.path.join(HERE, ".."))
sys.path.insert(0, HERE)
from timing import GAP_S  # noqa: E402


def fmt(sec):
    sec = int(round(sec))
    return "%d:%02d" % (sec // 60, sec % 60)


def cue(v):
    t = v["type"]
    if t in ("tui",):
        return "TUI capture: %s" % v["frame"]
    if t == "cmdframe":
        return "terminal: %s  ->  %s" % (v["command"].split("\n")[0], v["frame"])
    if t == "cmd":
        return "terminal: %s" % v["command"].split("\n")[0]
    if t == "code":
        return "code: %s" % v.get("filename", v.get("lang", ""))
    if t == "diagram":
        return "diagram: %s" % v["name"]
    if t in ("points", "keys", "table"):
        return "%s: %s" % (t, v.get("heading", ""))
    if t == "callout":
        return "callout: %s" % v.get("heading", "")
    if t == "quote":
        return "quote: %s" % v.get("source", "")
    if t == "compare":
        return "compare: %s | %s" % (v["before"]["title"], v["after"]["title"])
    if t == "conditions":
        return "conditions: %s" % ", ".join(c["name"] for c in v["items"])
    if t == "title":
        return "title card"
    if t == "bigtext":
        return "end card: %s" % v["text"]
    return t


def main():
    manifest = {}
    mp = os.path.join(ROOT, "audio", "manifest.json")
    if os.path.exists(mp):
        with open(mp, encoding="utf-8") as fh:
            manifest = json.load(fh).get("episodes", {})

    out = ["# Process Compose video series", "",
           "Voiceover scripts. One section per episode, one paragraph per beat.",
           "The cue line says what should be on screen while the paragraph is read.",
           ""]

    total_words = total_time = 0
    index = []

    for path in sorted(glob.glob(os.path.join(ROOT, "content", "ep*.json"))):
        with open(path, encoding="utf-8") as fh:
            ep = json.load(fh)
        key = "%02d" % ep["number"]
        meta = manifest.get(key)
        words = sum(len(b["narration"].split()) for b in ep["beats"])
        total_words += words
        if meta:
            total_time += meta["duration"]
        index.append((ep["number"], ep["title"], words,
                      fmt(meta["duration"]) if meta else "-"))

        out.append("---")
        out.append("")
        out.append("## Episode %d. %s" % (ep["number"], ep["title"]))
        out.append("")
        out.append("*%s*" % ep["subtitle"])
        out.append("")
        out.append("`%d beats · %d words%s`" %
                   (len(ep["beats"]), words,
                    " · %s" % fmt(meta["duration"]) if meta else ""))
        out.append("")

        clock = 0.0
        for i, b in enumerate(ep["beats"]):
            stamp = fmt(clock) if meta else ""
            dur = meta["beats"][i]["duration"] if meta else 0
            out.append("**%02d**%s  `%s`" %
                       (i + 1, "  ·  " + stamp if stamp else "", cue(b["visual"])))
            out.append("")
            out.append(b["narration"])
            out.append("")
            clock += dur + GAP_S   # matches the rendered video, see timing.py

    head = ["", "## Index", "",
            "| # | episode | words | runtime |",
            "|---|---------|-------|---------|"]
    for n, t, w, d in index:
        head.append("| %d | %s | %d | %s |" % (n, t, w, d))
    head += ["", "**Total: %d words, %s.**" % (total_words, fmt(total_time)), ""]

    out = out[:4] + head + out[4:]

    dest = os.path.join(ROOT, "SCRIPTS.md")
    with open(dest, "w", encoding="utf-8") as fh:
        fh.write("\n".join(out))
    print("wrote %s (%d words, %s)" % (dest, total_words, fmt(total_time)))


if __name__ == "__main__":
    main()
