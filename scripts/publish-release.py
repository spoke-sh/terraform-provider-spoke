#!/usr/bin/env python3
"""Upload immutable assets, resuming only when existing bytes match exactly."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import tempfile


def gh(*args, **kwargs):
    return subprocess.run(["gh", *args], check=True, **kwargs)


def publish(directory, tag):
    files = {p.name: p for p in directory.iterdir()}
    sums = [p for p in files.values() if p.name.endswith("_SHA256SUMS")]
    if len(sums) != 1 or sums[0].name + ".sig" not in files:
        raise ValueError("complete signed asset set required")
    if tag != "v" + sums[0].name.removeprefix("terraform-provider-spoke_").removesuffix("_SHA256SUMS"):
        raise ValueError("tag and artifact version differ")
    response = subprocess.run(["gh", "release", "view", tag, "--json", "assets,isDraft,tagName"], capture_output=True, text=True)
    if response.returncode:
        # Creation is exclusive. Network/auth errors or a concurrent release fail;
        # they never authorize replacement/deletion of an existing release.
        gh("release", "create", tag, "--verify-tag", "--draft", "--prerelease", "--title", tag, "--notes", "Experimental current-profile provider. See attached provenance and repository qualification records.")
        existing = []
    else:
        release = json.loads(response.stdout)
        if release["tagName"] != tag: raise ValueError("release identity mismatch")
        existing = release["assets"]
    names = {asset["name"] for asset in existing}
    if not names <= files.keys():
        raise ValueError("existing release has unexpected assets")
    with tempfile.TemporaryDirectory() as temporary:
        for name in sorted(names):
            gh("release", "download", tag, "--pattern", name, "--dir", temporary)
            if hashlib.sha256((Path(temporary) / name).read_bytes()).digest() != hashlib.sha256(files[name].read_bytes()).digest():
                raise ValueError("existing release asset differs; refusing overwrite: " + name)
    for name in sorted(files.keys() - names):
        gh("release", "upload", tag, str(files[name])) # Deliberately no --clobber.
    gh("release", "edit", tag, "--draft=false", "--prerelease")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--directory", type=Path, required=True)
    parser.add_argument("--tag", required=True)
    args = parser.parse_args()
    publish(args.directory, args.tag)
