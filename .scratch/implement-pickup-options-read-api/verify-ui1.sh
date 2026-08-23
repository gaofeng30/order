#!/usr/bin/env bash
set -euo pipefail

pickup_ui_repo=$(git rev-parse --show-toplevel)
pickup_ui_asset_root=/Users/vivix/.codex/worktrees/order-delivery-integration/tools/miniprogram-ui
pickup_ui_link="${pickup_ui_repo}/tools/miniprogram-ui/node_modules"
pickup_ui_link_target="${pickup_ui_asset_root}/node_modules"
pickup_ui_link_owned=NO

cleanup_pickup_ui() {
  if [[ "${pickup_ui_link_owned}" == YES && -L "${pickup_ui_link}" && $(readlink "${pickup_ui_link}") == "${pickup_ui_link_target}" ]]; then
    unlink "${pickup_ui_link}"
  fi
}
trap cleanup_pickup_ui EXIT

if [[ ! -d "${pickup_ui_asset_root}/node_modules" ]]; then
  printf 'locked UI1 runtime asset is missing\n' >&2
  exit 90
fi
if [[ -e "${pickup_ui_link}" || -L "${pickup_ui_link}" ]]; then
  printf 'current UI1 node_modules target is not absent\n' >&2
  exit 91
fi

for pickup_ui_manifest in package.json package-lock.json; do
  pickup_ui_current_hash=$(shasum -a 256 "${pickup_ui_repo}/tools/miniprogram-ui/${pickup_ui_manifest}" | awk '{print $1}')
  pickup_ui_asset_hash=$(shasum -a 256 "${pickup_ui_asset_root}/${pickup_ui_manifest}" | awk '{print $1}')
  if [[ "${pickup_ui_current_hash}" != "${pickup_ui_asset_hash}" ]]; then
    printf 'UI1 runtime manifest hash mismatch: %s\n' "${pickup_ui_manifest}" >&2
    exit 92
  fi
done
printf 'UI1_RUNTIME_ASSET=LOCK_MATCH source=%s\n' "${pickup_ui_asset_root}/node_modules"

ln -s "${pickup_ui_link_target}" "${pickup_ui_link}"
pickup_ui_link_owned=YES
npm --prefix "${pickup_ui_repo}/tools/miniprogram-ui" run ui1

if [[ ! -L "${pickup_ui_link}" || $(readlink "${pickup_ui_link}") != "${pickup_ui_link_target}" ]]; then
  printf 'UI1 runtime symlink ownership changed during run\n' >&2
  exit 93
fi
cleanup_pickup_ui
pickup_ui_link_owned=NO
if [[ -e "${pickup_ui_link}" || -L "${pickup_ui_link}" ]]; then
  printf 'UI1 runtime symlink cleanup failed\n' >&2
  exit 93
fi
printf 'UI1_RUNTIME_CLEANUP=PASS source=%s\n' "$(git -C "${pickup_ui_repo}" rev-parse HEAD)"
