"""Negative distribution gates; no network, signing key or GitHub writes."""
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
        self.info = {"version": self.version, "revision": self.revision, "platform": "linux_amd64", "contract": "mock-v5"}
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

    def test_initial_graph_cannot_qualify_later_families(self):
        import re
        manifest = json.loads((release.ROOT / "internal/testutil/liveplatform/cases.json").read_text())
        source = (release.ROOT / "internal/nativecontract/codec.go").read_text()
        evidence = {"status": "passed", "gate": "service-graph", "profile": "spoke.management.native.v2", "blockers": [], "artifact_sha256": release.sha(self.binary.read_bytes())}
        for field, constant in [("platform_revision", "PlatformRevision"), ("transit_revision", "TransitRevision")]:
            evidence[field] = re.search(r'const ' + constant + r' = "([0-9a-f]{40})"', source).group(1)
        evidence["cases"] = [{"name": name, "status": "passed"} for family in manifest.values() for name in family]
        report = self.root / "complete.json"
        report.write_text(json.dumps(evidence))
        with self.assertRaises(ValueError): release.publication_gate(self.mirror, report)
        self.info["contract"] = "spoke.management.native.v2"
        release.publication_gate(self.mirror, report)
        evidence["cases"] = evidence["cases"][:10]
        report.write_text(json.dumps(evidence))
        with self.assertRaises(ValueError): release.publication_gate(self.mirror, report)

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
