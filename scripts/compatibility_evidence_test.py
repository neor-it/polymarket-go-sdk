"""Exercise compatibility evidence failure handling without network or Go builds."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile
import unittest

SOURCE_SCRIPT = Path(__file__).resolve().with_name("compatibility-evidence.sh")
WEEK = "2026-W43"


class CompatibilityEvidenceTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="sdk evidence qa ")
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name).resolve()
        self.repo = self.root / "fixture repository"
        self.caller = self.root / "caller directory"
        self.tools = self.root / "fake tools"
        for directory in (self.repo / "scripts", self.caller, self.tools):
            directory.mkdir(parents=True)
        self.script = self.repo / "scripts" / SOURCE_SCRIPT.name
        shutil.copyfile(SOURCE_SCRIPT, self.script)
        self.git("init", "-q")
        self.git("add", "scripts")
        self.git("-c", "user.name=Evidence QA", "-c", "user.email=qa@example.invalid", "commit", "-qm", "fixture")
        self.commit = self.git("rev-parse", "HEAD").strip()
        self.calls_file = self.root / "go calls.jsonl"
        fake_go = self.tools / "go"
        fake_go.write_text("""#!/usr/bin/env python3
import json, os, sys
with open(os.environ['QA_GO_CALLS'], 'a', encoding='utf-8') as output:
    output.write(json.dumps({'cwd':os.getcwd(),'args':sys.argv[1:]})+'\\n')
kind = 'schema' if './pkg/clob/ws' in sys.argv else 'compile'
print('executed '+kind+' fixture check')
sys.exit(1 if os.environ.get('QA_FAIL_CHECK') == kind else 0)
""")
        fake_go.chmod(0o755)
        self.environment = dict(os.environ, PATH=f"{self.tools}{os.pathsep}{os.environ['PATH']}", GOCACHE=str(self.root / "cache directory"), QA_GO_CALLS=str(self.calls_file))

    def git(self, *arguments):
        return subprocess.run(["git", *arguments], cwd=self.repo, check=True, capture_output=True, text=True).stdout

    def run_script(self, *arguments, fail=None):
        environment = dict(self.environment)
        if fail is not None:
            environment["QA_FAIL_CHECK"] = fail
        return subprocess.run(["bash", str(self.script), *arguments], cwd=self.caller, env=environment, capture_output=True, text=True, timeout=30)

    def assert_artifact(self, directory, passed):
        manifest = json.loads((directory / "manifest.json").read_text())
        self.assertEqual(manifest["commit_sha"], self.commit)
        self.assertEqual(manifest["all_passed"], passed)
        checksum = manifest.pop("manifest_checksum_sha256")
        self.assertEqual(checksum, hashlib.sha256(json.dumps(manifest, sort_keys=True, separators=(",", ":")).encode()).hexdigest())
        self.assertEqual(len(manifest["checks"]), 2)
        for check in manifest["checks"]:
            log = Path(check["log_file"])
            self.assertTrue(log.is_file())
            self.assertEqual(check["log_checksum_sha256"], hashlib.sha256(log.read_bytes()).hexdigest())
        digest, filename = (directory / "archive.sha256").read_text().strip().split("  ", 1)
        archive_path = directory / filename
        self.assertEqual(digest, hashlib.sha256(archive_path.read_bytes()).hexdigest())
        with tarfile.open(archive_path) as archive:
            contents = {name.removeprefix("./") for name in archive.getnames()}
            self.assertTrue({"manifest.json", "summary.md", "ws_schema_compat.log", "pkg_compile_gate.log"}.issubset(contents))
        return manifest

    def test_space_paths_foreign_cwd_checksums_and_source_identity(self):
        center = self.caller / "command center.json"
        center.write_text('{"status":"test evidence"}')
        result = self.run_script("--week", WEEK, "--out-dir", "output with spaces", "--command-center", center.name)
        self.assertEqual(result.returncode, 0, result.stderr)
        manifest = self.assert_artifact(self.caller / "output with spaces" / WEEK, True)
        self.assertFalse(manifest["working_tree_dirty"])
        self.assertEqual(manifest["command_center"]["checksum_sha256"], hashlib.sha256(center.read_bytes()).hexdigest())
        self.assertEqual(Path(manifest["command_center"]["path"]), center)
        calls = [json.loads(line) for line in self.calls_file.read_text().splitlines()]
        self.assertEqual(len(calls), 2)
        self.assertTrue(all(call["cwd"] == str(self.repo) for call in calls))
        self.assertEqual([check["status"] for check in manifest["checks"]], ["pass", "pass"])

    def test_dirty_source_is_reported_without_hiding_successful_checks(self):
        (self.repo / "untracked source.txt").write_text("changed")
        result = self.run_script("--week", WEEK, "--out-dir", "dirty evidence")
        self.assertEqual(result.returncode, 0, result.stderr)
        manifest = self.assert_artifact(self.caller / "dirty evidence" / WEEK, True)
        self.assertTrue(manifest["working_tree_dirty"])

    def test_each_failed_check_still_collects_both_logs_and_archive(self):
        for failure in ("schema", "compile"):
            with self.subTest(failure=failure):
                result = self.run_script("--week", WEEK, "--out-dir", f"failed {failure}", fail=failure)
                self.assertEqual(result.returncode, 1, result.stderr)
                manifest = self.assert_artifact(self.caller / f"failed {failure}" / WEEK, False)
                statuses = [check["status"] for check in manifest["checks"]]
                self.assertEqual(statuses, ["fail", "pass"] if failure == "schema" else ["pass", "fail"])
                self.assertIn("one or more compatibility checks failed", result.stderr)

    def test_invalid_arguments_fail_before_running_checks(self):
        for arguments in (
            ("--week",), ("--out-dir",), ("--command-center",),
            ("--out-dir", ""), ("--unknown",),
            ("--week", "2026-W00"), ("--week", "2026-W54"),
            ("--week", "2026-W43/../escape"),
            ("--out-dir", "tab\tpath"), ("--out-dir", "line\npath"),
            ("--command-center", "missing file"),
        ):
            with self.subTest(arguments=arguments):
                result = self.run_script(*arguments)
                self.assertEqual(result.returncode, 2, (arguments, result.stderr))
                self.assertFalse(self.calls_file.exists(), "invalid arguments must never produce successful check evidence")


if __name__ == "__main__":
    unittest.main()
