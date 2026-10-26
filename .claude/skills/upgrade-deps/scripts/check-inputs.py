#!/usr/bin/env python3
"""Check every workflow step's `with:` inputs against the action.yml at the pinned ref.

Usage: check-inputs.py [workflow dir]

Reports inputs the action does not declare (removed or renamed upstream, GitHub only warns
about these at run time) and required inputs without a default that the step omits.
Exits 1 when it finds either.
"""
import base64
import glob
import json
import os
import subprocess
import sys

import yaml

wf_dir = sys.argv[1] if len(sys.argv) > 1 else ".github/workflows"
cache = {}


def action_inputs(uses):
    """owner/repo[/path]@ref -> {input name: spec}, or None when no action.yml is found."""
    if uses in cache:
        return cache[uses]
    target, ref = uses.split("@", 1)
    parts = target.split("/")
    repo, path = "/".join(parts[:2]), "/".join(parts[2:])
    inputs = None
    for name in ("action.yml", "action.yaml"):
        file_path = f"{path}/{name}" if path else name
        res = subprocess.run(
            ["gh", "api", f"repos/{repo}/contents/{file_path}?ref={ref}"],
            capture_output=True, text=True,
        )
        if res.returncode == 0:
            meta = yaml.safe_load(base64.b64decode(json.loads(res.stdout)["content"]))
            inputs = meta.get("inputs") or {}
            break
    cache[uses] = inputs
    return inputs


problems = 0
for wf in sorted(glob.glob(os.path.join(wf_dir, "*.y*ml"))):
    doc = yaml.safe_load(open(wf))
    for job_name, job in (doc.get("jobs") or {}).items():
        for step in job.get("steps") or []:
            uses = step.get("uses")
            if not uses or uses.startswith(("./", "docker://")) or "/.github/workflows/" in uses:
                continue
            given = set((step.get("with") or {}).keys())
            inputs = action_inputs(uses)
            label = f"{os.path.basename(wf)} [{job_name}] {uses.split('@')[0]}"
            if inputs is None:
                print(f"?  {label}: no action.yml at {uses.split('@')[1]}")
                continue
            unknown = sorted(given - set(inputs))
            missing = sorted(
                k for k, v in inputs.items()
                if (v or {}).get("required") and "default" not in (v or {}) and k not in given
            )
            for k in unknown:
                print(f"X  {label}: input '{k}' is not declared by the action")
            for k in missing:
                print(f"X  {label}: required input '{k}' is missing")
            if unknown or missing:
                problems += 1
            else:
                print(f"ok {label}: {', '.join(sorted(given)) or 'no inputs'}")

sys.exit(1 if problems else 0)
