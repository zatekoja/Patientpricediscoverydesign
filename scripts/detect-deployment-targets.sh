#!/usr/bin/env bash
# detect-deployment-targets.sh - Set deployment target outputs for cd-deploy workflow.
#
# Usage: ./scripts/detect-deployment-targets.sh [deployment-type]
# deployment-type: auto | backend-only | frontend-only | infrastructure-only | full

set -euo pipefail

DEPLOY_TYPE="${1:-auto}"

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

deploy_infrastructure="false"
deploy_backend="false"
deploy_frontend="false"

if [ "$DEPLOY_TYPE" = "auto" ]; then
    CHANGED_FILES="$(changed_files)"

    # Check each component independently
    if echo "$CHANGED_FILES" | grep -q '^infrastructure/pulumi/'; then
        deploy_infrastructure="true"
    fi

    if echo "$CHANGED_FILES" | grep -q '^backend/'; then
        deploy_backend="true"
    fi

    if echo "$CHANGED_FILES" | grep -q '^Frontend/\|^package\.json\|^tsconfig\.json'; then
        deploy_frontend="true"
    fi

    # Scripts changes should trigger backend deploy
    if echo "$CHANGED_FILES" | grep -q '^scripts/'; then
        deploy_backend="true"
    fi

    # Workflow file changes should trigger backend + frontend deploy
    if echo "$CHANGED_FILES" | grep -q '^\.github/workflows/cd-deploy\.yml'; then
        deploy_backend="true"
        deploy_frontend="true"
    fi
else
    case "$DEPLOY_TYPE" in
        "backend-only")
            deploy_backend="true"
            ;;
        "frontend-only")
            deploy_frontend="true"
            ;;
        "infrastructure-only")
            deploy_infrastructure="true"
            ;;
        "full")
            deploy_infrastructure="true"
            deploy_backend="true"
            deploy_frontend="true"
            ;;
        *)
            echo "Invalid deployment type: $DEPLOY_TYPE" >&2
            exit 1
            ;;
    esac
fi

write_output "deploy-infrastructure" "$deploy_infrastructure"
write_output "deploy-backend" "$deploy_backend"
write_output "deploy-frontend" "$deploy_frontend"
