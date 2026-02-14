#!/usr/bin/env node

/**
 * Pulumi Config Linter
 *
 * Validates Pulumi YAML files against rules from the official Pulumi docs:
 * https://www.pulumi.com/docs/iac/concepts/projects/project-file/#config-options
 * https://www.pulumi.com/docs/iac/concepts/config/#project-level-configuration
 *
 * Rules enforced:
 *
 *  R1 – "Schemas are only valid for project property keys."
 *       Config keys whose namespace does NOT match the project name
 *       must not use schema attributes (type, default, items, secret).
 *       Provider keys (e.g. aws:region) should use a direct value or
 *       a nested { value: … } declaration.
 *
 *  R2 – Stack config files must not contain schema attributes (type,
 *       default, items) — those belong in Pulumi.yaml only.
 *
 *  R3 – All keys in stack config files must be namespaced (contain ':').
 *
 * Usage:
 *   node scripts/lint-pulumi-config.js          # from infrastructure/pulumi/
 *   npm run lint:config                          # via package.json script
 *
 * Zero external dependencies — uses only Node.js built-ins.
 */

const fs = require('fs');
const path = require('path');

const ROOT = path.resolve(__dirname, '..');
const PROJECT_FILE = path.join(ROOT, 'Pulumi.yaml');

let exitCode = 0;
let warnings = 0;

function error(file, msg) {
  console.error(`❌  ${path.relative(ROOT, file)}: ${msg}`);
  exitCode = 1;
}

function warn(file, msg) {
  console.warn(`⚠️  ${path.relative(ROOT, file)}: ${msg}`);
  warnings++;
}

// Schema-only attributes that are NOT valid on non-project keys
const SCHEMA_ATTRS = ['type', 'default', 'items'];

// ── 1.  Read Pulumi.yaml ─────────────────────────────────────────────────────
if (!fs.existsSync(PROJECT_FILE)) {
  console.error('❌  Pulumi.yaml not found — run this from infrastructure/pulumi/');
  process.exit(1);
}

const projectContent = fs.readFileSync(PROJECT_FILE, 'utf8');
const lines = projectContent.split('\n');

// Extract project name (first "name:" at top level)
const nameMatch = projectContent.match(/^name:\s*(.+)$/m);
if (!nameMatch) {
  error(PROJECT_FILE, '"name" field is missing');
  process.exit(1);
}
const projectName = nameMatch[1].trim();
console.log(`Project: ${projectName}\n`);

// ── 2.  R1: Detect schema attrs on non-project-namespaced keys ───────────────
//
// Per the docs:
//   "Schemas are only valid for project property keys. For setting the
//    value of a provider configuration either use a direct value, or
//    the nested value declaration."
//
// Project property keys are keys whose namespace matches the project name,
// e.g. if project is "my-app", then "my-app:foo" is project-namespaced.

let inConfig = false;
let currentKey = null;
let currentKeyIndent = 0;

for (let i = 0; i < lines.length; i++) {
  const line = lines[i];
  const lineNum = i + 1;

  // Detect start of config block
  if (/^config:\s*$/.test(line)) {
    inConfig = true;
    continue;
  }

  if (!inConfig) continue;

  // Exit config block when we hit a top-level key
  if (/^\S/.test(line) && !/^config:/.test(line)) {
    inConfig = false;
    currentKey = null;
    continue;
  }

  // Detect a config key line (e.g. "  aws:region:" or "  ohi:environment:")
  // A key can have an inline value ("  aws:region: us-east-1") or be a block ("  aws:region:\n")
  const keyMatch = line.match(/^(\s+)([a-zA-Z0-9_-]+:[a-zA-Z0-9_.-]+):\s*(.*)$/);
  if (keyMatch) {
    currentKey = keyMatch[2];
    currentKeyIndent = keyMatch[1].length;
    // If there's an inline value, no sub-attributes to check — move on
    if (keyMatch[3].trim() !== '') {
      currentKey = null;
    }
    continue;
  }

  // Check sub-attributes of the current key for schema-only attrs
  if (currentKey) {
    const indent = (line.match(/^(\s*)/) || ['', ''])[1].length;

    // If we've returned to the same or lower indent, this key block is over
    if (indent <= currentKeyIndent && line.trim() !== '') {
      currentKey = null;
      // Re-process this line as a potential new key
      i--;
      continue;
    }

    // Check if this sub-attribute is a schema attribute
    const attrMatch = line.match(/^\s+(type|default|items|secret):\s*/);
    if (attrMatch) {
      const attr = attrMatch[1];
      const namespace = currentKey.split(':')[0];
      const isProjectKey = namespace === projectName;

      if (!isProjectKey && SCHEMA_ATTRS.includes(attr)) {
        error(
          PROJECT_FILE,
          `Line ${lineNum}: Config key '${currentKey}' is not namespaced by the project ` +
            `('${projectName}') and must not use '${attr}'. ` +
            `Per Pulumi docs: "Schemas are only valid for project property keys." ` +
            `Use a direct value or the nested { value: … } form instead.`
        );
      }
    }
  }
}

// ── 3.  Validate stack config files (R2 + R3) ────────────────────────────────
const stackFiles = fs.readdirSync(ROOT).filter(f => /^Pulumi\..+\.yaml$/.test(f));

for (const stackFile of stackFiles) {
  const stackPath = path.join(ROOT, stackFile);
  const stackContent = fs.readFileSync(stackPath, 'utf8');
  const stackLines = stackContent.split('\n');

  // R3: Check for keys without namespace
  const configKeyRegex = /^\s{2}([a-zA-Z0-9_.-]+):\s/gm;
  let match;
  while ((match = configKeyRegex.exec(stackContent)) !== null) {
    const key = match[1];
    if (!key.includes(':')) {
      error(stackPath, `Config key '${key}' is missing a namespace (expected format 'namespace:key')`);
    }
  }

  // R2: Check for schema attributes in stack files (never valid there)
  for (let i = 0; i < stackLines.length; i++) {
    const sLine = stackLines[i];
    const schemaMatch = sLine.match(/^\s+(type|default|items):\s/);
    if (schemaMatch) {
      warn(
        stackPath,
        `Line ${i + 1}: Stack config files should not contain '${schemaMatch[1]}:' ` +
          `— schema declarations belong in Pulumi.yaml only`
      );
    }
  }
}

// ── 4.  Summary ──────────────────────────────────────────────────────────────
console.log('');
if (exitCode === 0) {
  const warnMsg = warnings > 0 ? ` (${warnings} warning(s))` : '';
  console.log(`✅  Pulumi config lint passed — ${stackFiles.length} stack file(s) checked${warnMsg}`);
} else {
  console.error(`💥  Pulumi config lint failed`);
}

process.exit(exitCode);
