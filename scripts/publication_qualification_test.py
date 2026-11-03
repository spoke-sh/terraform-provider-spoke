"""The installed lane must reject missing or skipped execution, not just exit 0."""
import importlib.util
import json
from pathlib import Path
import tempfile
import subprocess
import sys
import unittest

spec = importlib.util.spec_from_file_location("qualification", Path(__file__).with_name("qualify-publication-terraform.py"))
qualification = importlib.util.module_from_spec(spec)
spec.loader.exec_module(qualification)


class QualificationTests(unittest.TestCase):
    def test_selected_fixture_resolves_its_sibling_helpers_outside_checkout(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            tests = root / "platform/tests"
            tests.mkdir(parents=True)
            (tests / "fixture_helper.py").write_text("VALUE = 'selected fixture'\n")
            (tests / "publication-http.integration.py").write_text("from fixture_helper import VALUE\n")
            # An unrelated caller directory must not supply the helper instead.
            (root / "fixture_helper.py").write_text("raise RuntimeError('wrong checkout')\n")
            script = Path(__file__).with_name("qualify-publication-client.py").resolve()
            code = """
import importlib.util, sys
spec = importlib.util.spec_from_file_location('qualification', sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
before = sys.path[:]
assert module.load_fixture(sys.argv[2]).VALUE == 'selected fixture'
assert sys.path == before
"""
            subprocess.run([sys.executable, "-I", "-B", "-c", code, str(script), str(root / "platform")],
                           cwd=root, check=True, timeout=10)

    def test_executed_recovery_cases_are_required(self):
        test = "TestPublicationPlatformApplicationRecovery"
        names = [test] + [test + "/" + phase for phase in ("before_accept", "after_accept", "after_complete")]
        complete = [{"Action": "pass", "Test": name} for name in names]
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "events.jsonl"
            def write(events):
                path.write_text("".join(json.dumps(event) + "\n" for event in events))
            write(complete)
            qualification.check_test_events(path, test, "create")
            for invalid in [[], complete[:-1], complete + [complete[0]], complete + [{"Action": "skip", "Test": "other"}], complete + [{"Action": "fail"}]]:
                with self.subTest(events=invalid):
                    write(invalid)
                    with self.assertRaises(RuntimeError):
                        qualification.check_test_events(path, test, "create")


if __name__ == "__main__":
    unittest.main()
