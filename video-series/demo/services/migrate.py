#!/usr/bin/env python3
"""One-shot schema migration for linkstash. Exits 0 when the schema is current."""
import os
import sqlite3
import sys
import time

DB = os.environ.get("LINKSTASH_DB", "./data/linkstash.db")

MIGRATIONS = [
    ("0001_create_links", """
        CREATE TABLE IF NOT EXISTS links (
            slug       TEXT PRIMARY KEY,
            url        TEXT NOT NULL,
            created_at INTEGER NOT NULL,
            expires_at INTEGER
        )
    """),
    ("0002_create_hits", """
        CREATE TABLE IF NOT EXISTS hits (
            id      INTEGER PRIMARY KEY AUTOINCREMENT,
            slug    TEXT NOT NULL,
            seen_at INTEGER NOT NULL
        )
    """),
    ("0003_index_hits_slug", """
        CREATE INDEX IF NOT EXISTS idx_hits_slug ON hits(slug)
    """),
]


def log(msg):
    print(msg, flush=True)


def main():
    os.makedirs(os.path.dirname(DB), exist_ok=True)
    log(f"connecting to {DB}")
    conn = sqlite3.connect(DB)
    conn.execute("CREATE TABLE IF NOT EXISTS schema_migrations (name TEXT PRIMARY KEY)")
    applied = {r[0] for r in conn.execute("SELECT name FROM schema_migrations")}

    for name, ddl in MIGRATIONS:
        if name in applied:
            log(f"skip    {name} (already applied)")
            continue
        time.sleep(0.6)
        conn.execute(ddl)
        conn.execute("INSERT INTO schema_migrations (name) VALUES (?)", (name,))
        conn.commit()
        log(f"applied {name}")

    count = conn.execute("SELECT count(*) FROM schema_migrations").fetchone()[0]
    conn.close()
    log(f"schema up to date, {count} migrations total")
    return 0


if __name__ == "__main__":
    sys.exit(main())
