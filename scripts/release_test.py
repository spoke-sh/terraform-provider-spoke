"""Negative distribution gates; no network, signing key or GitHub writes."""
import copy
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import zipfile

import release

spec = importlib.util.spec_from_file_location("publish", Path(__file__).with_name("publish-release.py"))
publish = importlib.util.module_from_spec(spec)
spec.loader.exec_module(publish)


class ReleaseTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.mirror = self.root / "mirror"
        self.output = self.root / "release"
        self.revision = "a" * 40
        self.version = (release.ROOT / "VERSION").read_text().strip() + "-dev.g" + self.revision
        self.binary = self.mirror / "registry.terraform.io/spoke-sh/spoke" / self.version / "linux_amd64" / ("terraform-provider-spoke_v" + self.version)
        self.binary.parent.mkdir(parents=True)
        self.contract = release.contract_identity()
        self.info = {"version": self.version, "revision": self.revision, "platform": "linux_amd64", "contract": self.contract["version"], "transport": self.contract["transport"], "contract_status": self.contract["status"], "contract_schema_sha256": self.contract["schema_sha256"], "platform_revision": self.contract["platform_revision"]}
        self.binary.write_bytes(b"synthetic artifact")
        self.prefix = "terraform-provider-spoke_" + self.version
        self.identity = patch.object(release, "identity", return_value=(self.binary, self.info))
        self.identity.start()
        self.addCleanup(self.identity.stop)
        with patch.object(release, "git", side_effect=lambda *args: "" if args[0] == "status" else self.revision):
            release.build(self.mirror, self.output)

    def test_exact_bytes_and_determinism(self):
        with zipfile.ZipFile(self.output / (self.prefix + "_linux_amd64.zip")) as archive:
            self.assertEqual(archive.read(self.binary.name), self.binary.read_bytes())
        other = self.root / "second"
        with patch.object(release, "git", side_effect=lambda *args: "" if args[0] == "status" else self.revision):
            release.build(self.mirror, other)
        for file in self.output.iterdir():
            self.assertEqual(file.read_bytes(), (other / file.name).read_bytes())

    def test_tamper_and_interrupted_download(self):
        archive = self.output / (self.prefix + "_linux_amd64.zip")
        original = archive.read_bytes()
        for bad in [original[:-10], original + b"tampered"]:
            archive.write_bytes(bad)
            with self.assertRaises(ValueError): release.verify(self.mirror, self.output)
        archive.write_bytes(original)
        (self.output / "unadvertised.zip").write_bytes(b"other platform")
        with self.assertRaises(ValueError): release.verify(self.mirror, self.output)

    def test_dirty_source_and_unqualified_live_are_refused(self):
        with patch.object(release, "git", return_value="dirty"):
            with self.assertRaises(ValueError): release.build(self.mirror, self.root / "dirty")
        report = self.root / "report.json"
        for value in [{}, {"status":"blocked"}, {"status":"passed", "gate":"metadata"}, {"status":"passed", "gate":"service-graph", "profile":"spoke.management.native.v2", "artifact_sha256": release.sha(self.binary.read_bytes()), "cases":[]}]:
            report.write_text(json.dumps(value))
            with self.assertRaises(ValueError): release.publication_gate(self.mirror, report)

    def test_http_publication_requires_published_contract_and_all_families(self):
        manifest = json.loads((release.ROOT / "internal/testutil/liveplatform/cases.json").read_text())
        evidence = {
            "status": "passed", "gate": "service-graph", "transport": "https-json",
            "api_version": self.contract["version"], "api_schema_sha256": self.contract["schema_sha256"],
            "platform_revision": self.contract["platform_revision"], "blockers": [],
            "artifact_sha256": release.sha(self.binary.read_bytes()),
            "cases": [{"name": name, "status": "passed"} for family in manifest.values() for name in family],
        }
        report = self.root / "complete.json"
        report.write_text(json.dumps(evidence))
        # A fabricated complete report cannot make the actual provisional source
        # contract publishable. Merely relabelling artifact metadata also fails.
        with self.assertRaises(ValueError): release.publication_gate(self.mirror, report)
        self.info["contract_status"] = "published"
        with self.assertRaises(ValueError): release.publication_gate(self.mirror, report)
        # Exercise the full gate with an explicitly synthetic future handoff,
        # independent of the real source's permanently provisional test schema.
        published = {**self.contract, "version": "synthetic.http.v1", "status": "published", "platform_revision": "b" * 40}
        self.info.update(contract=published["version"], platform_revision=published["platform_revision"])
        evidence.update(api_version=published["version"], platform_revision=published["platform_revision"])
        report.write_text(json.dumps(evidence))
        with patch.object(release, "contract_identity", return_value=published):
            release.publication_gate(self.mirror, report)
            bad_values = [
                {**evidence, "cases": evidence["cases"][:10]},
                {**evidence, "cases": []},
                {**evidence, "cases": "invalid"},
                {**evidence, "gate": "metadata"},
                {**evidence, "status": "blocked"},
                {**evidence, "blockers": ["missing_backend"]},
                {**evidence, "blockers": None},
                {**evidence, "transport": "native"},
                {**evidence, "api_version": "other"},
                {**evidence, "api_schema_sha256": "c" * 64},
                {**evidence, "platform_revision": "c" * 40},
                {**evidence, "artifact_sha256": "c" * 64},
                {**evidence, "transit_revision": "c" * 40},
            ]
            skipped, duplicate = copy.deepcopy(evidence), copy.deepcopy(evidence)
            skipped["cases"][0]["status"] = "skipped"
            duplicate["cases"][1] = duplicate["cases"][0]
            for bad in bad_values + [skipped, duplicate]:
                with self.subTest(evidence=bad):
                    report.write_text(json.dumps(bad))
                    with self.assertRaises(ValueError): release.publication_gate(self.mirror, report)

    def test_http_artifact_identity_cannot_be_substituted(self):
        release.validate_contract_identity(self.info, self.contract)
        for field in ["contract", "contract_status", "transport", "contract_schema_sha256", "platform_revision"]:
            wrong = {**self.info, field: "other"}
            with self.subTest(field=field):
                with self.assertRaises(ValueError): release.validate_contract_identity(wrong, self.contract)

    def test_existing_changed_asset_never_overwritten(self):
        sums = self.prefix + "_SHA256SUMS"
        (self.output / (sums + ".sig")).write_bytes(b"synthetic signature")
        response = type("Response", (), {"returncode":0, "stdout":json.dumps({"tagName":"v"+self.version,"assets":[{"name":sums}]})})()
        calls = []
        def fake_gh(*args, **kwargs):
            calls.append(args)
            self.assertEqual(args[:2], ("release", "download"))
            (Path(args[args.index("--dir")+1]) / sums).write_bytes(b"different published bytes")
        with patch.object(publish.subprocess, "run", return_value=response), patch.object(publish, "gh", side_effect=fake_gh):
            with self.assertRaises(ValueError): publish.publish(self.output,"v"+self.version)
        self.assertEqual(len(calls),1)


if __name__ == "__main__":
    unittest.main()
