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
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location("publication_client_qualification", ROOT / "scripts/qualify-publication-client.py")
common = importlib.util.module_from_spec(spec)
spec.loader.exec_module(common)


def worker(binary, evidence, options_path):
    options = json.loads(Path(options_path).read_text())
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
    groups = ["identity"] if options["identity_only"] else ["application", "create", "update", "delete", "graph", "aliases", "binding-create", "binding-delete", "target-create", "target-update", "target-delete"]
    cases = []
    for group in groups:
      # Platform's fixture currently hardcodes /tmp. Relocate only its scratch
      # directory when explicitly selected; service binaries/setup stay intact.
      make_directory = tempfile.mkdtemp
      def scratch(*args, **kwargs):
          kwargs["dir"] = os.environ.get("TMPDIR", "/tmp")
          return make_directory(*args, **kwargs)
      with patch.object(fixture.proof.tempfile, "mkdtemp", scratch):
          installation = fixture.Fixture(node, package)
      with installation as f:
        tokens = {}
        for name, principal in {"project": 201, "nightly": 202, "production": 203, "reader": 205, "foreign": 206}.items():
            path = f.root / (name + ".token")
            path.write_text(f.tokens[principal])
            path.chmod(0o600)
            tokens[name] = str(path)
        config = f.root / "provider-publication.json"
        config.write_text(json.dumps(dict(endpoint=f"https://localhost:{f.tls_port}", ca_file=str(f.root / "peer.pem"), tokens=tokens, control_directory=str(f.root))))
        config.chmod(0o600)
        caller_env = common.environment()
        caller_env["SPOKE_PUBLICATION_TERRAFORM_CONFIG"] = str(config)
        if options["identity_only"]:
            test = "TestPublicationPlatformIdentityProtocol"
            cases += ["lost_state_identity_lookup", "exact_create_update_delete_replay", "conflicting_intent_and_foreign_principal", "a_b_a_distinct_operations", "retained_recreation_and_stale_state_fence"]
        else:
            test = "TestPublicationPlatformApplication"
            caller_env.update(SPOKE_TEST_PROVIDER_MIRROR=options["mirror"], SPOKE_TEST_PROVIDER_VERSION=options["version"], TF_ACC_TERRAFORM_PATH=options["terraform"])
            if group == "application":
                cases += ["installed_application_lifecycle_import_empty_plan", "installed_application_a_b_a", "installed_application_protection_recreation", "installed_application_drift_stale_saved_plan", "installed_application_denied_refresh_preserves_state"]
            elif group == "graph":
                test = "TestPublicationPlatformGraph"
                cases += ["installed_catalog_nightly_production_independent_roots", "installed_binding_target_import_policy_drift", "installed_application_holds_and_authoritative_retirement", "installed_scoped_denial_and_observational_discovery"]
            elif group == "aliases":
                test = "TestPublicationPlatformAliases"
                cases += ["installed_provider_alias_scope_and_policy_isolation"]
            elif "-" in group:
                resource, action = group.split("-")
                test = "TestPublicationPlatformChildRecovery"
                caller_env.update(SPOKE_PUBLICATION_RECOVERY_RESOURCE=resource, SPOKE_PUBLICATION_RECOVERY_KIND=action)
                cases += ["installed_" + resource + "_" + action + "_death_" + phase for phase in ["before_accept", "after_accept", "after_complete"]]
            else:
                test = "TestPublicationPlatformApplicationRecovery"
                caller_env["SPOKE_PUBLICATION_RECOVERY_KIND"] = group
                cases += ["installed_application_" + group + "_death_" + phase for phase in ["before_accept", "after_accept", "after_complete"]]
        listed = subprocess.check_output([binary, "-test.list=^" + test + "$"], env=caller_env, text=True, timeout=15).splitlines()
        if listed != [test]:
            raise RuntimeError("Required installed qualification case is missing")
        process = subprocess.Popen([binary, "-test.v", "-test.timeout=600s", "-test.run=^" + test + "$"], env=caller_env)
        deadline = time.monotonic() + 610
        try:
            while process.poll() is None:
                request = f.root / "restart.request"
                if request.exists():
                    if request.read_text() != "restart":
                        raise RuntimeError("Unknown fixture control request")
                    request.unlink()
                    f.restart()
                    (f.root / "restart.complete").write_text("restarted")
                if time.monotonic() > deadline:
                    raise RuntimeError("Installed Terraform qualification deadline")
                time.sleep(.05)
            if process.returncode:
                raise RuntimeError("Installed Terraform qualification failed")
        finally:
            if process.poll() is None:
                process.kill()
            process.wait()
    report = dict(evidence_kind="local-publication-identity-protocol" if options["identity_only"] else "local-publication-terraform-declarations", result="passed", elapsed_seconds=round(time.monotonic() - started, 3),
                  cases=cases,
                  binaries={"provider_test": common.digest(binary), "platform_publication_http": common.digest(Path(package) / "bin/platform-publication-http"),
                            "native_node": common.digest(node), "publication_operator": common.digest(Path(placement) / "bin/publication-operator")},
                  unexecuted=["object upload and sealing", "hosted delivery and domains", "Registry release"])
    if options["identity_only"]:
        report["unexecuted"].append("installed Terraform static resources")
    else:
        report.update(provider_artifact=options["artifact"], terraform_version=options["terraform_version"], system="linux_amd64")
        report["unexecuted"].append("independent committed publication consumer locks")
        report["cases"].append("retained_native_node_owner_gateway_restart_during_installed_recovery")
    Path(evidence).write_text(json.dumps(report, indent=2) + "\n")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--platform", type=Path)
    parser.add_argument("--report", type=Path)
    parser.add_argument("--identity-only", action="store_true")
    parser.add_argument("--mirror", type=Path)
    parser.add_argument("--worker", nargs=3, help=argparse.SUPPRESS)
    args = parser.parse_args()
    if args.worker:
        worker(*args.worker)
        return
    if not args.platform or not args.report or not args.platform.is_absolute() or not args.report.is_absolute():
        parser.error("absolute --platform and absolute --report are required")
    if args.identity_only == bool(args.mirror):
        parser.error("select exactly one of --identity-only or --mirror")
    if args.report.exists() or str(args.report.resolve()).startswith("/nix/store/"):
        parser.error("--report must be a new external evidence file")
    platform = args.platform.resolve()
    revisions = dict(provider_revision=common.source(ROOT), platform_revision=common.source(platform))
    pin = json.loads((ROOT / "internal/platformclient/publication-contract/profile.json").read_text())
    if common.digest(platform / "contracts/http/publication-v1.schema.json") != pin["schema_sha256"]:
        raise RuntimeError("Platform schema differs from this provider's pin")
    directory = Path(tempfile.mkdtemp(prefix="spoke-publication-terraform-", dir=os.environ.get("TMPDIR", "/tmp")))
    binary, evidence = directory / "provider.test", directory / "evidence.json"
    options = dict(identity_only=args.identity_only)
    if args.mirror:
        if not args.mirror.is_absolute():
            parser.error("--mirror must be an absolute immutable artifact path")
        candidates = list(args.mirror.glob("registry.terraform.io/spoke-sh/spoke/*/linux_amd64/terraform-provider-spoke_v*"))
        if len(candidates) != 1:
            raise RuntimeError("Exact Linux amd64 mirror artifact required; no source build fallback")
        artifact = candidates[0]
        identity = json.loads(subprocess.check_output([str(artifact), "-version"], timeout=10))
        if identity["revision"] != revisions["provider_revision"] or identity["platform"] != "linux_amd64" or identity["version"] != artifact.parent.parent.name:
            raise RuntimeError("Installed artifact does not match the clean qualification source")
        terraform = shutil.which("terraform")
        if not terraform:
            raise RuntimeError("Run qualification from the locked provider Nix shell")
        terraform_version = json.loads(subprocess.check_output([terraform, "version", "-json"], timeout=10))["terraform_version"]
        options.update(mirror=str(args.mirror), version=identity["version"], terraform=terraform, terraform_version=terraform_version,
                       artifact=dict(revision=identity["revision"], version=identity["version"], sha256=common.digest(artifact)))
    options_path = directory / "options.json"
    options_path.write_text(json.dumps(options))
    common.bounded(["nix", "develop", "--no-update-lock-file", "--command", "go", "test", "-race", "-c", "-o", str(binary), "./internal/provider"], ROOT, common.environment(), 900)
    common.bounded(["nix", "develop", ".#publication-http", "--no-update-lock-file", "--command", "python3", str(Path(__file__).resolve()), "--worker", str(binary), str(evidence), str(options_path)], platform, common.environment(), 1800)
    if revisions != dict(provider_revision=common.source(ROOT), platform_revision=common.source(platform)):
        raise RuntimeError("Qualification sources changed during execution")
    report = json.loads(evidence.read_text())
    report.update(revisions, contract=pin, checked_at=datetime.now(timezone.utc).isoformat())
    with args.report.open("x") as f:
        f.write(json.dumps(report, indent=2) + "\n")
    args.report.chmod(0o600)
    print(f"Passed {report['evidence_kind']} qualification; report: {args.report}")


if __name__ == "__main__":
    main()
