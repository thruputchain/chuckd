"""Offline profile selection and debuginfo packaging contracts."""
import fnmatch
import os
from pathlib import Path, PurePosixPath
import shutil
import subprocess
import tarfile
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]


def install_command(root, profile, heap=False):
    """The cargo install line build_scpd.sh runs for a profile."""
    install_root = root / ".scpd"
    args = ["install", "--locked", "--force", "--version", "2.0.0",
            "--root", str(install_root), "--target-dir", str(install_root / "target")]
    if profile == "debug":
        args.append("--debug")
    if heap:
        args += ["--features", "heap-profiling"]
    return " ".join(args + ["thruput-scp-server"])


def run_build(root, profile=None, heap=None):
    (root / "scripts").mkdir()
    shutil.copy(ROOT / "scripts/build_scpd.sh", root / "scripts/build_scpd.sh")
    fakebin = root / "bin"
    fakebin.mkdir()
    cargo = fakebin / "cargo"
    # Records its arguments and "installs" a binary named after the profile.
    cargo.write_text('#!/bin/sh\nprintf "%s\\n" "$*" >> "$PROBE_LOG"\n'
                     'mkdir -p "$PROBE_ROOT/.scpd/bin"\n'
                     'case " $* " in *" --debug "*) printf debug ;; *) printf release ;; esac'
                     ' > "$PROBE_ROOT/.scpd/bin/scpd"\n')
    cargo.chmod(0o755)
    env = {**os.environ, "PATH": str(fakebin) + os.pathsep + os.environ["PATH"],
           "PROBE_LOG": str(root / "cargo.log"), "PROBE_ROOT": str(root)}
    env.pop("SCPD_PROFILE", None)
    env.pop("SCPD_HEAP_PROFILING", None)
    if profile is not None:
        env["SCPD_PROFILE"] = profile
    if heap is not None:
        env["SCPD_HEAP_PROFILING"] = heap
    return subprocess.run(["bash", "scripts/build_scpd.sh"], cwd=root, env=env,
                          capture_output=True, text=True)


class ProfileTests(unittest.TestCase):
    def test_heap_profiling_is_opt_in(self):
        for heap, enabled in ((None, False), ("", False), ("0", False), ("1", True), ("yes", None)):
            with self.subTest(heap=heap), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)
                result = run_build(root, "release", heap)
                if enabled is None:
                    self.assertEqual(result.returncode, 2)
                    self.assertFalse((root / "cargo.log").exists())
                    continue
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual((root / "cargo.log").read_text().splitlines(),
                                 [install_command(root, "release", enabled)])

    def test_only_the_coordinator_release_build_enables_heap_profiling(self):
        # Benches and hog-cluster builds must not compile jemalloc (OOM risk).
        enablers = []
        tracked = subprocess.check_output(["git", "ls-files", "-z"], cwd=ROOT, text=True)
        for name in sorted(filter(None, tracked.split("\0"))):
            path = ROOT / name
            if (not path.is_file() or name in (".github/tests/test_profile_build.py",
                                               "scripts/build_scpd.sh")):
                continue
            try:
                text = path.read_text()
            except (UnicodeDecodeError, OSError):
                continue
            if "SCPD_HEAP_PROFILING" in text or "heap-profiling" in text:
                enablers.append(name)
        self.assertEqual(enablers, [".github/workflows/coordinator-build.yml"])
        workflow = (ROOT / ".github/workflows/coordinator-build.yml").read_text()
        self.assertIn('SCPD_HEAP_PROFILING: "1"', workflow)
        self.assertNotIn("self-hosted", workflow)

    def test_profiles_and_invalid_values(self):
        for profile, target in ((None, "debug"), ("dev", "debug"), ("release", "release"),
                                ("invalid", None), ("", None)):
            with self.subTest(profile=profile), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)
                result = run_build(root, profile)
                if target is None:
                    self.assertEqual(result.returncode, 2)
                    self.assertFalse((root / "cargo.log").exists())
                    self.assertFalse((root / "scpd.dev").exists())
                else:
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertEqual((root / "scpd.dev").read_text(), target)
                    self.assertEqual((root / "cargo.log").read_text().splitlines(),
                                     [install_command(root, target)])


CHUCKD_INSTALL_PATH = "/usr/bin/chuckd"
# gdb resolves a debuglink without a GNU build-id under /usr/lib/debug/<binary dir>/.
FALLBACK_DEBUG = "usr/lib/debug" + str(PurePosixPath(CHUCKD_INSTALL_PATH).parent) + "/chuckd.debug"


def first_file(*candidates):
    return next((p for p in candidates if p and Path(p).is_file()), None)


class PackagingTests(unittest.TestCase):
    # BFD target names GNU objcopy reports in --info for each fixture architecture.
    BFD_TARGETS = {"x86_64": "elf64-x86-64", "arm64": "elf64-littleaarch64"}

    @classmethod
    def setUpClass(cls):
        brew = "/opt/homebrew/opt"
        cls.tools = {
            "clang": first_file(shutil.which("clang"), brew + "/llvm/bin/clang"),
            "ld.lld": first_file(shutil.which("ld.lld"), shutil.which("llvm-ld.lld"),
                                 brew + "/lld/bin/ld.lld"),
        }
        # Prefer LLVM tools: they handle every ELF machine. A native GNU objcopy may
        # only know its host architecture; require_objcopy_support skips the rest.
        for name in ("objcopy", "readelf"):
            cls.tools[name] = first_file(shutil.which("llvm-" + name),
                                         brew + "/llvm/bin/llvm-" + name, shutil.which(name))
        if not all(cls.tools.values()):
            missing = sorted(name for name, path in cls.tools.items() if not path)
            raise unittest.SkipTest("ELF fixture tools unavailable: " + ", ".join(missing))
        # GNU objcopy lists its BFD targets; llvm-objcopy rejects --info.
        info = subprocess.run([cls.tools["objcopy"], "--info"], capture_output=True, text=True)
        cls.bfd_targets = info.stdout if info.returncode == 0 and "elf64-" in info.stdout else None

    def require_objcopy_support(self, architecture):
        target = self.BFD_TARGETS[architecture]
        if self.bfd_targets is not None and target not in self.bfd_targets.split():
            self.skipTest(f"{self.tools['objcopy']} does not support {target}; "
                          "install llvm-objcopy and llvm-readelf to cover " + architecture)

    def fixture(self, root, name, target, *, build_id=True):
        source = root / (name + ".c")
        source.write_text("int main(void) { return " + ("1" if name == "chuckd" else "2") + "; }\n")
        obj = root / (name + ".o")
        subprocess.run([self.tools["clang"], "--target=" + target, "-g", "-c", str(source), "-o", str(obj)], check=True, capture_output=True)
        args = [self.tools["ld.lld"], "-e", "main", str(obj), "-o", str(root / (name + ".dev"))]
        if build_id:
            args.append("--build-id=sha1")
        subprocess.run(args, check=True, capture_output=True)

    def test_split_real_elf_for_both_architectures(self):
        for architecture, target in (("x86_64", "x86_64-linux-gnu"), ("arm64", "aarch64-linux-gnu")):
            for fallback in (False, True):
                with self.subTest(architecture=architecture, fallback=fallback), tempfile.TemporaryDirectory() as tmp:
                    self.require_objcopy_support(architecture)
                    root = Path(tmp)
                    self.fixture(root, "chuckd", target, build_id=not fallback)
                    self.fixture(root, "scpd", target)
                    fakebin = root / "bin"
                    fakebin.mkdir()
                    for name in ("objcopy", "readelf"):
                        (fakebin / name).symlink_to(self.tools[name])
                    env = {**os.environ, "PATH": str(fakebin) + os.pathsep + os.environ["PATH"],
                           "COPYFILE_DISABLE": "1"}
                    result = subprocess.run(["bash", str(ROOT / "scripts/package_coordinator.sh"), architecture],
                                            cwd=root, env=env, capture_output=True, text=True)
                    self.assertEqual(result.returncode, 0, result.stderr)
                    runtime = f"chuckd-coordinator-linux-{architecture}"
                    debug = f"chuckd-coordinator-debuginfo-linux-{architecture}.tar.gz"
                    self.assertFalse(fnmatch.fnmatch(debug, "chuckd-coordinator-linux-*.tar.gz"))
                    with tarfile.open(root / (runtime + ".tar.gz")) as archive:
                        self.assertEqual(archive.getnames(), ["chuckd", "scpd"])
                    with tarfile.open(root / debug) as archive:
                        files = [m.name for m in archive.getmembers() if m.isfile()]
                        self.assertEqual(len(files), 2)
                        build_id_files = [f for f in files if f.startswith("usr/lib/debug/.build-id/")]
                        if fallback:
                            self.assertIn(FALLBACK_DEBUG, files)
                            self.assertEqual(len(build_id_files), 1)
                        else:
                            self.assertEqual(len(build_id_files), 2)
                            self.assertNotIn(FALLBACK_DEBUG, files)
                    for binary in ("chuckd", "scpd"):
                        shipped = root / runtime / binary
                        sections = subprocess.check_output([self.tools["readelf"], "-S", str(shipped)], text=True)
                        self.assertIn(".symtab", sections)
                        self.assertIn(".gnu_debuglink", sections)
                        self.assertNotIn(".debug_info", sections)
                        notes = subprocess.check_output([self.tools["readelf"], "-n", str(shipped)], text=True)
                        link_dump = root / (binary + ".debuglink")
                        subprocess.run([self.tools["objcopy"], "--dump-section", f".gnu_debuglink={link_dump}",
                                        str(shipped), str(root / (binary + ".scratch"))], check=True, capture_output=True)
                        link = link_dump.read_bytes().split(b"\0")[0].decode()
                        if binary == "chuckd" and fallback:
                            self.assertNotIn("Build ID: ", notes)
                            # gdb looks up /usr/lib/debug/<dir of installed binary>/<debuglink>.
                            self.assertEqual(link, "chuckd.debug")
                            self.assertEqual("/" + str(PurePosixPath(FALLBACK_DEBUG).relative_to("usr/lib/debug")),
                                             str(PurePosixPath(CHUCKD_INSTALL_PATH).parent / link))
                            debug_file = root / debug.removesuffix(".tar.gz") / FALLBACK_DEBUG
                        else:
                            build_id = notes.split("Build ID: ")[1].split()[0]
                            debug_file = root / debug.removesuffix(".tar.gz") / "usr/lib/debug/.build-id" / build_id[:2] / (build_id[2:] + ".debug")
                        debug_sections = subprocess.check_output([self.tools["readelf"], "-S", str(debug_file)], text=True)
                        self.assertIn(".debug_info", debug_sections)


if __name__ == "__main__":
    unittest.main()
