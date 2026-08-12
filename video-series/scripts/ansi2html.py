#!/usr/bin/env python3
"""Convert captured ANSI terminal frames into HTML spans.

Reads video-series/tui/*.ansi, writes video-series/tui/frames.json which maps
frame name -> HTML string. The deck inlines that JSON so the published page has
no external dependencies.
"""
import html
import json
import os
import re
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
TUI = os.path.join(HERE, "..", "tui")

# xterm 256-colour palette -----------------------------------------------------
BASE16 = [
    (0x00, 0x00, 0x00), (0xCD, 0x31, 0x31), (0x0D, 0xBC, 0x79), (0xE5, 0xE5, 0x10),
    (0x24, 0x72, 0xC8), (0xBC, 0x3F, 0xBC), (0x11, 0xA8, 0xCD), (0xE5, 0xE5, 0xE5),
    (0x66, 0x66, 0x66), (0xF1, 0x4C, 0x4C), (0x23, 0xD1, 0x8B), (0xF5, 0xF5, 0x43),
    (0x3B, 0x8E, 0xEA), (0xD6, 0x70, 0xD6), (0x29, 0xB8, 0xDB), (0xFF, 0xFF, 0xFF),
]


def xterm256(n):
    if n < 16:
        return BASE16[n]
    if n < 232:
        n -= 16
        levels = [0, 95, 135, 175, 215, 255]
        return (levels[n // 36], levels[(n // 6) % 6], levels[n % 6])
    v = 8 + (n - 232) * 10
    return (v, v, v)


def rgb(t):
    return "#%02x%02x%02x" % t


DEFAULT_FG = (0xD4, 0xD4, 0xD4)
DEFAULT_BG = None  # transparent, deck paints the terminal background


class State:
    __slots__ = ("fg", "bg", "bold", "dim", "italic", "underline", "reverse")

    def __init__(self):
        self.reset()

    def reset(self):
        self.fg = None
        self.bg = None
        self.bold = False
        self.dim = False
        self.italic = False
        self.underline = False
        self.reverse = False

    def key(self):
        return (self.fg, self.bg, self.bold, self.dim,
                self.italic, self.underline, self.reverse)

    def style(self):
        fg = self.fg if self.fg is not None else DEFAULT_FG
        bg = self.bg if self.bg is not None else DEFAULT_BG
        if self.reverse:
            fg, bg = (bg if bg is not None else (0x0B, 0x0E, 0x14)), fg
        css = []
        if self.bold:
            # Bold in a terminal usually means "brighter", which the palette
            # already encodes. Keep the weight but do not double up.
            css.append("font-weight:600")
        if self.dim:
            css.append("opacity:.65")
        if self.italic:
            css.append("font-style:italic")
        if self.underline:
            css.append("text-decoration:underline")
        css.append("color:%s" % rgb(fg))
        if bg is not None:
            css.append("background:%s" % rgb(bg))
        return ";".join(css)


def apply_sgr(state, params):
    if not params:
        params = [0]
    i = 0
    while i < len(params):
        p = params[i]
        if p == 0:
            state.reset()
        elif p == 1:
            state.bold = True
        elif p == 2:
            state.dim = True
        elif p == 3:
            state.italic = True
        elif p == 4:
            state.underline = True
        elif p == 7:
            state.reverse = True
        elif p == 22:
            state.bold = state.dim = False
        elif p == 23:
            state.italic = False
        elif p == 24:
            state.underline = False
        elif p == 27:
            state.reverse = False
        elif 30 <= p <= 37:
            state.fg = BASE16[p - 30]
        elif p == 39:
            state.fg = None
        elif 40 <= p <= 47:
            state.bg = BASE16[p - 40]
        elif p == 49:
            state.bg = None
        elif 90 <= p <= 97:
            state.fg = BASE16[p - 90 + 8]
        elif 100 <= p <= 107:
            state.bg = BASE16[p - 100 + 8]
        elif p in (38, 48):
            target = "fg" if p == 38 else "bg"
            if i + 1 < len(params) and params[i + 1] == 5:
                setattr(state, target, xterm256(params[i + 2]))
                i += 2
            elif i + 1 < len(params) and params[i + 1] == 2:
                setattr(state, target,
                        (params[i + 2], params[i + 3], params[i + 4]))
                i += 4
        i += 1


# Strip OSC (title-setting) and any non-SGR CSI sequence, keep SGR.
OSC = re.compile(r"\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)")
CSI = re.compile(r"\x1b\[([0-9;:?]*)([a-zA-Z])")
OTHER_ESC = re.compile(r"\x1b[()][0-9A-Za-z]|\x1b[=>]")


def convert(text):
    text = OSC.sub("", text)
    text = OTHER_ESC.sub("", text)
    text = text.replace("\r\n", "\n").replace("\r", "")
    text = text.rstrip("\n")

    state = State()
    out = []
    open_span = False
    pos = 0

    def close():
        nonlocal open_span
        if open_span:
            out.append("</span>")
            open_span = False

    for m in CSI.finditer(text):
        chunk = text[pos:m.start()]
        if chunk:
            out.append(html.escape(chunk))
        pos = m.end()
        if m.group(2) != "m":
            continue  # cursor movement etc, nothing to render
        raw = m.group(1)
        if "?" in raw:
            continue
        params = []
        for part in raw.replace(":", ";").split(";"):
            params.append(int(part) if part.isdigit() else 0)
        close()
        apply_sgr(state, params)
        if state.key() != (None, None, False, False, False, False, False):
            out.append('<span style="%s">' % state.style())
            open_span = True

    tail = text[pos:]
    if tail:
        out.append(html.escape(tail))
    close()
    return "".join(out)


def main():
    frames = {}
    names = sorted(f for f in os.listdir(TUI) if f.endswith(".ansi"))
    if not names:
        print("no .ansi frames found in %s" % TUI, file=sys.stderr)
        return 1
    for name in names:
        key = name[:-5]
        with open(os.path.join(TUI, name), encoding="utf-8", errors="replace") as fh:
            raw = fh.read()
        frames[key] = convert(raw)
        lines = frames[key].count("\n") + 1
        print("  %-26s %3d lines  %6d bytes html" % (key, lines, len(frames[key])))

    dest = os.path.join(TUI, "frames.json")
    with open(dest, "w", encoding="utf-8") as fh:
        json.dump(frames, fh, ensure_ascii=False)
    print("\nwrote %s (%d frames, %.1f KB)" %
          (dest, len(frames), os.path.getsize(dest) / 1024))
    return 0


if __name__ == "__main__":
    sys.exit(main())
