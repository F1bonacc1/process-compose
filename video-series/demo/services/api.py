#!/usr/bin/env python3
"""linkstash API. Real HTTP server with a /health endpoint for readiness probes."""
import json
import os
import signal
import sqlite3
import sys
import time
import random
import string
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

DB = os.environ.get("LINKSTASH_DB", "./data/linkstash.db")
PORT = int(os.environ.get("API_PORT", "8110"))
BOOT_DELAY = float(os.environ.get("API_BOOT_DELAY", "2.5"))

started_at = time.time()
ready = False


def log(msg):
    print(f"{time.strftime('%H:%M:%S')} {msg}", flush=True)


def slug():
    return "".join(random.choices(string.ascii_lowercase + string.digits, k=7))


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def _send(self, code, payload):
        body = json.dumps(payload).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Access-Control-Allow-Origin", "*")
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        if self.path == "/health":
            if not ready:
                self._send(503, {"status": "starting"})
                return
            self._send(200, {"status": "ok", "uptime_s": round(time.time() - started_at, 1)})
            return

        if self.path == "/links":
            conn = sqlite3.connect(DB)
            rows = conn.execute(
                "SELECT slug, url, created_at FROM links ORDER BY created_at DESC LIMIT 20"
            ).fetchall()
            conn.close()
            self._send(200, {"links": [{"slug": r[0], "url": r[1], "created_at": r[2]} for r in rows]})
            return

        if self.path == "/stats":
            conn = sqlite3.connect(DB)
            links = conn.execute("SELECT count(*) FROM links").fetchone()[0]
            hits = conn.execute("SELECT count(*) FROM hits").fetchone()[0]
            conn.close()
            self._send(200, {"links": links, "hits": hits})
            return

        self._send(404, {"error": "not found"})

    def do_POST(self):
        if self.path != "/links":
            self._send(404, {"error": "not found"})
            return
        length = int(self.headers.get("Content-Length", "0"))
        raw = self.rfile.read(length) if length else b"{}"
        try:
            url = json.loads(raw).get("url", "https://example.com")
        except json.JSONDecodeError:
            url = "https://example.com"
        s = slug()
        conn = sqlite3.connect(DB)
        conn.execute(
            "INSERT INTO links (slug, url, created_at) VALUES (?, ?, ?)",
            (s, url, int(time.time())),
        )
        conn.commit()
        conn.close()
        log(f'POST /links -> created "{s}"')
        self._send(201, {"slug": s, "url": url})

    def log_message(self, fmt, *args):
        if "/health" not in (args[0] if args else ""):
            log(f"{self.address_string()} {fmt % args}")


class Server(ThreadingHTTPServer):
    daemon_threads = True

    def handle_error(self, request, client_address):
        # Probes and page reloads drop keep-alive sockets. That is normal and
        # should not spam the log with tracebacks.
        exc = sys.exc_info()[1]
        if isinstance(exc, (ConnectionResetError, BrokenPipeError)):
            return
        super().handle_error(request, client_address)


def shutdown(signum, frame):
    log(f"received {signal.Signals(signum).name}, draining connections")
    time.sleep(0.4)
    log("api stopped cleanly")
    sys.exit(0)


def main():
    global ready
    signal.signal(signal.SIGTERM, shutdown)
    signal.signal(signal.SIGINT, shutdown)

    log(f"linkstash api starting, pid {os.getpid()}")
    log(f"database {DB}")
    log("loading routes: GET /health GET /links POST /links GET /stats")
    time.sleep(BOOT_DELAY)

    server = Server(("0.0.0.0", PORT), Handler)
    ready = True
    log(f"listening on http://0.0.0.0:{PORT}")
    server.serve_forever()


if __name__ == "__main__":
    main()
