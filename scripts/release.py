#!/usr/bin/env python3
"""Stage and verify the exact Nix mirror bytes; never rebuild inside a release."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys
import zipfile

ROOT = Path(__file__).resolve().parents[1]


def sha(data):
    return hashlib.sha256(data).hexdigest()


def git(*args):
    return subprocess.check_output(["git", "-C", str(ROOT), *args], text=True).strip()


def contract_identity():
    contract = json.loads((ROOT / "internal/platformclient/contract.json").read_text())
    if set(contract) != {"version", "transport", "status", "schema_sha256", "platform_revision"} or contract["transport"] != "https-json":
        raise ValueError("invalid Platform HTTP contract identity")
    if not re.fullmatch(r"[a-z][a-z0-9._-]{1,127}", contract["version"]) or not re.fullmatch(r"[0-9a-f]{64}", contract["schema_sha256"]):
        raise ValueError("invalid HTTP contract version or schema digest")
    if contract["status"] == "published":
        if not re.fullmatch(r"[0-9a-f]{40}", contract["platform_revision"]):
            raise ValueError("published HTTP contract requires a Platform revision")
    elif contract["status"] != "provisional" or contract["platform_revision"]:
        raise ValueError("invalid HTTP contract publication status")
    return contract


def validate_contract_identity(artifact, contract):
    fields = {"contract": "version", "transport": "transport", "contract_status": "status", "contract_schema_sha256": "schema_sha256", "platform_revision": "platform_revision"}
    if any(artifact.get(field) != contract[key] for field, key in fields.items()):
        raise ValueError("artifact does not match the selected Platform HTTP contract")


def identity(mirror):
    binaries = list(mirror.glob("registry.terraform.io/spoke-sh/spoke/*/linux_amd64/terraform-provider-spoke_v*"))
    if len(binaries) != 1:
        raise ValueError("expected exactly one Linux amd64 provider")
    binary = binaries[0]
    value = json.loads(subprocess.check_output([str(binary), "-version"], timeout=10))
    if value["platform"] != "linux_amd64" or not re.fullmatch(r"[0-9a-f]{40}", value["revision"]):
        raise ValueError("unqualified platform or dirty/unknown source identity")
    version = value["version"]
    if version != (ROOT / "VERSION").read_text().strip() + "-dev.g" + value["revision"]:
        raise ValueError("artifact does not follow immutable development version policy")
    if binary.name != "terraform-provider-spoke_v" + version or binary.parent.parent.name != version:
        raise ValueError("mirror filename/version mismatch")
    if sorted(p.name for p in binary.parent.iterdir()) != [binary.name]:
        raise ValueError("archive and mirror must contain the same complete payload")
    validate_contract_identity(value, contract_identity())
    return binary, value


def build(mirror, output):
    if git("status", "--porcelain"):
        raise ValueError("release construction requires a clean source checkout")
    binary, value = identity(mirror)
    if value["revision"] != git("rev-parse", "HEAD"):
        raise ValueError("artifact is not from the selected source commit")
    output.mkdir(parents=True, exist_ok=False)
    prefix = "terraform-provider-spoke_" + value["version"]
    payload = binary.read_bytes()
    archive = output / (prefix + "_linux_amd64.zip")
    entry = zipfile.ZipInfo(binary.name, (1980, 1, 1, 0, 0, 0))
    entry.create_system = 3
    entry.external_attr = 0o100755 << 16
    entry.compress_type = zipfile.ZIP_DEFLATED
    with zipfile.ZipFile(archive, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as z:
        z.writestr(entry, payload)
    (output / (prefix + "_manifest.json")).write_bytes((ROOT / "terraform-registry-manifest.json").read_bytes())
    (output / (prefix + "_LICENSE.txt")).write_bytes((ROOT / "LICENSE").read_bytes())
    provenance = {
        "identity": value,
        "binary_sha256": sha(payload),
        "mirror": str(mirror),
        "source": "https://github.com/spoke-sh/terraform-provider-spoke/commit/" + value["revision"],
        "inputs": {name: sha((ROOT / name).read_bytes()) for name in ["flake.lock", "go.mod", "go.sum", "nix/provider.nix", "internal/platformclient/contract.json"]},
        "qualification": "candidate-only; requires exact-artifact live gate before publication",
    }
    (output / (prefix + "_provenance.json")).write_text(json.dumps(provenance, indent=2, sort_keys=True) + "\n")
    checksums = "".join(f"{sha(p.read_bytes())}  {p.name}\n" for p in sorted(output.iterdir()))
    (output / (prefix + "_SHA256SUMS")).write_text(checksums)
    verify(mirror, output)


def verify(mirror, output):
    binary, value = identity(mirror)
    prefix = "terraform-provider-spoke_" + value["version"]
    expected = {prefix + suffix for suffix in ["_linux_amd64.zip", "_manifest.json", "_LICENSE.txt", "_provenance.json"]}
    sums_name = prefix + "_SHA256SUMS"
    actual = {p.name for p in output.iterdir()}
    if actual - (expected | {sums_name, sums_name + ".sig"}) or not (expected | {sums_name}) <= actual:
        raise ValueError("release asset inventory mismatch")
    if any(not p.is_file() or p.is_symlink() for p in output.iterdir()):
        raise ValueError("release assets must be regular files")
    seen = set()
    for line in (output / sums_name).read_text().splitlines():
        digest, name = line.split("  ", 1)
        if name not in expected or name in seen or digest != sha((output / name).read_bytes()):
            raise ValueError("release checksum mismatch")
        seen.add(name)
    if seen != expected:
        raise ValueError("incomplete checksums")
    manifest = json.loads((output / (prefix + "_manifest.json")).read_text())
    if manifest != {"version": 1, "metadata": {"protocol_versions": ["6.0"]}}:
        raise ValueError("invalid protocol manifest")
    with zipfile.ZipFile(output / (prefix + "_linux_amd64.zip")) as archive:
        if archive.namelist() != [binary.name] or archive.read(binary.name) != binary.read_bytes():
            raise ValueError("archive differs from exact mirror payload")
    provenance = json.loads((output / (prefix + "_provenance.json")).read_text())
    if provenance["identity"] != value or provenance["binary_sha256"] != sha(binary.read_bytes()):
        raise ValueError("provenance identity mismatch")


def publication_gate(mirror, report):
    binary, artifact = identity(mirror)
    contract = contract_identity()
    validate_contract_identity(artifact, contract)
    if contract["status"] != "published":
        raise ValueError("publication requires a published Platform HTTP contract; provisional artifacts are experimental")
    evidence = json.loads(report.read_text())
    manifest = json.loads((ROOT / "internal/testutil/liveplatform/cases.json").read_text())
    required = {case for family in manifest.values() for case in family}
    if not isinstance(evidence, dict) or set(evidence) != {"transport", "api_version", "api_schema_sha256", "platform_revision", "gate", "status", "blockers", "artifact_sha256", "cases"}:
        raise ValueError("invalid HTTP qualification report schema")
    if evidence["status"] != "passed" or evidence["gate"] != "service-graph" or evidence["blockers"] != [] or evidence["artifact_sha256"] != sha(binary.read_bytes()):
        raise ValueError("publication requires full live evidence for these exact bytes")
    for field, key in [("transport", "transport"), ("api_version", "version"), ("api_schema_sha256", "schema_sha256"), ("platform_revision", "platform_revision")]:
        if evidence[field] != contract[key]:
            raise ValueError("live evidence HTTP contract or Platform revision mismatch")
    cases = evidence["cases"]
    if not isinstance(cases, list) or any(not isinstance(c, dict) or set(c) != {"name", "status"} or not isinstance(c["name"], str) for c in cases):
        raise ValueError("invalid live case schema")
    if len(cases) != len(required) or {c["name"] for c in cases} != required or any(c["status"] != "passed" for c in cases):
        raise ValueError("publication cannot accept missing/skipped/duplicate cases")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("operation", choices=["build", "verify", "gate"])
    parser.add_argument("--mirror", type=Path, required=True)
    parser.add_argument("--output", type=Path)
    parser.add_argument("--report", type=Path)
    args = parser.parse_args()
    if args.operation == "gate":
        if not args.report: parser.error("--report is required")
        publication_gate(args.mirror, args.report)
    else:
        if not args.output: parser.error("--output is required")
        (build if args.operation == "build" else verify)(args.mirror, args.output)


if __name__ == "__main__":
    try:
        main()
    except (ValueError, KeyError, OSError, subprocess.SubprocessError, zipfile.BadZipFile) as error:
        print(f"release refused: {error}", file=sys.stderr)
        sys.exit(1)
