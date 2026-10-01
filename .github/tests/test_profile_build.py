"""Offline profile selection and debuginfo packaging contracts."""
import fnmatch
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]


class ProfileTests(unittest.TestCase):
    def test_profiles_and_invalid_values(self):
        for profile, target in ((None, "debug"), ("dev", "debug"), ("release", "release"),
                                ("invalid", None), ("", None)):
            with self.subTest(profile=profile), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)
                (root / "scripts").mkdir()
                shutil.copy(ROOT / "scripts/build_scpd.sh", root / "scripts/build_scpd.sh")
                for name in ("debug", "release"):
                    folder = root / "scp/target" / name
                    folder.mkdir(parents=True)
                    (folder / "scpd").write_text(name)
                fakebin = root / "bin"
                fakebin.mkdir()
                cargo = fakebin / "cargo"
                cargo.write_text('#!/bin/sh\nprintf "%s\\n" "$*" >> "$PROBE_LOG"\n')
                cargo.chmod(0o755)
                env = {**os.environ, "PATH": str(fakebin) + os.pathsep + os.environ["PATH"],
                       "PROBE_LOG": str(root / "cargo.log")}
                env.pop("SCPD_PROFILE", None)
                if profile is not None:
                    env["SCPD_PROFILE"] = profile
                result = subprocess.run(["bash", "scripts/build_scpd.sh"], cwd=root, env=env,
                                        capture_output=True, text=True)
                if target is None:
                    self.assertEqual(result.returncode, 2)
                    self.assertFalse((root / "cargo.log").exists())
                    self.assertFalse((root / "scpd.dev").exists())
                else:
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertEqual((root / "scpd.dev").read_text(), target)
                    expected = "build --offline --locked "
                    if profile == "release":
                        expected += "--release "
                    expected += "-p scp_server --bin scpd"
                    self.assertEqual((root / "cargo.log").read_text().splitlines(),
                                     ["--config net.git-fetch-with-cli=true fetch --manifest-path Cargo.toml", expected])


class PackagingTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.tools = {}
        for name in ("clang", "ld.lld", "objcopy", "readelf"):
            alternatives = [shutil.which(name), shutil.which("llvm-" + name)]
            if name == "ld.lld":
                alternatives.append("/opt/homebrew/opt/lld/bin/ld.lld")
            else:
                alternatives.append("/opt/homebrew/opt/llvm/bin/" +
                                    (name if name == "clang" else "llvm-" + name))
            cls.tools[name] = next((p for p in alternatives if p and Path(p).is_file()), None)
        if not all(cls.tools.values()):
            raise unittest.SkipTest("ELF fixture tools unavailable")

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
                        if fallback:
                            self.assertIn("usr/lib/debug/chuckd.debug", files)
                    for binary in ("chuckd", "scpd"):
                        shipped = root / runtime / binary
                        sections = subprocess.check_output([self.tools["readelf"], "-S", str(shipped)], text=True)
                        self.assertIn(".symtab", sections)
                        self.assertIn(".gnu_debuglink", sections)
                        self.assertNotIn(".debug_info", sections)
                        notes = subprocess.check_output([self.tools["readelf"], "-n", str(shipped)], text=True)
                        if binary == "chuckd" and fallback:
                            debug_file = root / debug.removesuffix(".tar.gz") / "usr/lib/debug/chuckd.debug"
                        else:
                            build_id = notes.split("Build ID: ")[1].split()[0]
                            debug_file = root / debug.removesuffix(".tar.gz") / "usr/lib/debug/.build-id" / build_id[:2] / (build_id[2:] + ".debug")
                        debug_sections = subprocess.check_output([self.tools["readelf"], "-S", str(debug_file)], text=True)
                        self.assertIn(".debug_info", debug_sections)


if __name__ == "__main__":
    unittest.main()
