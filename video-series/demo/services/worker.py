#!/usr/bin/env python3
"""linkstash worker. Expires stale links and rolls up hit counts."""
import os
import signal
import sqlite3
import sys
import time
import random

DB = os.environ.get("LINKSTASH_DB", "./data/linkstash.db")
REPLICA = os.environ.get("PC_REPLICA_NUM", "0")
INTERVAL = float(os.environ.get("WORKER_INTERVAL", "3"))

running = True


def log(msg):
    print(f"{time.strftime('%H:%M:%S')} [w{REPLICA}] {msg}", flush=True)


def stop(signum, frame):
    global running
    log(f"received {signal.Signals(signum).name}, finishing current batch")
    running = False


def main():
    signal.signal(signal.SIGTERM, stop)
    signal.signal(signal.SIGINT, stop)

    log(f"worker replica {REPLICA} online, pid {os.getpid()}")
    batch = 0
    while running:
        batch += 1
        conn = sqlite3.connect(DB)
        now = int(time.time())
        # Simulate traffic so the rollup has something to do.
        slugs = [r[0] for r in conn.execute("SELECT slug FROM links LIMIT 50")]
        written = 0
        for s in slugs:
            if random.random() < 0.35:
                conn.execute("INSERT INTO hits (slug, seen_at) VALUES (?, ?)", (s, now))
                written += 1
        expired = conn.execute(
            "DELETE FROM links WHERE expires_at IS NOT NULL AND expires_at < ?", (now,)
        ).rowcount
        conn.commit()
        total = conn.execute("SELECT count(*) FROM hits").fetchone()[0]
        conn.close()

        log(f"batch {batch}: {written} hits recorded, {expired} expired, {total} total")
        if batch % 4 == 0:
            log(f"WARN queue depth {random.randint(12, 60)} above soft limit")

        slept = 0.0
        while running and slept < INTERVAL:
            time.sleep(0.1)
            slept += 0.1

    log("worker drained, exiting 0")
    return 0


if __name__ == "__main__":
    sys.exit(main())
