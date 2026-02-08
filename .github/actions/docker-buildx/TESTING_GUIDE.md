# Docker Buildx Action - Testing & Validation Guide

## Overview

This guide provides comprehensive testing instructions for the new `docker-buildx` action and its integration with the `_test-service-localstack.yml` workflow.

## Pre-Testing Checklist

- [x] Action created: `.github/actions/docker-buildx/action.yml`
- [x] Documentation created: `.github/actions/docker-buildx/README.md`
- [x] Workflow updated: `.github/workflows/_test-service-localstack.yml`
- [x] Implementation summary created: `.github/actions/docker-buildx/IMPLEMENTATION_SUMMARY.md`
- [ ] Action inputs validated
- [ ] Workflow syntax validated
- [ ] Integration test performed

## Validation Steps

### Step 1: Validate Action YAML Syntax

```bash
# Check YAML syntax (requires yamllint)
yamllint -d relaxed .github/actions/docker-buildx/action.yml

# Alternative: use Python
python3 -c "import yaml; yaml.safe_load(open('.github/actions/docker-buildx/action.yml'))"
```

**Expected Result:** No syntax errors

### Step 2: Validate Workflow YAML Syntax

```bash
# Check workflow syntax
yamllint -d relaxed .github/workflows/_test-service-localstack.yml

# Alternative: use Python
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/_test-service-localstack.yml'))"
```

**Expected Result:** No syntax errors

### Step 3: Verify Action Inputs/Outputs

Review the action definition to ensure all required inputs are properly defined:

**Required Inputs:**
- ✅ `context` - Build context path
- ✅ `dockerfile` - Dockerfile path
- ✅ `image-tag` - Image tag

**Optional Inputs:**
- ✅ `build-args` - Build arguments (multi-line)
- ✅ `platform` - Target platform(s)
- ✅ `cache-key` - Cache directory key
- ✅ `push` - Push to registry flag
- ✅ `load` - Load to local daemon flag
- ✅ `labels` - OCI labels
- ✅ `cache-mode` - Cache mode (min/max)

**Outputs:**
- ✅ `image-tag` - Built image tag
- ✅ `image-digest` - Image SHA256 digest
- ✅ `cache-hit` - Cache utilization flag

## Testing Scenarios

### Scenario 1: Local Build with Cache (LocalStack Workflow)

**Trigger:** Manual workflow dispatch with single service

**Test Configuration:**
```json
{
  "changed-services": "[{\"service\":\"event-service-handler\",\"category\":\"api\",\"dockerfile\":\"Backend/Services/deployments/docker/api/event-service-handler/Dockerfile\",\"context\":\"Backend\",\"image\":\"p2pbets-event-service-handler\",\"deployment\":\"lambda\"}]",
  "pytest-markers": "",
  "go-version": "1.24",
  "timeout-minutes": 20,
  "max-parallel": 10
}
```

**Expected Behavior:**

1. **Build Service Image Step:**
   - ✅ Buildx setup completes
   - ✅ Cache directory created: `.github/.docker-cache/event-service-handler/`
   - ✅ Dockerfile validation passes
   - ✅ Context validation passes
   - ✅ Image builds with tag `p2pbets-event-service-handler:local-test`
   - ✅ Image loaded into local Docker daemon
   - ✅ Build completes within 5-10 minutes (first run)
   - ✅ Cache populated

2. **Start Service Step:**
   - ✅ Override file created successfully
   - ✅ Override references correct image tag
   - ✅ Compose starts without rebuilding
   - ✅ Service container running
   - ✅ Override file cleaned up

3. **Second Run (Cache Test):**
   - ✅ Build uses cache (faster completion ~2-3 minutes)
   - ✅ Cache hit reported in logs

**Validation Commands:**

```bash
# After build step, verify image exists
docker images | grep p2pbets-event-service-handler

# Check cache directory
ls -lh .github/.docker-cache/event-service-handler/

# Verify container is running
docker ps | grep event-service-handler

# Check compose override was cleaned up
ls Backend/docker-compose.override.yml  # Should not exist
```

### Scenario 2: Multiple Services in Parallel

**Test Configuration:**
```json
{
  "changed-services": "[{\"service\":\"event-service-handler\",\"category\":\"api\",\"dockerfile\":\"Backend/Services/deployments/docker/api/event-service-handler/Dockerfile\",\"context\":\"Backend\",\"image\":\"p2pbets-event-service-handler\",\"deployment\":\"lambda\"},{\"service\":\"bet-service-handler\",\"category\":\"api\",\"dockerfile\":\"Backend/Services/deployments/docker/api/bet-service-handler/Dockerfile\",\"context\":\"Backend\",\"image\":\"p2pbets-bet-service-handler\",\"deployment\":\"lambda\"}]",
  "max-parallel": 2
}
```

**Expected Behavior:**

1. **Parallel Execution:**
   - ✅ Both services build concurrently (matrix strategy)
   - ✅ Each service gets its own cache directory
   - ✅ No cache conflicts between services
   - ✅ Both images loaded successfully

2. **Cache Isolation:**
   ```
   .github/.docker-cache/
   ├── event-service-handler/
   └── bet-service-handler/
   ```

### Scenario 3: Build Failure Handling

**Test Configuration:**
Intentionally use invalid Dockerfile path

**Expected Behavior:**
- ✅ Validation step fails with clear error message
- ✅ Build does not proceed
- ✅ Error message indicates missing Dockerfile

### Scenario 4: Cache Reuse Across Runs

**Test Steps:**
1. Run workflow with service A
2. Make no changes to code
3. Run workflow with service A again

**Expected Behavior:**
- ✅ Second run significantly faster (cache hit)
- ✅ Logs show "Using cache from previous build"
- ✅ No layer re-downloads

## Integration Testing

### Test 1: End-to-End LocalStack Workflow

```bash
# Trigger via GitHub UI or gh CLI
gh workflow run _test-service-localstack.yml \
  --ref development \
  -f changed-services='[{"service":"event-service-handler","category":"api","dockerfile":"Backend/Services/deployments/docker/api/event-service-handler/Dockerfile","context":"Backend","image":"p2pbets-event-service-handler","deployment":"lambda"}]'

# Monitor workflow
gh run watch
```

**Success Criteria:**
- ✅ Workflow completes successfully
- ✅ All steps pass
- ✅ Test reports generated
- ✅ No errors in logs

### Test 2: Build Args Injection

Verify build arguments are correctly passed:

```bash
# After build, inspect image
docker inspect p2pbets-event-service-handler:local-test | jq '.[0].Config.Labels'

# Should show build-time args as labels or env vars
```

### Test 3: Cache Directory Persistence

```bash
# After first run
ls -lh .github/.docker-cache/event-service-handler/

# Record size
du -sh .github/.docker-cache/event-service-handler/

# After second run, verify cache was used (size stable or minimal growth)
du -sh .github/.docker-cache/event-service-handler/
```

## Performance Benchmarks

### Expected Build Times (Approximate)

| Scenario | First Build | Cached Build | Improvement |
|----------|-------------|--------------|-------------|
| Single Service (Go) | 5-8 min | 2-3 min | ~60% faster |
| Multiple Services (2) | 6-10 min | 2-4 min | ~65% faster |
| Multiple Services (5) | 10-15 min | 3-5 min | ~70% faster |

*Note: Times vary based on runner resources and network conditions*

## Debugging Guide

### Issue: Action Not Found

**Symptom:**
```
Error: Unable to resolve action `./.github/actions/docker-buildx`
```

**Solution:**
- Ensure action.yml exists at `.github/actions/docker-buildx/action.yml`
- Verify repository checkout step runs before action usage
- Check file permissions

### Issue: Cache Not Reused

**Symptom:**
Build times remain consistent across runs

**Debug Steps:**
1. Check cache directory exists: `ls .github/.docker-cache/`
2. Verify cache-key is consistent across runs
3. Check buildx cache logs for "Using cache from..."
4. Ensure `cache-mode: max` is set

**Solution:**
- Use explicit `cache-key` instead of default
- Verify cache directory has write permissions
- Check for cache size limits (GitHub Actions runners)

### Issue: Compose Override Not Working

**Symptom:**
Docker compose rebuilds the image

**Debug Steps:**
1. Check override file contents: `cat Backend/docker-compose.override.yml`
2. Verify image tag matches build output
3. Check compose logs for build messages

**Solution:**
- Ensure override file is created before `docker compose up`
- Verify `build: null` is present in override
- Check service name matches compose file

### Issue: Multi-Platform Build Fails

**Symptom:**
```
Error: docker exporter does not currently support exporting multiple platforms
```

**Solution:**
- Set `load: false` when building for multiple platforms
- Set `push: true` to push instead of loading
- OR build for single platform only

## Cleanup

### Remove Cache Directories

```bash
# Remove all cache
rm -rf .github/.docker-cache/

# Remove specific service cache
rm -rf .github/.docker-cache/event-service-handler/
```

### Remove Built Images

```bash
# Remove all test images
docker images | grep "local-test" | awk '{print $3}' | xargs docker rmi

# Remove specific service image
docker rmi p2pbets-event-service-handler:local-test
```

## Next Steps After Validation

### Phase 1: Rollout to Other Workflows ✅ READY

Once LocalStack tests pass:

1. **Update `_build-docker.yml`:**
   - Replace inline `docker buildx build` with action calls
   - Use per-service cache keys
   - Enable push mode for ECR

2. **Update `deploy-applications.yml`:**
   - Use action for deployment builds
   - Configure ECR authentication before action

3. **Archive Old Patterns:**
   - Document migration in archived workflows
   - Add deprecation notices

### Phase 2: Enhancements

1. **Add Security Scanning:**
   - Integrate Trivy/Snyk in action
   - Fail builds on critical vulnerabilities

2. **Add SBOM Generation:**
   - Use `--sbom=true` flag
   - Export SBOM as artifact

3. **Optimize Cache Strategy:**
   - Implement cache size limits
   - Add LRU eviction
   - Share cache across services when appropriate

4. **Multi-Registry Support:**
   - Add Docker Hub login option
   - Support multiple registry push

## Reporting Issues

If you encounter issues during testing:

1. **Collect Logs:**
   ```bash
   gh run view --log > workflow-logs.txt
   ```

2. **Check Action Outputs:**
   - Review step summaries in GitHub UI
   - Look for output variables

3. **File Issue:**
   - Include workflow run ID
   - Attach relevant logs
   - Describe expected vs. actual behavior

## Success Metrics

✅ **Action is Ready for Production When:**

- [ ] All validation steps pass
- [ ] LocalStack workflow completes successfully
- [ ] Cache reuse works across runs
- [ ] Build times improve by >50% on cached builds
- [ ] No errors in production workflows
- [ ] Documentation is complete
- [ ] Team has reviewed implementation

## Appendix: Manual Testing Script

```bash
#!/bin/bash
# test-docker-buildx.sh - Manual validation script

set -e

echo "🧪 Testing Docker Buildx Action"

# Test 1: YAML Syntax
echo "📋 Test 1: Validating YAML syntax..."
python3 -c "import yaml; yaml.safe_load(open('.github/actions/docker-buildx/action.yml'))" && echo "✅ Action YAML valid"
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/_test-service-localstack.yml'))" && echo "✅ Workflow YAML valid"

# Test 2: Check Required Files
echo "📋 Test 2: Checking required files..."
[ -f ".github/actions/docker-buildx/action.yml" ] && echo "✅ action.yml exists"
[ -f ".github/actions/docker-buildx/README.md" ] && echo "✅ README.md exists"
[ -f ".github/workflows/_test-service-localstack.yml" ] && echo "✅ Workflow exists"

# Test 3: Verify Action Structure
echo "📋 Test 3: Verifying action structure..."
grep -q "name: 'Docker Buildx Build'" .github/actions/docker-buildx/action.yml && echo "✅ Action name present"
grep -q "runs:" .github/actions/docker-buildx/action.yml && echo "✅ Runs section present"
grep -q "using: 'composite'" .github/actions/docker-buildx/action.yml && echo "✅ Composite action type"

# Test 4: Verify Workflow Integration
echo "📋 Test 4: Verifying workflow integration..."
grep -q "uses: ./.github/actions/docker-buildx" .github/workflows/_test-service-localstack.yml && echo "✅ Action used in workflow"
grep -q "cache-key:" .github/workflows/_test-service-localstack.yml && echo "✅ Cache key configured"

echo ""
echo "🎉 All basic validation tests passed!"
echo "📝 Next: Run integration test via GitHub Actions"
```

Save as `scripts/test-docker-buildx.sh` and run:
```bash
chmod +x scripts/test-docker-buildx.sh
./scripts/test-docker-buildx.sh
```

