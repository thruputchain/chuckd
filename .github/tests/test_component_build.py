"""Regression tests for the manual component-build workflow (no network)."""
from __future__ import annotations

import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import textwrap
import unittest

ROOT = Path(__file__).resolve().parents[2]
PUBLISHER = ROOT / ".github/workflows/component-build.yml"
SHA = "1234567890abcdef1234567890abcdef12345678"


def run_block(path: Path, name: str) -> str:
    """Extract one named literal run block without adding a YAML dependency."""
    lines = path.read_text().splitlines()
    start = next(i for i, line in enumerate(lines) if line.strip().removeprefix("- ") == f"name: {name}")
    run = next(i for i in range(start + 1, len(lines)) if lines[i].strip() == "run: |")
    indent = len(lines[run]) - len(lines[run].lstrip())
    body = []
    for line in lines[run + 1:]:
        if line.strip() and len(line) - len(line.lstrip()) <= indent:
            break
        body.append(line)
    return textwrap.dedent("\n".join(body)) + "\n"


class MeasurementTests(unittest.TestCase):
    def measure(self, timing: str | None, *, platform: str = "linux/amd64", attempt: str = "1") -> dict:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            measurements = root / "chuckd-measurement"
            measurements.mkdir()
            if timing is not None:
                (measurements / "build-time.txt").write_text(timing)
                (measurements / "build-start-epoch").write_text("100")
                (measurements / "build-end-epoch").write_text("112")
                (measurements / "runner-start-epoch").write_text("90")
                (measurements / "workflow-created-at").write_text("1970-01-01T00:01:20Z")
                architecture = "x86_64" if platform == "linux/amd64" else "arm64"
                (root / f"chuckd-coordinator-linux-{architecture}.tar.gz").write_bytes(b"fixture")
            env = {**os.environ, "RUNNER_TEMP": tmp, "PLATFORM": platform,
                   "GITHUB_REPOSITORY": "thruputchain/chuckd", "GITHUB_RUN_ID": "12",
                   "GITHUB_RUN_ATTEMPT": attempt, "SOURCE_COMMIT": SHA,
                   "COMPONENT_VERSION": "0.0.0-main.g" + SHA[:12],
                   "BUILD_OUTCOME": "success" if timing is not None else "",
                   "PACKAGE_OUTCOME": "success" if timing is not None else "",
                   "CACHE_HIT_SIGNAL": " "}
            result = subprocess.run([sys.executable, str(ROOT / ".github/scripts/write-build-measurement.py")], cwd=root, env=env, capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            return json.loads((measurements / "build-measurement.json").read_text())

    def test_gnu_time_indentation_is_parsed(self):
        record = self.measure('\tCommand being timed: "build"\n\tUser time (seconds): 3600.00\n\tSystem time (seconds): 1800.00\n')
        self.assertEqual(record["build"]["cpuHours"], 1.5)
        self.assertEqual(record["build"]["durationSeconds"], 12)
        self.assertIsNone(record["build"]["allocatedCpuHours"])
        self.assertGreater(record["build"]["visibleCpuCount"], 0)
        self.assertGreater(record["build"]["visibleCpuHours"], 0)
        self.assertEqual(record["build"]["artifactBytes"], 7)
        self.assertEqual(record["run"]["queueDelaySeconds"], 10)

    def test_malformed_time_is_unknown_not_a_script_failure(self):
        record = self.measure("User time (seconds): ...\nSystem time (seconds): 1.0\n")
        self.assertIsNone(record["build"]["cpuHours"])

    def test_missing_build_still_emits_failure_evidence(self):
        record = self.measure(None)
        self.assertEqual(record["build"]["outcome"], "not-run")
        for field in ("durationSeconds", "cpuHours", "allocatedCpuHours", "artifactBytes"):
            self.assertIsNone(record["build"][field])
        self.assertIsNone(record["run"]["queueDelaySeconds"])
        self.assertIsNone(record["cache"]["hitRate"])

    def test_zero_cpu_time_and_arm64_retry(self):
        record = self.measure("  User time (seconds): 0.0\n  System time (seconds): 0.0\n", platform="linux/arm64", attempt="2")
        self.assertEqual(record["build"]["cpuHours"], 0)
        self.assertEqual(record["build"]["artifactBytes"], 7)
        self.assertTrue(record["build"]["retry"])
        self.assertIsNone(record["cache"]["hitSignal"])


class WorkflowTests(unittest.TestCase):
    def run_authorize(self, target: str, *, ancestor=True, existing=False, sha=SHA, api_status="0"):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            fakebin = root / "bin"
            fakebin.mkdir()
            git = fakebin / "git"
            git.write_text("#!/bin/sh\ncase \"$1\" in\nfetch) exit 0;;\nrev-parse) printf '%s\\n' \"$PROBE_SHA\";;\nmerge-base) exit \"$PROBE_ANCESTOR_STATUS\";;\n*) exit 99;;\nesac\n")
            gh = fakebin / "gh"
            gh.write_text("#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$PROBE_GH_LOG\"\nif [ \"$1\" = api ]; then\n  if [ \"$PROBE_API_STATUS\" != 0 ]; then exit \"$PROBE_API_STATUS\"; fi\n  if [ \"$PROBE_RELEASE_STATUS\" = 0 ]; then echo 1; else echo 0; fi\n  exit 0\nfi\nexit \"$PROBE_RELEASE_STATUS\"\n")
            git.chmod(0o755)
            gh.chmod(0o755)
            output = root / "output"
            log = root / "gh.log"
            env = {**os.environ, "PATH": str(fakebin) + os.pathsep + os.environ["PATH"],
                   "TARGET_COMMIT": target, "GITHUB_OUTPUT": str(output),
                   "GITHUB_REPOSITORY": "thruputchain/chuckd", "PROBE_SHA": sha,
                   "PROBE_ANCESTOR_STATUS": "0" if ancestor else "1",
                   "PROBE_RELEASE_STATUS": "0" if existing else "1", "PROBE_GH_LOG": str(log), "PROBE_API_STATUS": api_status}
            result = subprocess.run(["bash", "-c", run_block(PUBLISHER, "Resolve and authorize target commit")], cwd=root, env=env, capture_output=True, text=True)
            return result, output.read_text() if output.exists() else "", log.read_text() if log.exists() else ""

    def test_source_determines_version_independent_of_ref_spelling(self):
        for ref in ("main", SHA, "v0.1.27-rc.4"):
            result, output, _ = self.run_authorize(ref)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("source_commit=" + SHA, output)
            self.assertIn("component_version=0.0.0-main.g" + SHA[:12], output)

    def test_invalid_refs_never_reach_publication_preflight(self):
        for ref in ("", "-main", "../main", "a..b", "main;echo bad", "main\nother"):
            result, output, log = self.run_authorize(ref)
            self.assertNotEqual(result.returncode, 0, ref)
            self.assertEqual(output, "")
            self.assertEqual(log, "")

    def test_non_ancestor_and_invalid_resolved_sha_are_rejected(self):
        for result, output, log in (
            self.run_authorize("main", ancestor=False),
            self.run_authorize("main", sha="bad"),
        ):
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(output, "")
            self.assertEqual(log, "")

    def test_existing_coordinate_is_refused_before_build(self):
        result, output, log = self.run_authorize("main", existing=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("refusing", result.stderr)
        self.assertEqual(output, "")
        self.assertIn("packaging-main-" + SHA, log)
        self.assertNotIn("create", log)
        self.assertNotIn("delete", log)

    def test_api_failure_cannot_authorize_a_build(self):
        result, output, _ = self.run_authorize("main", api_status="2")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(output, "")

    def test_source_version_reaches_build_and_descriptor(self):
        workflow = PUBLISHER.read_text()
        self.assertNotIn("inputs.component_version", workflow)
        self.assertEqual(workflow.count("needs.authorize.outputs.component_version"), 2)
        self.assertIn("cancel-in-progress: false", workflow)
        self.assertIn("if: github.ref == 'refs/heads/main'", workflow)

    def test_descriptor_uses_exact_roles_and_digests(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            artifacts = root / "artifacts"
            artifacts.mkdir()
            expected = {}
            for platform, architecture in (("linux/amd64", "x86_64"), ("linux/arm64", "arm64")):
                name = f"chuckd-coordinator-linux-{architecture}.tar.gz"
                data = architecture.encode()
                (artifacts / name).write_bytes(data)
                expected[platform] = {"coordinator-archive": {"filename": name, "sha256": "sha256:" + hashlib.sha256(data).hexdigest()}}
            # Emulate GNU sha256sum on macOS as well as Linux. Hash real bytes.
            fakebin = root / "bin"
            fakebin.mkdir()
            checksum = fakebin / "sha256sum"
            checksum.write_text(f"#!{sys.executable}\nimport hashlib, pathlib, sys\np=sys.argv[1]\nprint(hashlib.sha256(pathlib.Path(p).read_bytes()).hexdigest()+'  '+p)\n")
            checksum.chmod(0o755)
            version = "0.0.0-main.g" + SHA[:12]
            env = {**os.environ, "PATH": str(fakebin) + os.pathsep + os.environ["PATH"], "SOURCE_COMMIT": SHA,
                   "COMPONENT_VERSION": version, "GITHUB_REPOSITORY": "thruputchain/chuckd",
                   "GITHUB_STEP_SUMMARY": str(root / "summary")}
            result = subprocess.run(["bash", "-c", run_block(PUBLISHER, "Generate component-build-v1 descriptor")], cwd=root, env=env, capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            descriptor = json.loads((artifacts / "groundhog-component-build.json").read_text())
            self.assertEqual(set(descriptor), {"schema", "component", "artifacts"})
            self.assertEqual(descriptor["schema"], "groundhog-component-build-v1")
            self.assertEqual(descriptor["component"], {"name": "chuckd-go", "repository": "thruputchain/chuckd", "sourceCommit": SHA, "version": version})
            self.assertEqual(descriptor["artifacts"], expected)

    def run_publish(self, *, api_status="0", release_id="", tag_status="0", complete=True):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            artifacts = root / "artifacts"
            artifacts.mkdir()
            if complete:
                for name in ("chuckd-coordinator-linux-x86_64.tar.gz", "chuckd-coordinator-linux-arm64.tar.gz",
                             "chuckd-coordinator-debuginfo-linux-x86_64.tar.gz", "chuckd-coordinator-debuginfo-linux-arm64.tar.gz",
                             "groundhog-component-build.json"):
                    (artifacts / name).write_text("fixture")
            # Unexpected downloaded files must never enter the release.
            (artifacts / "unexpected.txt").write_text("not a release asset")
            fakebin = root / "bin"
            fakebin.mkdir()
            gh = fakebin / "gh"
            gh.write_text(textwrap.dedent("""\
                #!/bin/sh
                printf '%s\\n' "$*" >> "$PROBE_GH_LOG"
                case "$*" in
                  "api --paginate "*)
                    [ "$PROBE_API_STATUS" = 0 ] || exit "$PROBE_API_STATUS"
                    printf '%s' "$PROBE_RELEASE_ID" ;;
                  "api --method POST "*) exit "$PROBE_TAG_STATUS" ;;
                  "release create "*) exit 0 ;;
                  *) exit 99 ;;
                esac
                """))
            gh.chmod(0o755)
            log = root / "gh.log"
            env = {**os.environ, "PATH": str(fakebin) + os.pathsep + os.environ["PATH"],
                   "SOURCE_COMMIT": SHA, "GITHUB_REPOSITORY": "thruputchain/chuckd",
                   "PROBE_GH_LOG": str(log), "PROBE_API_STATUS": api_status,
                   "PROBE_RELEASE_ID": release_id, "PROBE_TAG_STATUS": tag_status}
            result = subprocess.run(["bash", "-c", run_block(PUBLISHER, "Refuse replacement and publish immutable prerelease")], cwd=root, env=env, capture_output=True, text=True)
            return result, log.read_text() if log.exists() else ""

    def test_publisher_api_failure_stops_before_any_write(self):
        result, log = self.run_publish(api_status="2")
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn("POST", log)
        self.assertNotIn("release create", log)

    def test_publisher_refuses_existing_draft_or_published_release(self):
        result, log = self.run_publish(release_id="123")
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn("POST", log)
        self.assertNotIn("release create", log)

    def test_publisher_refuses_tag_creation_race(self):
        result, log = self.run_publish(tag_status="1")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("POST", log)
        self.assertNotIn("release create", log)

    def test_publisher_requires_complete_assets_before_writing(self):
        result, log = self.run_publish(complete=False)
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn("POST", log)
        self.assertNotIn("release create", log)

    def test_publisher_creates_exact_tag_and_only_declared_assets(self):
        result, log = self.run_publish()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("ref=refs/tags/packaging-main-" + SHA, log)
        self.assertIn("sha=" + SHA, log)
        self.assertIn("--verify-tag", log)
        self.assertIn("--latest=false", log)
        self.assertIn("chuckd-coordinator-debuginfo-linux-x86_64.tar.gz", log)
        self.assertIn("chuckd-coordinator-debuginfo-linux-arm64.tar.gz", log)
        self.assertNotIn("unexpected.txt", log)
        self.assertNotIn("delete", log)
        self.assertNotIn("PATCH", log)

    def test_historical_build_uses_workflow_measurement_tooling(self):
        workflow = (ROOT / ".github/workflows/coordinator-build.yml").read_text()
        self.assertIn("ref: ${{ github.workflow_sha }}", workflow)
        self.assertIn("measurement_script=.workflow-tooling/.github/scripts/write-build-measurement.py", workflow)

    def test_historical_build_uses_workflow_packaging_tooling(self):
        workflow = (ROOT / ".github/workflows/coordinator-build.yml").read_text()
        self.assertIn("            scripts/package_coordinator.sh\n", workflow)
        self.assertIn("bash .workflow-tooling/scripts/package_coordinator.sh", workflow)
        self.assertLess(workflow.index("path: .workflow-tooling"), workflow.index("id: package"))


if __name__ == "__main__":
    unittest.main()
