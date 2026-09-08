#!/usr/bin/env python3
"""Write one aggregatable measurement record for a coordinator build."""

from __future__ import annotations

import json
import os
import re
from datetime import datetime
from pathlib import Path


measurement_dir = Path(os.environ["RUNNER_TEMP"]) / "chuckd-measurement"


def read_int(name: str) -> int | None:
    try:
        return int((measurement_dir / name).read_text(encoding="utf-8").strip())
    except (OSError, ValueError):
        return None


def elapsed(start_name: str, end_name: str) -> int | None:
    start = read_int(start_name)
    end = read_int(end_name)
    return None if start is None or end is None else max(0, end - start)


def time_value(label: str) -> float | None:
    try:
        text = (measurement_dir / "build-time.txt").read_text(encoding="utf-8")
    except OSError:
        return None
    match = re.search(rf"^[ \t]*{re.escape(label)}:[ \t]*([0-9]+(?:\.[0-9]+)?)[ \t]*$", text, re.MULTILINE)
    return float(match.group(1)) if match else None


runner_start = read_int("runner-start-epoch")
try:
    created_at = (measurement_dir / "workflow-created-at").read_text(encoding="utf-8").strip()
    workflow_created = int(datetime.fromisoformat(created_at.replace("Z", "+00:00")).timestamp())
except (OSError, ValueError):
    workflow_created = None

build_seconds = elapsed("build-start-epoch", "build-end-epoch")
user_seconds = time_value("User time (seconds)")
system_seconds = time_value("System time (seconds)")
cpu_seconds = None if user_seconds is None or system_seconds is None else user_seconds + system_seconds
archive = Path(f"chuckd-coordinator-linux-{os.environ['PLATFORM'].split('/')[-1].replace('amd64', 'x86_64')}.tar.gz")
cache_signal = os.environ.get("CACHE_HIT_SIGNAL", "").strip()
record = {
    "schema": "groundhog-component-build-measurement-v1",
    "component": "chuckd-go",
    "repository": os.environ["GITHUB_REPOSITORY"],
    "sourceCommit": os.environ["SOURCE_COMMIT"],
    "version": os.environ["COMPONENT_VERSION"],
    "platform": os.environ["PLATFORM"],
    "run": {
        "id": int(os.environ["GITHUB_RUN_ID"]),
        "attempt": int(os.environ["GITHUB_RUN_ATTEMPT"]),
        "queueDelaySeconds": None if runner_start is None or workflow_created is None else max(0, runner_start - workflow_created),
    },
    "build": {
        "outcome": os.environ.get("BUILD_OUTCOME") or "not-run",
        "durationSeconds": build_seconds,
        "cpuHours": None if cpu_seconds is None else round(cpu_seconds / 3600, 6),
        "allocatedCpuHours": None,
        "visibleCpuCount": os.cpu_count(),
        "visibleCpuHours": None if build_seconds is None else round((os.cpu_count() or 1) * build_seconds / 3600, 6),
        "allocationLimitation": "Visible OS CPUs need not equal runner allocation; allocated CPU hours are unknown.",
        "artifactBytes": archive.stat().st_size if archive.is_file() else None,
        "packageOutcome": os.environ.get("PACKAGE_OUTCOME") or "not-run",
        "retry": int(os.environ["GITHUB_RUN_ATTEMPT"]) > 1,
    },
    "cache": {
        "provider": "namespacelabs/nscloud-cache-action",
        "hitRate": None,
        "hitSignal": cache_signal or None,
        "limitation": "The cache action does not expose hit and lookup counts, so a numeric hit rate cannot be measured.",
    },
}
measurement_dir.mkdir(parents=True, exist_ok=True)
(measurement_dir / "build-measurement.json").write_text(
    json.dumps(record, sort_keys=True, separators=(",", ":")) + "\n", encoding="utf-8"
)
