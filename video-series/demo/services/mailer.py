#!/usr/bin/env python3
"""Outbound mail relay. Prints a ready line that process-compose watches for."""
import os
import signal
import sys
import time
import random

QUEUE = os.environ.get("MAILER_QUEUE", "./data/mail")
BOOT_DELAY = float(os.environ.get("MAILER_BOOT_DELAY", "4"))

running = True


def log(msg):
    print(f"{time.strftime('%H:%M:%S')} mailer {msg}", flush=True)


def stop(signum, frame):
    global running
    log(f"caught {signal.Signals(signum).name}, flushing spool")
    running = False


def main():
    signal.signal(signal.SIGTERM, stop)
    signal.signal(signal.SIGINT, stop)

    os.makedirs(QUEUE, exist_ok=True)
    log("opening spool directory")
    time.sleep(BOOT_DELAY / 2)
    log("verifying relay credentials")
    time.sleep(BOOT_DELAY / 2)
    # process-compose watches stdout for this exact line via ready_log_line.
    log("relay ready, accepting mail")

    sent = 0
    while running:
        time.sleep(2.5)
        if not running:
            break
        n = random.randint(0, 3)
        sent += n
        if n:
            log(f"delivered {n} message(s), {sent} this session")
        else:
            log("spool empty")

    log("mailer stopped")
    return 0


if __name__ == "__main__":
    sys.exit(main())
