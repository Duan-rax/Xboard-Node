#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work_dir="${repo_root}/.xray-audit"
source_dir="${1:-${work_dir}/src}"
mod_file="${work_dir}/go.mod"
patch_file="${repo_root}/tools/patches/xray-access-traffic.patch"
cedar_repo="https://github.com/cedar2025/Xray-core.git"
cedar_commit="f479355399657c0ed07b3e8ac33e02f727094188"

command -v git >/dev/null || { echo "git is required" >&2; exit 1; }
command -v go >/dev/null || { echo "go is required" >&2; exit 1; }

mkdir -p "${work_dir}" "$(dirname "${source_dir}")"
if [[ ! -d "${source_dir}/.git" ]]; then
  if [[ -e "${source_dir}" ]]; then
    echo "refusing to replace non-git path: ${source_dir}" >&2
    exit 1
  fi
  git init -q "${source_dir}"
  git -C "${source_dir}" remote add origin "${cedar_repo}"
fi

git -C "${source_dir}" remote set-url origin "${cedar_repo}"
git -C "${source_dir}" fetch --depth 1 origin "${cedar_commit}"
head_commit="$(git -C "${source_dir}" rev-parse --verify HEAD 2>/dev/null || true)"
if [[ -z "${head_commit}" ]] && git -C "${source_dir}" cat-file -e "${cedar_commit}^{commit}" 2>/dev/null; then
  git -C "${source_dir}" checkout -q --detach "${cedar_commit}"
  head_commit="${cedar_commit}"
fi
if [[ "${head_commit}" != "${cedar_commit}" ]]; then
  echo "${source_dir} is at ${head_commit:-unknown}, expected ${cedar_commit}" >&2
  echo "remove that generated cache directory and rerun this script" >&2
  exit 1
fi

if git -C "${source_dir}" apply --reverse --check "${patch_file}" 2>/dev/null; then
  echo "Xray audit patch already applied"
elif git -C "${source_dir}" apply --check "${patch_file}"; then
  git -C "${source_dir}" apply "${patch_file}"
  echo "Xray audit patch applied"
else
  echo "Xray audit patch does not match ${cedar_commit}" >&2
  exit 1
fi

awk -v replacement="${source_dir}" '
  /^replace github\.com\/xtls\/xray-core =>/ {
    print "replace github.com/xtls/xray-core => " replacement
    next
  }
  { print }
' "${repo_root}/go.mod" >"${mod_file}"
cp "${repo_root}/go.sum" "${work_dir}/go.sum"

# The local Xray module carries its own transitive dependency graph. Refresh
# the alternate module file so Go can verify every package used by the binary.
(
  cd "${repo_root}"
  go mod tidy -modfile="${mod_file}"
)

echo "patched Xray source: ${source_dir}"
echo "Go alternate modfile: ${mod_file}"
