#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SCRIPT_PATH="$ROOT_DIR/scripts/detect-deployment-targets.sh"

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

run_auto_case() {
  local name="$1"
  shift
  local files=("$@")
  local repo_dir="$TEST_TMP_BASE/$name"
  create_repo_with_two_commits "$repo_dir" "${files[@]}"

  local output_file="$repo_dir/output.txt"
  (
    cd "$repo_dir"
    GITHUB_OUTPUT="$output_file" bash "$SCRIPT_PATH" auto >/dev/null
  )
  echo "$output_file"
}

run_manual_case() {
  local name="$1"
  local mode="$2"
  local repo_dir="$TEST_TMP_BASE/$name"
  mkdir -p "$repo_dir"
  git -C "$repo_dir" init -q
  git -C "$repo_dir" config user.name "Test User"
  git -C "$repo_dir" config user.email "test@example.com"
  echo "baseline" > "$repo_dir/README.md"
  git -C "$repo_dir" add README.md
  git -C "$repo_dir" commit -q -m "baseline"

  local output_file="$repo_dir/output.txt"
  (
    cd "$repo_dir"
    GITHUB_OUTPUT="$output_file" bash "$SCRIPT_PATH" "$mode" >/dev/null
  )
  echo "$output_file"
}

# Case 1: Pulumi precedence should force infra-only
case1_output="$(run_auto_case pulumi_precedence \
  "infrastructure/pulumi/src/index.ts" \
  "backend/internal/service/foo.go" \
  "Frontend/src/App.tsx")"
assert_output "$case1_output" "deploy-infrastructure" "true"
assert_output "$case1_output" "deploy-backend" "false"
assert_output "$case1_output" "deploy-frontend" "false"
pass_count=$((pass_count + 1))

# Case 2: Backend-only change
case2_output="$(run_auto_case backend_only "backend/internal/service/foo.go")"
assert_output "$case2_output" "deploy-infrastructure" "false"
assert_output "$case2_output" "deploy-backend" "true"
assert_output "$case2_output" "deploy-frontend" "false"
pass_count=$((pass_count + 1))

# Case 3: Frontend-only change through root package file
case3_output="$(run_auto_case frontend_root_only "package.json")"
assert_output "$case3_output" "deploy-infrastructure" "false"
assert_output "$case3_output" "deploy-backend" "false"
assert_output "$case3_output" "deploy-frontend" "true"
pass_count=$((pass_count + 1))

# Case 4: Non-matching change
case4_output="$(run_auto_case no_match "docs/README.md")"
assert_output "$case4_output" "deploy-infrastructure" "false"
assert_output "$case4_output" "deploy-backend" "false"
assert_output "$case4_output" "deploy-frontend" "false"
pass_count=$((pass_count + 1))

# Case 5: Manual full
case5_output="$(run_manual_case manual_full "full")"
assert_output "$case5_output" "deploy-infrastructure" "true"
assert_output "$case5_output" "deploy-backend" "true"
assert_output "$case5_output" "deploy-frontend" "true"
pass_count=$((pass_count + 1))

# Case 6: Invalid mode should fail
invalid_repo="$TEST_TMP_BASE/invalid_mode"
mkdir -p "$invalid_repo"
git -C "$invalid_repo" init -q
git -C "$invalid_repo" config user.name "Test User"
git -C "$invalid_repo" config user.email "test@example.com"
echo "baseline" > "$invalid_repo/README.md"
git -C "$invalid_repo" add README.md
git -C "$invalid_repo" commit -q -m "baseline"
if (
  cd "$invalid_repo"
  GITHUB_OUTPUT="$invalid_repo/output.txt" bash "$SCRIPT_PATH" not-a-mode >/dev/null 2>&1
); then
  echo "Assertion failed: invalid mode should return non-zero" >&2
  exit 1
fi
pass_count=$((pass_count + 1))

echo "All detect-deployment-targets tests passed (${pass_count} cases)."
