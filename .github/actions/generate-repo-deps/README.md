# Generate Repo Dependencies

This directory contains a composite action for generating GraphQL code and mocks for Go services.

## Usage

### As a Composite Action (Reusable in Other Workflows)

You can use this action in any workflow by referencing it with the `uses` keyword:

```yaml
steps:
  - name: Checkout code
    uses: actions/checkout@v4
    with:
      fetch-depth: 0

  - name: Generate dependencies
    uses: ./.github/actions/generate-repo-deps
    with:
      working-directory: 'Backend/Services'
      generate-mocks: 'true'
      go-version: '1.24'
```

### As a Standalone Workflow

There's also a standalone workflow at `.github/workflows/generate-repo-deps.yml` that can be triggered manually via `workflow_dispatch`.

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `working-directory` | Base working directory for the Go services | No | `Backend/Services` |
| `generate-mocks` | Whether to also generate mocks if mockery config exists | No | `true` |
| `go-version` | Go version to use | No | `1.24` |

## What It Does

1. **Sets up Go** with the specified version
2. **Generates GraphQL code** using gqlgen in the `{working-directory}/gql-gen` directory
3. **Generates mocks** (if enabled) using mockery based on `.mockery.yaml` config

## Requirements

- The repository must be checked out before using this action
- For GraphQL generation: `{working-directory}/gql-gen` directory must exist with gqlgen configuration
- For mock generation: `.mockery.yaml` file must exist in the working directory

## Examples

### Example 1: Use in PR Validation

```yaml
name: PR Validation
on: pull_request

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Generate code
        uses: ./.github/actions/generate-repo-deps
        with:
          working-directory: 'Backend/Services'
          generate-mocks: 'true'
      
      - name: Check for uncommitted changes
        run: |
          git diff --exit-code || (echo "Generated code is out of date!" && exit 1)
```

### Example 2: Custom Go Version

```yaml
- name: Generate with Go 1.23
  uses: ./.github/actions/generate-repo-deps
  with:
    go-version: '1.23'
    working-directory: 'Backend/Services'
```

### Example 3: Skip Mock Generation

```yaml
- name: Generate only GraphQL
  uses: ./.github/actions/generate-repo-deps
  with:
    generate-mocks: 'false'
```

## Troubleshooting

### Error: "Can't find 'action.yml'"

This error occurs when:
1. The action is referenced with an incorrect path
2. The repository hasn't been checked out with `actions/checkout`

**Solution**: Ensure you run `actions/checkout` before using this action, and use the correct path `./.github/actions/generate-repo-deps`

### GraphQL Generation Fails

Ensure that:
- `gql-gen` directory exists in your working directory
- `gqlgen.yml` configuration file is present
- GraphQL schema files are properly configured

### Mock Generation Skipped

If you see "No .mockery.yaml found", it means:
- The `.mockery.yaml` file doesn't exist in the working directory
- This is expected behavior if you don't need mocks

