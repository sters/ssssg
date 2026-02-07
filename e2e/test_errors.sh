#!/usr/bin/env bash
# e2e: error handling
source "$(dirname "$0")/helpers.sh"
echo "=== e2e: errors ==="

# ── Missing config file ──────────────────────────────────────

begin_test "missing config file returns error"
rc=0
"$SSSSG_BIN" build --config "/nonexistent/site.yaml" >/dev/null 2>&1 || rc=$?
assert_exit_code 1 "$rc" && pass

# ── Invalid YAML ─────────────────────────────────────────────

PROJECT="$WORK_DIR/invalid_yaml"
mkdir -p "$PROJECT"
echo "{{{{invalid yaml" > "$PROJECT/site.yaml"

begin_test "invalid YAML returns error"
rc=0
"$SSSSG_BIN" build --config "$PROJECT/site.yaml" >/dev/null 2>&1 || rc=$?
assert_exit_code 1 "$rc" && pass

# ── Missing template field ───────────────────────────────────

PROJECT="$WORK_DIR/no_template"
mkdir -p "$PROJECT/templates"

cat > "$PROJECT/site.yaml" <<'YAML'
pages:
  - output: "index.html"
YAML

begin_test "missing template field returns error"
rc=0
"$SSSSG_BIN" build --config "$PROJECT/site.yaml" >/dev/null 2>&1 || rc=$?
assert_exit_code 1 "$rc" && pass

# ── Missing output field ─────────────────────────────────────

PROJECT="$WORK_DIR/no_output"
mkdir -p "$PROJECT/templates"

cat > "$PROJECT/site.yaml" <<'YAML'
pages:
  - template: "index.html"
YAML

begin_test "missing output field returns error"
rc=0
"$SSSSG_BIN" build --config "$PROJECT/site.yaml" >/dev/null 2>&1 || rc=$?
assert_exit_code 1 "$rc" && pass

# ── Path traversal in output ─────────────────────────────────

PROJECT="$WORK_DIR/traversal"
mkdir -p "$PROJECT/templates"

cat > "$PROJECT/site.yaml" <<'YAML'
pages:
  - template: "index.html"
    output: "../../etc/passwd"
YAML

begin_test "path traversal in output is rejected"
rc=0
output=$("$SSSSG_BIN" build --config "$PROJECT/site.yaml" 2>&1 || true)
rc=$?
# The error message should mention path traversal; the exit code should be non-zero
if echo "$output" | grep -qi "escape\|traversal"; then
  pass
else
  # Even if the message doesn't match, a non-zero exit is acceptable
  assert_exit_code 1 "$rc" && pass
fi

# ── Absolute output path ─────────────────────────────────────

PROJECT="$WORK_DIR/abs_output"
mkdir -p "$PROJECT/templates"

cat > "$PROJECT/site.yaml" <<'YAML'
pages:
  - template: "index.html"
    output: "/tmp/evil.html"
YAML

begin_test "absolute output path is rejected"
rc=0
"$SSSSG_BIN" build --config "$PROJECT/site.yaml" >/dev/null 2>&1 || rc=$?
assert_exit_code 1 "$rc" && pass

# ── Missing template file on disk ────────────────────────────

PROJECT="$WORK_DIR/missing_tmpl_file"
mkdir -p "$PROJECT/templates" "$PROJECT/static"

cat > "$PROJECT/site.yaml" <<'YAML'
pages:
  - template: "nonexistent.html"
    output: "index.html"
YAML

begin_test "missing template file returns error"
rc=0
"$SSSSG_BIN" build --config "$PROJECT/site.yaml" >/dev/null 2>&1 || rc=$?
assert_exit_code 1 "$rc" && pass

# ── Fetch of nonexistent local file ──────────────────────────

PROJECT="$WORK_DIR/bad_fetch"
mkdir -p "$PROJECT/templates" "$PROJECT/static"

cat > "$PROJECT/site.yaml" <<'YAML'
pages:
  - template: "index.html"
    output: "index.html"
    fetch:
      data: "nonexistent_file.txt"
YAML

cat > "$PROJECT/templates/index.html" <<'TMPL'
<html><body>{{ .Page.data }}</body></html>
TMPL

begin_test "fetch of nonexistent file returns error"
rc=0
"$SSSSG_BIN" build --config "$PROJECT/site.yaml" --timeout 5s >/dev/null 2>&1 || rc=$?
assert_exit_code 1 "$rc" && pass

summary
