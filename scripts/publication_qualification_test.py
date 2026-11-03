"""The installed lane must reject missing or skipped execution, not just exit 0."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location("qualification", Path(__file__).with_name("qualify-publication-terraform.py"))
qualification = importlib.util.module_from_spec(spec)
spec.loader.exec_module(qualification)


class QualificationTests(unittest.TestCase):
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
