#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SCRIPT_PATH="$ROOT_DIR/scripts/detect-main-pipeline-changes.sh"

if [ ! -x "$SCRIPT_PATH" ]; then
  chmod +x "$SCRIPT_PATH"
fi

TEST_TMP_BASE="$(mktemp -d)"
trap 'rm -rf "$TEST_TMP_BASE"' EXIT

pass_count=0

create_repo_with_two_commits() {
  local repo_dir="$1"
  shift
  local files=("$@")

  mkdir -p "$repo_dir"
  git -C "$repo_dir" init -q
  git -C "$repo_dir" config user.name "Test User"
  git -C "$repo_dir" config user.email "test@example.com"

  echo "baseline" > "$repo_dir/README.md"
  git -C "$repo_dir" add README.md
  git -C "$repo_dir" commit -q -m "baseline"

  for file in "${files[@]}"; do
    mkdir -p "$repo_dir/$(dirname "$file")"
    echo "change" > "$repo_dir/$file"
  done
  git -C "$repo_dir" add .
  git -C "$repo_dir" commit -q -m "change"
}

read_output_value() {
  local output_file="$1"
  local key="$2"
  grep "^${key}=" "$output_file" | tail -n1 | cut -d= -f2
}

assert_output() {
  local output_file="$1"
  local key="$2"
  local expected="$3"
  local actual
  actual="$(read_output_value "$output_file" "$key")"
  if [ "$actual" != "$expected" ]; then
    echo "Assertion failed for ${key}: expected '${expected}', got '${actual}'" >&2
    exit 1
  fi
}

run_case() {
  local name="$1"
  shift
  local files=("$@")
  local repo_dir="$TEST_TMP_BASE/$name"
  create_repo_with_two_commits "$repo_dir" "${files[@]}"

  local output_file="$repo_dir/output.txt"
  (
    cd "$repo_dir"
    GITHUB_OUTPUT="$output_file" bash "$SCRIPT_PATH" >/dev/null
  )
  echo "$output_file"
}

# Case 1: Backend Go change
case1_output="$(run_case backend_go "backend/internal/service/foo.go")"
assert_output "$case1_output" "backend" "true"
assert_output "$case1_output" "typescript-provider" "false"
assert_output "$case1_output" "frontend" "false"
assert_output "$case1_output" "infrastructure" "false"
pass_count=$((pass_count + 1))

# Case 2: TypeScript provider change
case2_output="$(run_case typescript_provider "backend/providers/provider.ts")"
assert_output "$case2_output" "backend" "false"
assert_output "$case2_output" "typescript-provider" "true"
assert_output "$case2_output" "frontend" "false"
assert_output "$case2_output" "infrastructure" "false"
pass_count=$((pass_count + 1))

# Case 3: Frontend change
case3_output="$(run_case frontend "Frontend/src/App.tsx")"
assert_output "$case3_output" "backend" "false"
assert_output "$case3_output" "typescript-provider" "false"
assert_output "$case3_output" "frontend" "true"
assert_output "$case3_output" "infrastructure" "false"
pass_count=$((pass_count + 1))

# Case 4: Pulumi infra change
case4_output="$(run_case pulumi_infra "infrastructure/pulumi/src/index.ts")"
assert_output "$case4_output" "backend" "false"
assert_output "$case4_output" "typescript-provider" "false"
assert_output "$case4_output" "frontend" "false"
assert_output "$case4_output" "infrastructure" "true"
pass_count=$((pass_count + 1))

echo "All detect-main-pipeline-changes tests passed (${pass_count} cases)."
