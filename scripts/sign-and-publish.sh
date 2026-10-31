#!/usr/bin/env bash
set -euo pipefail
: "${RUNNER_TEMP:?}" "${RELEASE_TAG:?}" "${SIGNING_FINGERPRINT:?}" "${GPG_PRIVATE_KEY:?}"
export GNUPGHOME
GNUPGHOME="$(mktemp -d "$RUNNER_TEMP/spoke-gpg.XXXXXX")"
trap 'rm -rf "$GNUPGHOME"' EXIT
chmod 700 "$GNUPGHOME"
printf '%s\n' "$GPG_PRIVATE_KEY" | gpg --batch --import
sums=("$RUNNER_TEMP"/release/*_SHA256SUMS)
[[ ${#sums[@]} == 1 ]] || { echo 'Expected one checksum manifest' >&2; exit 1; }
# Reuse an existing valid signature on a partial-publication retry. A new
# timestamped signature must not replace an already uploaded asset.
signature="$(basename "${sums[0]}").sig"
existing="$(gh release view "$RELEASE_TAG" --json assets --jq '.assets[].name' 2>/dev/null || true)"
if printf '%s\n' "$existing" | grep -Fxq "$signature"; then
  gh release download "$RELEASE_TAG" --pattern "$signature" --dir "$RUNNER_TEMP/release"
else
  printf '%s' "$GPG_PASSPHRASE" | gpg --batch --pinentry-mode loopback --passphrase-fd 0 --local-user "$SIGNING_FINGERPRINT" --detach-sign --output "${sums[0]}.sig" "${sums[0]}"
fi
# gpgv/key selection is anchored to the configured full fingerprint.
[[ "$SIGNING_FINGERPRINT" =~ ^[A-Fa-f0-9]{40}$ ]] || { echo 'Full signing fingerprint required' >&2; exit 1; }
gpg --batch --export "$SIGNING_FINGERPRINT" > "$GNUPGHOME/trusted.gpg"
gpgv --keyring "$GNUPGHOME/trusted.gpg" "${sums[0]}.sig" "${sums[0]}"
python3 scripts/publish-release.py --directory "$RUNNER_TEMP/release" --tag "$RELEASE_TAG"
