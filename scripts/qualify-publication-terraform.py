#!/usr/bin/env python3
"""Explicit publication adapter qualification using Platform's installed fixture.

Identity-only mode qualifies the provider's recovery construction through HTTP;
it does not qualify Terraform resources or hosted delivery. Both sources must be
clean. The fixture and all native setup remain owned by Platform.
"""
import argparse
from datetime import datetime, timezone
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location("publication_client_qualification", ROOT / "scripts/qualify-publication-client.py")
common = importlib.util.module_from_spec(spec)
spec.loader.exec_module(common)


def worker(binary, evidence):
    env = dict(os.environ, PWD=str(Path.cwd()))
    native = subprocess.check_output([
        "nix", "eval", "--impure", "--raw", "--expr",
        '(builtins.getFlake ("git+file://" + builtins.getEnv "PWD")).inputs.transit-upstream.outPath',
    ], env=env, text=True, timeout=120).strip()
    sys.path.insert(0, str(Path(native) / "tests"))
    package = common.output(["nix", "build", "--no-update-lock-file", "--no-link", "--print-out-paths", ".#publication-http"], Path.cwd(), timeout=900)
    placement = common.output(["nix", "build", "--no-update-lock-file", "--no-link", "--print-out-paths", ".#publication-tools"], Path.cwd(), timeout=900)
    os.environ["PUBLICATION_OPERATOR"] = str(Path(placement) / "bin/publication-operator")
    fixture_spec = importlib.util.spec_from_file_location("publication_fixture", Path.cwd() / "tests/publication-http.integration.py")
    fixture = importlib.util.module_from_spec(fixture_spec)
    fixture_spec.loader.exec_module(fixture)
    node = shutil.which("transit-node")
    if not node:
        raise RuntimeError("Missing exact native fixture executable")
    started = time.monotonic()
    with fixture.Fixture(node, package) as f:
        tokens = {}
        for name, principal in {"project": 201, "nightly": 202, "production": 203, "reader": 205, "foreign": 206}.items():
            path = f.root / (name + ".token")
            path.write_text(f.tokens[principal])
            path.chmod(0o600)
            tokens[name] = str(path)
        config = f.root / "provider-publication.json"
        config.write_text(json.dumps(dict(endpoint=f"https://localhost:{f.tls_port}", ca_file=str(f.root / "peer.pem"), tokens=tokens)))
        config.chmod(0o600)
        caller_env = common.environment()
        caller_env["SPOKE_PUBLICATION_TERRAFORM_CONFIG"] = str(config)
        subprocess.run([binary, "-test.v", "-test.timeout=150s", "-test.run=^TestPublicationPlatformIdentityProtocol$"], env=caller_env, check=True, timeout=160)
    report = dict(evidence_kind="local-publication-identity-protocol", result="passed", elapsed_seconds=round(time.monotonic() - started, 3),
                  cases=["lost_state_identity_lookup", "exact_create_update_delete_replay", "conflicting_intent_and_foreign_principal", "a_b_a_distinct_operations", "retained_recreation_and_stale_state_fence"],
                  binaries={"provider_test": common.digest(binary), "platform_publication_http": common.digest(Path(package) / "bin/platform-publication-http"),
                            "native_node": common.digest(node), "publication_operator": common.digest(Path(placement) / "bin/publication-operator")},
                  unexecuted=["installed Terraform static resources", "object upload and sealing", "hosted delivery and domains", "Registry release"])
    Path(evidence).write_text(json.dumps(report, indent=2) + "\n")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--platform", type=Path)
    parser.add_argument("--report", type=Path)
    parser.add_argument("--identity-only", action="store_true")
    parser.add_argument("--worker", nargs=2, help=argparse.SUPPRESS)
    args = parser.parse_args()
    if args.worker:
        worker(*args.worker)
        return
    if not args.identity_only or not args.platform or not args.report or not args.platform.is_absolute() or not args.report.is_absolute():
        parser.error("--identity-only, absolute --platform and absolute --report are required")
    if args.report.exists() or str(args.report.resolve()).startswith("/nix/store/"):
        parser.error("--report must be a new external evidence file")
    platform = args.platform.resolve()
    revisions = dict(provider_revision=common.source(ROOT), platform_revision=common.source(platform))
    pin = json.loads((ROOT / "internal/platformclient/publication-contract/profile.json").read_text())
    if common.digest(platform / "contracts/http/publication-v1.schema.json") != pin["schema_sha256"]:
        raise RuntimeError("Platform schema differs from this provider's pin")
    directory = Path(tempfile.mkdtemp(prefix="spoke-publication-terraform-", dir="/tmp"))
    binary, evidence = directory / "provider.test", directory / "evidence.json"
    common.bounded(["nix", "develop", "--no-update-lock-file", "--command", "go", "test", "-race", "-c", "-o", str(binary), "./internal/provider"], ROOT, common.environment(), 900)
    common.bounded(["nix", "develop", ".#publication-http", "--no-update-lock-file", "--command", "python3", str(Path(__file__).resolve()), "--worker", str(binary), str(evidence)], platform, common.environment(), 1800)
    if revisions != dict(provider_revision=common.source(ROOT), platform_revision=common.source(platform)):
        raise RuntimeError("Qualification sources changed during execution")
    report = json.loads(evidence.read_text())
    report.update(revisions, contract=pin, checked_at=datetime.now(timezone.utc).isoformat())
    with args.report.open("x") as f:
        f.write(json.dumps(report, indent=2) + "\n")
    args.report.chmod(0o600)
    print(f"Passed publication identity protocol qualification; report: {args.report}")


if __name__ == "__main__":
    main()
