#!/usr/bin/env python3
"""Explicit local Go HTTP client qualification using Platform's owned fixture.

This never runs Terraform or qualifies installed static resources/cloud delivery.
Both source checkouts must be clean. All native setup stays in Platform's fixture.
"""
import argparse
from datetime import datetime, timezone
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]


def output(args, cwd, timeout=120):
    return subprocess.check_output(args, cwd=cwd, text=True, timeout=timeout).strip()


def source(path):
    if output(["git", "status", "--porcelain"], path):
        raise RuntimeError("Commit qualification sources before executing the fixture")
    return output(["git", "rev-parse", "HEAD"], path)


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def environment():
    # Do not inherit service credentials, Terraform overrides or fixture controls.
    return {key: os.environ[key] for key in ("PATH", "HOME", "USER", "TMPDIR", "SSL_CERT_FILE", "NIX_SSL_CERT_FILE") if key in os.environ}


def load_fixture(platform):
    # The selected Platform fixture imports its own sibling recovery helpers.
    # Resolve them from that checkout, never the caller's working directory or
    # an ambient PYTHONPATH. Native dependencies remain in the locked shell.
    tests = Path(platform) / "tests"
    previous = sys.path[:]
    sys.path.insert(0, str(tests))
    try:
        spec = importlib.util.spec_from_file_location("publication_fixture", tests / "publication-http.integration.py")
        fixture = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(fixture)
        return fixture
    finally:
        sys.path[:] = previous


def bounded(args, cwd, env, timeout):
    process = subprocess.Popen(args, cwd=cwd, env=env, start_new_session=True)
    try:
        if process.wait(timeout=timeout):
            raise RuntimeError("Publication client qualification command failed")
    except BaseException:
        try:
            os.killpg(process.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
        process.wait()
        raise


def worker(binary, evidence):
    # This function runs inside Platform's locked publication-http shell. It
    # imports the pinned fixture, never a provider-owned native implementation.
    env = dict(os.environ, PWD=str(Path.cwd()))
    native = subprocess.check_output([
        "nix", "eval", "--impure", "--raw", "--expr",
        '(builtins.getFlake ("git+file://" + builtins.getEnv "PWD")).inputs.transit-upstream.outPath',
    ], env=env, text=True, timeout=120).strip()
    sys.path.insert(0, str(Path(native) / "tests"))
    package = output(["nix", "build", "--no-update-lock-file", "--no-link", "--print-out-paths", ".#publication-http"], Path.cwd(), timeout=900)
    placement = output(["nix", "build", "--no-update-lock-file", "--no-link", "--print-out-paths", ".#publication-tools"], Path.cwd(), timeout=900)
    os.environ["PUBLICATION_OPERATOR"] = str(Path(placement) / "bin/publication-operator")
    fixture = load_fixture(Path.cwd())
    started = time.monotonic()
    node = shutil.which("transit-node")
    if not node:
        raise RuntimeError("Missing exact native fixture executable")
    with fixture.Fixture(node, package) as f:
        token_paths = {}
        for name, principal in {"project": 201, "nightly": 202, "reader": 205, "foreign": 206}.items():
            path = f.root / (name + ".token")
            path.write_text(f.tokens[principal])
            path.chmod(0o600)
            token_paths[name] = str(path)
        barrier = f.root / "go-client-accepted"
        config = dict(endpoint=f"https://localhost:{f.tls_port}", ca_file=str(f.root / "peer.pem"), tokens=token_paths,
                      barrier=str(barrier), expected_operation="")
        config_path = f.root / "go-client.json"
        config_path.write_text(json.dumps(config))
        config_path.chmod(0o600)
        client_env = environment()
        client_env["SPOKE_PUBLICATION_TEST_CONFIG"] = str(config_path)
        def run(name):
            subprocess.run([binary, "-test.v", "-test.timeout=150s", "-test.run=^" + name + "$"], env=client_env, check=True, timeout=160)
        interrupted = subprocess.Popen([binary, "-test.v", "-test.timeout=150s", "-test.run=^TestPublicationPlatformInterruptedCaller$"], env=client_env)
        try:
            deadline = time.monotonic() + 60
            while not barrier.exists():
                if interrupted.poll() is not None or time.monotonic() > deadline:
                    raise RuntimeError("Go caller did not reach durable acceptance before process death")
                time.sleep(.05)
            original = f.get("/operation-keys/client_crash_create", kind="Operation")
            if original["id"] != barrier.read_text():
                raise RuntimeError("Native receipt did not match caller acceptance")
        finally:
            interrupted.kill()
            interrupted.wait()
        if interrupted.returncode != -signal.SIGKILL:
            raise RuntimeError("Independent Go caller was not killed")
        config["expected_operation"] = original["id"]
        config_path.write_text(json.dumps(config))
        run("TestPublicationPlatformRecoverOriginalCaller")
        run("TestPublicationPlatformDeclarationLifecycle")
        report = dict(profile="spoke.platform.http.publication.v1", evidence_kind="local-publication-http-client",
                      result="passed", elapsed_seconds=round(time.monotonic() - started, 3),
                      original_operation_id=original["id"], caller_sigkill=True,
                      cases=["declaration_lifecycle_and_exact_replay", "protected_and_dependency_deletion", "cross_environment_and_reader_denial",
                             "external_drift_and_stale_revision", "complete_paginated_history", "original_key_recovery_after_caller_death"],
                      binaries={"go_client_test": digest(binary), "platform_publication_http": digest(Path(package) / "bin/platform-publication-http"),
                                "native_node": digest(node), "publication_operator": digest(Path(placement) / "bin/publication-operator")},
                      unexecuted=["installed Terraform static resources", "object upload and sealing", "hosted delivery and domains", "Registry release"])
        Path(evidence).write_text(json.dumps(report, indent=2) + "\n")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--platform", type=Path)
    parser.add_argument("--report", type=Path)
    parser.add_argument("--worker", nargs=2, metavar=("TEST_BINARY", "EVIDENCE"), help=argparse.SUPPRESS)
    args = parser.parse_args()
    if args.worker:
        worker(*args.worker)
        return
    if not args.platform or not args.report or not args.platform.is_absolute() or not args.report.is_absolute():
        parser.error("--platform and --report must be explicit absolute paths")
    if args.report.exists() or str(args.report.resolve()).startswith("/nix/store/"):
        parser.error("--report must be a new external evidence file")
    platform = args.platform.resolve()
    revisions = {"provider_revision": source(ROOT), "platform_revision": source(platform)}
    pin = json.loads((ROOT / "internal/platformclient/publication-contract/profile.json").read_text())
    if digest(platform / "contracts/http/publication-v1.schema.json") != pin["schema_sha256"]:
        raise RuntimeError("Platform schema differs from this client's contract pin")
    directory = Path(tempfile.mkdtemp(prefix="spoke-publication-client-", dir="/tmp"))
    binary, evidence = directory / "client.test", directory / "evidence.json"
    bounded(["nix", "develop", "--no-update-lock-file", "--command", "go", "test", "-race", "-c", "-o", str(binary), "./internal/platformclient"], ROOT, environment(), 900)
    bounded(["nix", "develop", ".#publication-http", "--no-update-lock-file", "--command", "python3", str(Path(__file__).resolve()), "--worker", str(binary), str(evidence)], platform, environment(), 1800)
    if revisions != {"provider_revision": source(ROOT), "platform_revision": source(platform)}:
        raise RuntimeError("Qualification sources changed during execution")
    report = json.loads(evidence.read_text())
    report.update(revisions, contract=pin, checked_at=datetime.now(timezone.utc).isoformat())
    with args.report.open("x") as f:
        f.write(json.dumps(report, indent=2) + "\n")
    args.report.chmod(0o600)
    print(f"Passed local publication HTTP client qualification; report: {args.report}")


if __name__ == "__main__":
    main()
