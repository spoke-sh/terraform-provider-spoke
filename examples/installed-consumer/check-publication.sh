#!/usr/bin/env bash
set -euo pipefail
# Explicit, already bootstrapped normal publication API; values are file paths,
# never credentials. The qualification runner supplies its isolated fixture.
[[ $# == 1 && "$1" == /* && -f "$1" ]] || { echo 'Expected absolute fixture configuration path' >&2; exit 1; }
for tool in go zig cc gcc clang; do
  if command -v "$tool" >/dev/null; then echo "Unexpected compiler: $tool" >&2; exit 1; fi
done
[[ -z ${TF_REATTACH_PROVIDERS:-} && -z ${TF_CLI_ARGS:-} && -z ${TF_PLUGIN_CACHE_DIR:-} ]] || exit 1
config="$1"
export TF_VAR_endpoint="$(jq -er .endpoint "$config")"
export TF_VAR_ca_file="$(jq -er .ca_file "$config")"
export TF_VAR_protected=true
select_root() {
  root="publication/$1"
  role="$1"
  [[ $role != catalog ]] || role=project
  export TF_VAR_token_file="$(jq -er --arg role "$role" '.tokens[$role]' "$config")"
  export TF_VAR_name="consumer-$1"
  if [[ $1 != catalog ]]; then
    export TF_VAR_binding_id="$(terraform -chdir=publication/catalog output -raw "${1}_binding")"
  fi
}
tf() { terraform -chdir="$root" "$@" -input=false -no-color; }
for name in catalog nightly production; do
  select_root "$name"
  [[ ! -e "$root/terraform.tfstate" && ! -d "$root/.terraform" ]] || { echo 'Use a fresh consumer copy' >&2; exit 1; }
  tf init -backend=false -lockfile=readonly
  terraform -chdir="$root" validate -no-color
  tf apply -auto-approve
  tf plan -detailed-exitcode
done
for name in nightly production; do
  select_root "$name"
  id="$(terraform -chdir="$root" output -raw target_import_id)"
  terraform -chdir="$root" state rm spoke_static_target.site
  terraform -chdir="$root" import -input=false -no-color spoke_static_target.site "$id"
  tf plan -detailed-exitcode
done
select_root nightly
export TF_VAR_name=consumer-nightly-updated
tf apply -auto-approve
tf plan -detailed-exitcode
select_root production
tf plan -detailed-exitcode
if tf destroy -auto-approve; then echo 'Protected target was deleted' >&2; exit 1; fi
for name in nightly production; do
  select_root "$name"
  export TF_VAR_protected=false
  tf apply -auto-approve
  tf destroy -auto-approve
done
select_root catalog
tf apply -auto-approve
tf destroy -auto-approve
printf 'Independent publication consumer passed: catalog, nightly and production\n'
