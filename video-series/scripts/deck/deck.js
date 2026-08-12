/* Process Compose video series deck.
   COURSE, FRAMES and MANIFEST are injected by build-deck.py before this runs. */

(function () {
  "use strict";

  const $ = (s, r) => (r || document).querySelector(s);
  const esc = (s) =>
    String(s).replace(/[&<>"']/g, (c) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));

  /* ------------------------------------------------------ syntax colour -- */

  // Highlight the interpolation forms that can appear inside a quoted string.
  function inner(text) {
    let out = "";
    const re = /(\{\{[^}]*\}\})|(\$\$?\{[^}]*\}|\$\$?[A-Za-z_][A-Za-z0-9_]*)/g;
    let last = 0, m;
    while ((m = re.exec(text))) {
      out += esc(text.slice(last, m.index));
      out += '<span class="v">' + esc(m[0]) + "</span>";
      last = re.lastIndex;
    }
    return out + esc(text.slice(last));
  }

  function tokenize(line, rules) {
    const re = new RegExp(rules.map((r) => "(" + r.re + ")").join("|"), "g");
    let out = "", last = 0, m;
    while ((m = re.exec(line))) {
      if (m[0] === "") { re.lastIndex++; continue; }
      out += esc(line.slice(last, m.index));
      let cls = null;
      for (let i = 0; i < rules.length; i++) {
        if (m[i + 1] !== undefined) { cls = rules[i]; break; }
      }
      out += cls && cls.cls
        ? '<span class="' + cls.cls + '">' +
          (cls.inner ? inner(m[0]) : esc(m[0])) + "</span>"
        : esc(m[0]);
      last = re.lastIndex;
    }
    return out + esc(line.slice(last));
  }

  const YAML_RULES = [
    { re: "#.*$", cls: "c" },
    { re: '"(?:[^"\\\\]|\\\\.)*"', cls: "s", inner: true },
    { re: "'(?:[^'\\\\]|\\\\.)*'", cls: "s", inner: true },
    { re: "\\{\\{[^}]*\\}\\}", cls: "v" },
    { re: "\\$\\$?\\{[^}]*\\}", cls: "v" },
    { re: "^\\s*(?:-\\s*)?[\\w.\\-/]+(?=\\s*:)", cls: "k" },
    { re: "\\b(?:true|false|null)\\b", cls: "n" },
    { re: "\\b\\d+(?:\\.\\d+)?\\b", cls: "n" },
  ];

  const SHELL_RULES = [
    { re: "#.*$", cls: "c" },
    { re: '"(?:[^"\\\\]|\\\\.)*"', cls: "s", inner: true },
    { re: "'(?:[^'\\\\]|\\\\.)*'", cls: "s", inner: true },
    { re: "\\$\\$?\\{[^}]*\\}", cls: "v" },
    { re: "\\$[A-Za-z_][A-Za-z0-9_]*", cls: "v" },
    { re: "(?:^|(?<=[|&;]\\s))\\s*[a-z][\\w.-]*", cls: "cm" },
    { re: "\\s--?[A-Za-z][\\w-]*", cls: "f" },
  ];

  const JSON_RULES = [
    { re: '"(?:[^"\\\\]|\\\\.)*"(?=\\s*:)', cls: "k" },
    { re: '"(?:[^"\\\\]|\\\\.)*"', cls: "s" },
    { re: "\\b(?:true|false|null)\\b", cls: "n" },
    { re: "\\b\\d+(?:\\.\\d+)?\\b", cls: "n" },
  ];

  const RULES = { yaml: YAML_RULES, shell: SHELL_RULES, json: JSON_RULES };

  function highlight(code, lang, hl) {
    const rules = RULES[lang] || YAML_RULES;
    const marks = new Set(hl || []);
    // .ln is display:block, so the lines must be joined with nothing at all.
    // Joining with "\n" would render a blank line between every line.
    return code.split("\n").map((line, i) =>
      '<span class="ln' + (marks.has(i + 1) ? " hl" : "") +
      '" style="animation-delay:' + (i * 26) + 'ms">' +
      (line === "" ? "&nbsp;" : tokenize(line, rules)) + "</span>"
    ).join("");
  }

  // Pick a font size in cqw so the block fits the stage without scrolling.
  // Mono advance width is ~0.6em; line box is ~1.55em.
  function fitCode(code, availW, availH, maxFs) {
    const lines = code.split("\n");
    const longest = lines.reduce((a, l) => Math.max(a, l.length), 1);
    const byWidth = availW / (0.6 * longest);
    const byHeight = availH / (1.55 * lines.length);
    const fs = Math.min(byWidth, byHeight, maxFs);
    return Math.max(0.68, Math.round(fs * 100) / 100);
  }

  /* ---------------------------------------------------------- diagrams -- */

  /* NOTE: .d-box, .d-arrow, .d-label, .d-title and .d-small declare fill and
     stroke in the stylesheet. A CSS declaration always beats an SVG
     presentation attribute, so per-element colours have to be set with an
     inline style, never with fill="..." or stroke="...". */

  const wrap = (inner) =>
    '<div class="diagram"><svg viewBox="0 0 200 100" ' +
    'preserveAspectRatio="xMidYMid meet">' + inner + "</svg></div>";

  const st = (s) => (s ? ' style="' + s + '"' : "");

  // rounded panel
  const box = (x, y, w, h, style, cls) =>
    '<rect class="' + (cls || "d-box") + '" x="' + x + '" y="' + y +
    '" width="' + w + '" height="' + h + '" rx="1.6"' + st(style) + "/>";

  // text; style carries the colour, e.g. "fill:#8aadf4;font-weight:700"
  const label = (x, y, t, cls, anchor, style) =>
    '<text class="' + (cls || "d-label") + '" x="' + x + '" y="' + y +
    '" text-anchor="' + (anchor || "middle") + '"' + st(style) + ">" +
    esc(t) + "</text>";

  const path = (d, style, cls) =>
    '<path class="' + (cls || "d-arrow") + '" d="' + d + '"' + st(style) + "/>";

  // solid arrowhead pointing right at (x, y)
  const tip = (x, y, colour) =>
    '<path d="M' + (x - 3) + " " + (y - 1.6) + " l3 1.6 l-3 1.6 z\"" +
    st("fill:" + (colour || "#5b6078") + ";stroke:none") + "/>";

  const C = {
    red: "#ed8796", peach: "#f5a97f", yellow: "#eed49f", green: "#a6da95",
    blue: "#8aadf4", teal: "#8bd5ca", mauve: "#c6a0f6", grey: "#8087a2",
    line: "#5b6078",
  };

  const DIAGRAMS = {

    tabs() {
      let s = label(100, 9, "One stack, five terminals, no shared state", "d-title");
      const tabs = [
        { n: "postgres", ok: true }, { n: "migrations", ok: true },
        { n: "api", ok: true }, { n: "worker", ok: false }, { n: "frontend", ok: true },
      ];
      tabs.forEach((t, i) => {
        const x = 8 + i * 37, y = 20 + i * 2.5;
        s += box(x, y, 33, 50, t.ok ? "" : "stroke:" + C.red + ";stroke-width:1;fill:#2b1b23");
        s += '<rect x="' + x + '" y="' + y + '" width="33" height="6" rx="1.6" fill="' +
             (t.ok ? "#363a4f" : "#5c2f3d") + '"/>';
        s += '<circle cx="' + (x + 3.5) + '" cy="' + (y + 3) + '" r="1" fill="#ed8796"/>';
        s += '<circle cx="' + (x + 7) + '" cy="' + (y + 3) + '" r="1" fill="#eed49f"/>';
        s += '<circle cx="' + (x + 10.5) + '" cy="' + (y + 3) + '" r="1" fill="#a6da95"/>';
        s += label(x + 16.5, y + 13, t.n, "d-label",
                   "middle", "fill:" + (t.ok ? "#cad3f5" : C.red) + ";font-weight:700");
        for (let l = 0; l < 5; l++) {
          const w = 8 + ((i * 7 + l * 11) % 16);
          s += '<rect x="' + (x + 3.5) + '" y="' + (y + 18 + l * 5) + '" width="' + w +
               '" height="1.7" rx=".85" fill="#494d64" opacity="' + (t.ok ? .85 : .25) + '"/>';
        }
        if (!t.ok) {
          s += '<g class="pulse">' +
               label(x + 16.5, y + 44, "exited 1", "d-label", "middle",
                     "fill:" + C.red + ";font-weight:700") +
               label(x + 16.5, y + 48.5, "20 min ago", "d-small", "middle",
                     "fill:" + C.red) + "</g>";
        }
      });
      s += label(100, 94, "and the one that died is behind three other tabs", "d-small");
      return wrap(s);
    },

    tuilayout() {
      let s = "";
      const parts = [
        { y: 8,  h: 17, t: "project header", d: "version · project · running / total · RAM and CPU", c: C.blue },
        { y: 27, h: 30, t: "process table",  d: "PID  NAME  NAMESPACE  STATUS  AGE  HEALTH  MEM  CPU  RESTARTS  EXIT", c: C.green },
        { y: 59, h: 22, t: "log pane",       d: "output of whichever process is selected", c: C.yellow },
        { y: 83, h: 10, t: "shortcut bar",   d: "the keys that are live right now, in this context", c: C.mauve },
      ];
      parts.forEach((p) => {
        s += box(10, p.y, 180, p.h, "stroke:" + p.c + ";stroke-opacity:.6");
        s += label(15, p.y + 7, p.t, "d-label", "start", "fill:" + p.c + ";font-weight:700");
        s += label(15, p.y + 12.6, p.d, "d-small", "start");
      });
      s += label(185, 45, "Tab switches focus", "d-small", "end", "fill:" + C.teal);
      return wrap(s);
    },

    states() {
      let s = label(100, 8, "Process states", "d-title");
      const row = [
        { x: 4,   t: "Pending",     c: C.grey },
        { x: 43,  t: "Launching",   c: C.blue },
        { x: 82,  t: "Running",     c: C.green },
        { x: 121, t: "Terminating", c: C.yellow },
        { x: 160, t: "Completed",   c: C.teal },
      ];
      row.forEach((n, i) => {
        s += box(n.x, 36, 36, 13, "stroke:" + n.c);
        s += label(n.x + 18, 44, n.t, "d-label", "middle", "fill:" + n.c + ";font-weight:700");
        if (i < row.length - 1) {
          s += path("M" + (n.x + 36) + " 42.5 H" + (row[i + 1].x - 1), "", "d-arrow dash");
          s += tip(row[i + 1].x - 0.5, 42.5);
        }
      });
      // restart loop back to launching
      s += path("M100 36 C100 22, 61 22, 61 35", "stroke:" + C.peach, "d-arrow dash");
      s += '<path d="M59.4 32.2 l3 3.2 l-4.5 .9 z"' + st("fill:" + C.peach) + "/>";
      s += label(80, 19, "Restarting  ·  availability decides", "d-label",
                 "middle", "fill:" + C.peach);
      // error branch
      s += path("M100 49 V62 H139", "stroke:" + C.red);
      s += tip(139.5, 62, C.red);
      s += box(140, 56, 36, 13, "stroke:" + C.red);
      s += label(158, 64, "Error", "d-label", "middle", "fill:" + C.red + ";font-weight:700");
      // parked states
      s += box(4, 74, 36, 13, "stroke:#6e738d;stroke-dasharray:2 1.5");
      s += label(22, 82, "Disabled", "d-label", "middle", "fill:" + C.grey);
      s += box(46, 74, 36, 13, "stroke:#6e738d;stroke-dasharray:2 1.5");
      s += label(64, 82, "Scheduled", "d-label", "middle", "fill:" + C.grey);
      s += label(140, 82, "Skipped happens when a dependency never became ready",
                 "d-small", "middle");
      return wrap(s);
    },

    readygap() {
      let s = label(100, 10, "Started is not ready", "d-title");
      s += '<rect x="14" y="34" width="172" height="15" rx="2" fill="#1e2030" stroke="#363a4f" stroke-width=".5"/>';
      s += '<rect class="grow" x="14" y="34" width="62" height="15" rx="2" fill="#ed8796" fill-opacity=".22" style="transform-origin:14px 0"/>';
      s += '<rect class="grow" x="76" y="34" width="110" height="15" rx="2" fill="#a6da95" fill-opacity=".2" style="transform-origin:76px 0;animation-delay:.45s"/>';
      s += label(45, 43.5, "config · DB pool · bind port", "d-label");
      s += label(131, 43.5, "serving requests", "d-label", "middle", "fill:" + C.green);

      s += path("M14 34 V25", "stroke:" + C.blue);
      s += label(14, 22, "fork / exec", "d-label", "start", "fill:" + C.blue + ";font-weight:700");
      s += label(14, 58, "process_started fires here", "d-small", "start");

      s += path("M76 34 V25", "stroke:" + C.green);
      s += label(76, 22, "first 200 on /health", "d-label", "middle",
                 "fill:" + C.green + ";font-weight:700");
      // anchored start at 82 so it clears the process_started caption,
      // which runs from x=14 to about x=61
      s += label(82, 58, "process_healthy fires here", "d-small", "start",
                 "fill:" + C.green);

      s += path("M16 68 H74", "stroke:" + C.red, "d-arrow dash");
      s += tip(15.5, 68, C.red) + tip(74.5, 68, C.red);
      s += label(45, 76, "every connection in this window fails", "d-label",
                 "middle", "fill:" + C.red);
      s += label(100, 92,
        "The gap is real on anything with a warm-up. A probe is how you measure it.",
        "d-small");
      return wrap(s);
    },

    probeflow() {
      let s = label(100, 9, "What a failing readiness probe actually does", "d-title");
      const step = (x, t, sub, c) =>
        box(x, 24, 42, 20, "stroke:" + c) +
        label(x + 21, 32, t, "d-label", "middle", "fill:" + c + ";font-weight:700") +
        label(x + 21, 38.5, sub, "d-small");
      s += step(4, "probe fails", "N times in a row", C.red);
      s += step(54, "not ready", "HEALTH column flips", C.yellow);
      s += step(104, "process stopped", "an internal stop", C.peach);
      s += step(154, "availability", "decides what happens", C.blue);
      [46, 96, 146].forEach((x) => {
        s += path("M" + x + " 34 H" + (x + 8), "", "d-arrow dash");
        s += tip(x + 8.5, 34);
      });
      s += path("M164 44 V56 H122", "stroke:" + C.green);
      s += tip(121.5, 56, C.green);
      s += box(80, 50, 42, 13, "stroke:" + C.green);
      s += label(101, 58, "restart: always", "d-label", "middle",
                 "fill:" + C.green + ";font-weight:700");
      s += label(101, 68, "it comes back", "d-small", "middle", "fill:" + C.green);

      s += path("M176 44 V80 H122", "stroke:" + C.red);
      s += tip(121.5, 80, C.red);
      s += box(80, 74, 42, 13, "stroke:" + C.red);
      s += label(101, 82, "no policy", "d-label", "middle",
                 "fill:" + C.red + ";font-weight:700");
      s += label(101, 92, "it stays stopped", "d-small", "middle", "fill:" + C.red);
      return wrap(s);
    },

    signals() {
      let s = label(100, 9, "Who actually gets the signal", "d-title");
      s += box(70, 16, 60, 12, "stroke:" + C.peach);
      s += label(100, 23.5, "process-compose", "d-label", "middle",
                 "fill:" + C.peach + ";font-weight:700");
      s += path("M100 28 V36");
      s += box(70, 37, 60, 12, "stroke:" + C.blue);
      s += label(100, 44.5, 'sh -c "your command"', "d-label", "middle",
                 "fill:" + C.blue + ";font-weight:700");
      s += path("M100 49 V57");
      s += box(70, 58, 60, 12, "stroke:" + C.green);
      s += label(100, 65.5, "your program", "d-label", "middle",
                 "fill:" + C.green + ";font-weight:700");
      s += path("M100 70 V75 M100 75 H60 V80 M100 75 H140 V80");
      s += box(42, 80, 36, 11, "stroke:" + C.teal + ";stroke-dasharray:2 1.5");
      s += label(60, 87, "child", "d-small", "middle", "fill:" + C.teal);
      s += box(122, 80, 36, 11, "stroke:" + C.teal + ";stroke-dasharray:2 1.5");
      s += label(140, 87, "child", "d-small", "middle", "fill:" + C.teal);

      // group vs parent only. The dashed region must enclose the children too.
      s += '<rect x="38" y="33" width="124" height="61" rx="3" class="dash"' +
           st("fill:none;stroke:" + C.green + ";stroke-opacity:.55;stroke-dasharray:3 2") + "/>";
      s += label(196, 26, "parent_only: false", "d-label", "end",
                 "fill:" + C.green + ";font-weight:700");
      s += label(196, 31.5, "the whole group · default", "d-small", "end");
      s += '<rect x="68" y="35" width="64" height="16" rx="2.5"' +
           st("fill:none;stroke:" + C.yellow + ";stroke-opacity:.85") + "/>";
      s += label(4, 40, "parent_only: true", "d-label", "start",
                 "fill:" + C.yellow + ";font-weight:700");
      s += label(4, 45.5, "only the shell you launched", "d-small", "start");
      return wrap(s);
    },

    twophase() {
      let s = label(100, 9, "Two substitutions, two different moments", "d-title");
      s += box(6, 20, 86, 30, "stroke:" + C.blue);
      s += label(49, 28, "1.  the file is read", "d-label", "middle",
                 "fill:" + C.blue + ";font-weight:700");
      s += label(49, 35, "${VAR} is replaced now", "d-small");
      s += label(49, 40.5, "source: .env and the launching shell", "d-small");
      s += label(49, 46, "process environment does not exist yet", "d-small",
                 "middle", "fill:" + C.red);

      s += path("M92 35 H104", "", "d-arrow dash") + tip(104.5, 35);

      s += box(108, 20, 86, 30, "stroke:" + C.green);
      s += label(151, 28, "2.  the process starts", "d-label", "middle",
                 "fill:" + C.green + ";font-weight:700");
      s += label(151, 35, "$$VAR becomes $VAR, the shell expands it", "d-small");
      s += label(151, 40.5, "source: the shell, with the full environment", "d-small");
      s += label(151, 46, "command only. no other field runs a shell", "d-small",
                 "middle", "fill:" + C.yellow);

      s += box(6, 58, 188, 34, "stroke:#363a4f");
      s += label(12, 67, 'environment: ["ENV_TEST=from-process"]', "d-label",
                 "start", "fill:" + C.mauve);
      s += label(12, 77, 'command: "echo [${ENV_TEST}]"', "d-label", "start");
      s += label(124, 77, "prints  []", "d-label", "start",
                 "fill:" + C.red + ";font-weight:700");
      s += label(12, 87, 'command: "echo [$${ENV_TEST}]"', "d-label", "start");
      s += label(124, 87, "prints  [from-process]", "d-label", "start",
                 "fill:" + C.green + ";font-weight:700");
      return wrap(s);
    },

    clientserver() {
      let s = label(100, 8, "One binary, two roles", "d-title");
      s += box(56, 16, 88, 48, "stroke:" + C.peach);
      s += label(100, 24, "process-compose up", "d-label", "middle",
                 "fill:" + C.peach + ";font-weight:700");
      s += label(100, 30, "supervisor  +  HTTP server on :8080", "d-small");
      ["migrate", "api", "worker-0", "worker-1", "web"].forEach((n, i) => {
        const x = 60 + (i % 3) * 28, y = 36 + Math.floor(i / 3) * 12;
        s += '<rect x="' + x + '" y="' + y + '" width="25" height="9" rx="1.4" fill="#363a4f"/>';
        s += label(x + 12.5, y + 6, n, "d-small");
      });
      const clients = [
        { x: 2,   y: 20, t: "TUI",        d: "the default client" },
        { x: 2,   y: 44, t: "attach",     d: "reconnect any time" },
        { x: 152, y: 20, t: "CLI",        d: "process · project · ns" },
        { x: 152, y: 44, t: "curl · SDK", d: "REST and WebSocket" },
      ];
      clients.forEach((c) => {
        s += box(c.x, c.y, 46, 18, "stroke:" + C.blue);
        s += label(c.x + 23, c.y + 7, c.t, "d-label", "middle",
                   "fill:" + C.blue + ";font-weight:700");
        s += label(c.x + 23, c.y + 13.5, c.d, "d-small");
        const left = c.x < 100;
        const from = left ? c.x + 46 : c.x;
        const to = left ? 56 : 144;
        s += path("M" + from + " " + (c.y + 9) + " H" + to, "", "d-arrow dash");
      });
      s += label(100, 78,
        "The interface has no privileged access. It calls the same API you can.",
        "d-small");
      s += label(100, 90,
        "-U for a unix socket instead of a TCP port    ·    PC_API_TOKEN to require auth",
        "d-small", "middle", "fill:" + C.teal);
      return wrap(s);
    },
  };

  /* ------------------------------------------------------------ visuals -- */

  /* Drawn rather than typed. The information glyph U+2139 falls back to a
     plain letter "i" in most UI fonts, which reads as a typo. */
  const ICON = {
    warn:
      '<svg class="cicon" viewBox="0 0 24 24" aria-hidden="true">' +
      '<path d="M12 3.4 22.3 20.4H1.7Z" fill="none" stroke="currentColor" ' +
      'stroke-width="1.9" stroke-linejoin="round"/>' +
      '<path d="M12 10v4.4" stroke="currentColor" stroke-width="2.1" stroke-linecap="round"/>' +
      '<circle cx="12" cy="17.5" r="1.15" fill="currentColor"/></svg>',
    info:
      '<svg class="cicon" viewBox="0 0 24 24" aria-hidden="true">' +
      '<circle cx="12" cy="12" r="9.3" fill="none" stroke="currentColor" stroke-width="1.9"/>' +
      '<circle cx="12" cy="7.5" r="1.2" fill="currentColor"/>' +
      '<path d="M12 10.9v6.5" stroke="currentColor" stroke-width="2.1" stroke-linecap="round"/></svg>',
  };

  // Widest lead in a points/keys list, in cqw, so the second column lines up
  // instead of being run over by a long identifier like max_concurrent.
  function colWidth(items, key, fontCqw, padCqw) {
    const longest = items.reduce((a, i) => Math.max(a, String(i[key]).length), 1);
    return Math.round((longest * 0.6 * fontCqw + padCqw) * 100) / 100;
  }

  function codeBlock(v) {
    const fs = fitCode(v.code, 86, v.caption ? 35 : 38, 1.42);
    return '<div class="codewrap">' +
      '<div class="codebar"><span class="dot r"></span><span class="dot y"></span>' +
      '<span class="dot g"></span><span class="name">' + esc(v.filename || v.lang || "") +
      '</span></div>' +
      '<pre class="code" style="font-size:' + fs + 'cqw">' +
      highlight(v.code, v.lang, v.highlight) + "</pre></div>" +
      (v.caption ? '<div class="caption">' + esc(v.caption) + "</div>" : "");
  }

  function frameBlock(key, cli, caption) {
    const html = FRAMES[key];
    if (!html) return '<div class="caption">missing frame: ' + esc(key) + "</div>";
    return '<div class="term"><div class="codebar"><span class="dot r"></span>' +
      '<span class="dot y"></span><span class="dot g"></span>' +
      '<span class="name">' + esc(cli ? "shell" : "process-compose") + "</span></div>" +
      '<pre class="screen' + (cli ? " cli" : "") + '">' + html + "</pre></div>" +
      (caption ? '<div class="caption">' + esc(caption) + "</div>" : "");
  }

  function cmdBlock(command, output) {
    let h = '<div class="cmdline">';
    h += command.split("\n").map((l) =>
      l.trim().startsWith("#")
        ? '<span class="out">' + esc(l) + "</span>"
        : '<span class="pr">$</span>' + tokenize(l, SHELL_RULES)
    ).join("\n");
    if (!output) h += '<span class="cursor"></span>';
    h += "</div>";
    if (output) {
      h += '<div class="cmdline"><span class="out">' + esc(output) + "</span></div>";
    }
    return h;
  }

  function renderVisual(v) {
    switch (v.type) {
      case "title":
        return '<div class="kicker">' + esc(v.kicker) + "</div>" +
          "<h1>" + esc(v.title) + "</h1>" +
          '<div class="rule"></div>' +
          '<div class="sub">' + esc(v.subtitle) + "</div>";

      case "bigtext":
        return '<div class="big">' + esc(v.text) + "</div>" +
          (v.sub ? '<div class="sub" style="text-align:center">' + esc(v.sub) + "</div>" : "");

      case "points":
        return "<h2>" + esc(v.heading) + "</h2><div class=\"rule\"></div>" +
          '<div class="points' + (v.compact ? " compact" : "") +
          '" style="--leadw:' +
          colWidth(v.items, "lead", v.compact ? 1.3 : 1.5, 0.8) + 'cqw">' +
          v.items.map((it, i) =>
            '<div class="point" style="animation-delay:' + (120 + i * 70) + 'ms">' +
            '<div class="lead">' + esc(it.lead) + "</div>" +
            '<div class="body">' + esc(it.body) + "</div></div>").join("") +
          "</div>" + (v.note ? '<div class="note">' + esc(v.note) + "</div>" : "");

      case "keys":
        return "<h2>" + esc(v.heading) + '</h2><div class="rule"></div>' +
          '<div class="keys" data-cols="' + (v.columns || 1) +
          '" style="--keyw:' + colWidth(v.items, "key", 1.16, 1.9) + 'cqw">' +
          v.items.map((it, i) =>
            '<div class="keyrow" style="animation-delay:' + (110 + i * 45) + 'ms">' +
            "<kbd>" + esc(it.key) + "</kbd>" +
            '<div class="body">' + esc(it.body) + "</div></div>").join("") +
          "</div>" + (v.note ? '<div class="note">' + esc(v.note) + "</div>" : "");

      case "table":
        return "<h2>" + esc(v.heading) + '</h2><div class="rule"></div>' +
          '<table class="tbl"><thead><tr>' +
          v.columns.map((c) => "<th>" + esc(c) + "</th>").join("") +
          "</tr></thead><tbody>" +
          v.rows.map((r, i) =>
            '<tr style="animation-delay:' + (120 + i * 55) + 'ms">' +
            r.map((c) => "<td>" + esc(c) + "</td>").join("") + "</tr>").join("") +
          "</tbody></table>";

      case "conditions":
        return '<h2>depends_on <span class="p">·</span> condition</h2>' +
          '<div class="rule"></div>' +
          '<div class="conds">' + v.items.map((c, i) =>
          '<div class="cond' + (c.good ? " good" : "") +
          '" style="animation-delay:' + (100 + i * 140) + 'ms">' +
          '<div class="name">' + esc(c.name) + "</div>" +
          '<div class="meaning">' + esc(c.meaning) + "</div>" +
          '<div class="use">' + esc(c.use) + "</div>" +
          (c.warn ? '<div class="warn">' + esc(c.warn) + "</div>" : "") +
          "</div>").join("") + "</div>";

      case "callout":
        return '<div class="callout ' + (v.tone === "info" ? "info" : "") + '">' +
          '<div class="ch">' + (v.tone === "info" ? ICON.info : ICON.warn) +
          "<span>" + esc(v.heading) + "</span></div>" +
          '<div class="cb">' + esc(v.body) + "</div></div>";

      case "quote":
        return '<div class="quote">' + esc(v.text) + "</div>" +
          '<div class="qsrc">' + esc(v.source) + "</div>";

      case "compare": {
        const side = (o) =>
          '<div class="col"><h3>' + esc(o.title) + "</h3>" +
          (o.items ? "<ul>" + o.items.map((i) => "<li>" + esc(i) + "</li>").join("") + "</ul>" : "") +
          (o.code
            ? '<pre class="code" style="font-size:' +
              fitCode(o.code, 38, 26, 1.14) + 'cqw">' +
              highlight(o.code, "shell") + "</pre>"
            : "") +
          "</div>";
        return '<div class="compare">' + side(v.before) + side(v.after) + "</div>";
      }

      case "code":   return codeBlock(v);
      case "tui":    return frameBlock(v.frame, false, v.caption);
      case "cmd":    return cmdBlock(v.command, v.output) +
                            (v.caption ? '<div class="caption">' + esc(v.caption) + "</div>" : "");
      case "cmdframe":
        return cmdBlock(v.command, "") + frameBlock(v.frame, true, v.caption);
      case "diagram":
        return (DIAGRAMS[v.name] || (() => '<div class="caption">no diagram: ' + esc(v.name) + "</div>"))();
      default:
        return '<div class="caption">unknown visual: ' + esc(v.type) + "</div>";
    }
  }

  /* ------------------------------------------------------------- state -- */

  const EPS = COURSE.episodes;
  let ep = 0, beat = 0, playing = false;
  const audio = new Audio();
  audio.preload = "auto";

  const stage = $("#stage");
  const track = $("#track");

  function epMeta(i) {
    const key = String(EPS[i].number).padStart(2, "0");
    return (MANIFEST.episodes || {})[key] || null;
  }

  function beatAudio(i, b) {
    const m = epMeta(i);
    if (!m || !m.beats[b]) return null;
    return "audio/" + m.beats[b].file;
  }

  function render() {
    const e = EPS[ep], bt = e.beats[beat];
    const slideCls = "slide s-" + bt.visual.type;
    const html =
      '<div class="stamp"><span>Ep ' + String(e.number).padStart(2, "0") +
      "  ·  " + esc(e.title) + "</span>" +
      "<span>" + (beat + 1) + " / " + e.beats.length +
      ' <span class="flame">🔥</span></span></div>' +
      '<div class="' + slideCls + '">' + renderVisual(bt.visual) + "</div>";

    stage.innerHTML = html;
    // next frame so the transition actually runs
    requestAnimationFrame(() => {
      const el = $(".slide", stage);
      if (el) el.classList.add("on");
    });

    // progress segments
    track.innerHTML = e.beats.map((_, i) =>
      '<div class="seg' + (i < beat ? " done" : "") + '"><div class="fill"></div></div>'
    ).join("");

    $("#epname").textContent = "Ep " + String(e.number).padStart(2, "0") + " " + e.title;
    const m = epMeta(ep);
    $("#eptime").textContent = m ? fmt(m.duration) : "";
    syncScript();
    document.title = "Ep " + e.number + " · " + e.title + " · Process Compose";
  }

  function fmt(s) {
    s = Math.max(0, Math.round(s));
    return Math.floor(s / 60) + ":" + String(s % 60).padStart(2, "0");
  }

  function go(e, b, keepPlaying) {
    ep = Math.max(0, Math.min(EPS.length - 1, e));
    beat = Math.max(0, Math.min(EPS[ep].beats.length - 1, b));
    render();
    if (playing || keepPlaying) playBeat(); else audio.pause();
  }

  function next() {
    if (beat < EPS[ep].beats.length - 1) go(ep, beat + 1);
    else if (ep < EPS.length - 1) go(ep + 1, 0);
    else stop();
  }
  function prev() {
    if (beat > 0) go(ep, beat - 1);
    else if (ep > 0) go(ep - 1, EPS[ep - 1].beats.length - 1);
  }

  function playBeat() {
    const src = beatAudio(ep, beat);
    if (!src) { audio.pause(); return; }
    if (!audio.src.endsWith(src)) audio.src = src;
    audio.currentTime = 0;
    audio.play().catch(() => { /* autoplay blocked until a gesture */ });
  }

  function play() { playing = true; $("#play").textContent = "❚❚ Pause"; $("#play").classList.add("primary"); playBeat(); }
  function stop() { playing = false; $("#play").textContent = "▶ Play"; $("#play").classList.remove("primary"); audio.pause(); }
  function toggle() { playing ? stop() : play(); }

  audio.addEventListener("ended", () => { if (playing) next(); });
  audio.addEventListener("timeupdate", () => {
    const segs = track.children;
    if (!segs[beat] || !audio.duration) return;
    segs[beat].firstChild.style.width = (audio.currentTime / audio.duration * 100) + "%";
  });

  /* ----------------------------------------------------------- drawers -- */

  function buildEpisodes() {
    $("#eplist").innerHTML = EPS.map((e, i) => {
      const m = epMeta(i);
      return '<button class="epcard' + (i === ep ? " cur" : "") + '" data-ep="' + i + '">' +
        '<div class="n">Episode ' + String(e.number).padStart(2, "0") + "</div>" +
        '<div class="t">' + esc(e.title) + "</div>" +
        '<div class="d">' + esc(e.subtitle) + "</div>" +
        '<div class="m">' + e.beats.length + " beats" +
        (m ? "  ·  " + fmt(m.duration) : "") + "</div></button>";
    }).join("");
    $("#eplist").querySelectorAll(".epcard").forEach((b) =>
      b.addEventListener("click", () => {
        closeDrawers();
        go(+b.dataset.ep, 0);
      }));
  }

  function buildScript() {
    const e = EPS[ep];
    $("#scripthead").textContent =
      "Episode " + String(e.number).padStart(2, "0") + " · " + e.title;
    const m = epMeta(ep);
    const words = e.beats.reduce((a, b) => a + b.narration.split(/\s+/).length, 0);
    $("#scriptmeta").textContent =
      words + " words · " + e.beats.length + " beats" +
      (m ? " · " + fmt(m.duration) : "") +
      " · click a line to jump";
    $("#script").innerHTML = e.beats.map((b, i) => {
      const v = b.visual;
      const cue = v.type + (v.frame ? " " + v.frame : v.name ? " " + v.name : "");
      return '<div class="beat' + (i === beat ? " cur" : "") + '" data-b="' + i + '">' +
        '<span class="idx">' + String(i + 1).padStart(2, "0") + "</span>" +
        '<span class="cue">' + esc(cue) + "</span>" +
        "<p>" + esc(b.narration) + "</p></div>";
    }).join("");
    $("#script").querySelectorAll(".beat").forEach((el) =>
      el.addEventListener("click", () => { go(ep, +el.dataset.b); buildScript(); }));
  }

  function syncScript() {
    if (!$("#scriptdrawer").classList.contains("open")) return;
    buildScript();
    const cur = $("#script .beat.cur");
    if (cur) cur.scrollIntoView({ block: "center", behavior: "smooth" });
  }

  function closeDrawers() {
    document.querySelectorAll(".drawer").forEach((d) => d.classList.remove("open"));
    document.body.classList.remove("script-open");
  }

  function openDrawer(id) {
    const d = $(id);
    const wasOpen = d.classList.contains("open");
    closeDrawers();
    if (!wasOpen) {
      d.classList.add("open");
      if (id === "#scriptdrawer") { buildScript(); document.body.classList.add("script-open"); }
      if (id === "#epdrawer") buildEpisodes();
    }
  }

  /* -------------------------------------------------------------- wire -- */

  $("#next").addEventListener("click", next);
  $("#prev").addEventListener("click", prev);
  $("#play").addEventListener("click", toggle);
  $("#eps").addEventListener("click", () => openDrawer("#epdrawer"));
  $("#scr").addEventListener("click", () => openDrawer("#scriptdrawer"));
  $("#full").addEventListener("click", () => {
    if (document.fullscreenElement) document.exitFullscreen();
    else document.documentElement.requestFullscreen();
  });
  document.querySelectorAll(".drawer").forEach((d) =>
    d.addEventListener("click", (e) => { if (e.target === d) closeDrawers(); }));

  document.addEventListener("keydown", (e) => {
    if (e.metaKey || e.ctrlKey || e.altKey) return;
    switch (e.key) {
      case " ": case "ArrowRight": case "PageDown": e.preventDefault(); next(); break;
      case "ArrowLeft": case "PageUp": e.preventDefault(); prev(); break;
      case "]": e.preventDefault(); go(ep + 1, 0); break;
      case "[": e.preventDefault(); go(ep - 1, 0); break;
      case "Home": e.preventDefault(); go(ep, 0); break;
      case "p": case "P": e.preventDefault(); toggle(); break;
      case "e": case "E": openDrawer("#epdrawer"); break;
      case "s": case "S": openDrawer("#scriptdrawer"); break;
      case "f": case "F":
        if (document.fullscreenElement) document.exitFullscreen();
        else document.documentElement.requestFullscreen();
        break;
      case "Escape": closeDrawers(); break;
    }
  });

  // deep link: #ep3.5
  function fromHash() {
    const m = /^#ep(\d+)(?:\.(\d+))?$/.exec(location.hash || "");
    if (!m) return false;
    const i = EPS.findIndex((e) => e.number === +m[1]);
    if (i < 0) return false;
    go(i, m[2] ? +m[2] - 1 : 0);
    return true;
  }
  window.addEventListener("hashchange", fromHash);

  if (!fromHash()) render();
  buildEpisodes();
})();
