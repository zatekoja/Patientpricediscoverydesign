#!/usr/bin/env bash
# detect-main-pipeline-changes.sh - Set file-change outputs for main pipeline CI gating.
#
# Outputs:
#   backend
#   typescript-provider
#   frontend
#   infrastructure

set -euo pipefail

write_output() {
    local key="$1"
    local value="$2"

    if [ -n "${GITHUB_OUTPUT:-}" ]; then
        echo "${key}=${value}" >> "$GITHUB_OUTPUT"
    else
        echo "${key}=${value}"
    fi
}

changed_files() {
    if git rev-parse --verify HEAD~1 >/dev/null 2>&1; then
        git diff --name-only HEAD~1 HEAD
    else
        # Fallback for shallow/single-commit contexts.
        git ls-files
    fi
}

CHANGED_FILES="$(changed_files)"

if echo "$CHANGED_FILES" | grep -q '^backend/.*\.go\|^backend/go\.mod\|^backend/go\.sum\|^backend/cmd/\|^backend/internal/\|^backend/pkg/'; then
    write_output "backend" "true"
    echo "Backend Go changes detected"
else
    write_output "backend" "false"
    echo "No backend Go changes"
fi

if echo "$CHANGED_FILES" | grep -q '^backend/.*\.ts\|^backend/package\.json\|^backend/tsconfig\.json\|^backend/api/\|^backend/providers/\|^backend/types/\|^backend/interfaces/'; then
    write_output "typescript-provider" "true"
    echo "TypeScript provider changes detected"
else
    write_output "typescript-provider" "false"
    echo "No TypeScript provider changes"
fi

if echo "$CHANGED_FILES" | grep -q '^Frontend/\|^package\.json\|^package-lock\.json\|^tsconfig\.json\|^tsconfig\.node\.json\|^vite\.config\|^\.eslintrc\|^\.prettierrc'; then
    write_output "frontend" "true"
    echo "Frontend changes detected"
else
    write_output "frontend" "false"
    echo "No frontend changes"
fi

if echo "$CHANGED_FILES" | grep -q '^infrastructure/pulumi/'; then
    write_output "infrastructure" "true"
    echo "Pulumi infrastructure changes detected"
else
    write_output "infrastructure" "false"
    echo "No Pulumi infrastructure changes"
fi
