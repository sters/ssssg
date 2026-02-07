#!/usr/bin/env bash
# e2e: ssssg init command
source "$(dirname "$0")/helpers.sh"
echo "=== e2e: init ==="

# ── init in current directory ────────────────────────────────

PROJECT="$WORK_DIR/site1"
mkdir -p "$PROJECT"
"$SSSSG_BIN" init "$PROJECT" >/dev/null 2>&1

begin_test "init creates site.yaml"
assert_file_exists "$PROJECT/site.yaml" && pass

begin_test "init creates templates/_layout.html"
assert_file_exists "$PROJECT/templates/_layout.html" && pass

begin_test "init creates templates/_header.html"
assert_file_exists "$PROJECT/templates/_header.html" && pass

begin_test "init creates templates/_footer.html"
assert_file_exists "$PROJECT/templates/_footer.html" && pass

begin_test "init creates templates/index.html"
assert_file_exists "$PROJECT/templates/index.html" && pass

begin_test "init creates static/ directory"
assert_dir_exists "$PROJECT/static" && pass

# ── init does not overwrite existing files ───────────────────

echo "custom content" > "$PROJECT/site.yaml"
"$SSSSG_BIN" init "$PROJECT" >/dev/null 2>&1

begin_test "init does not overwrite existing site.yaml"
assert_file_contains "$PROJECT/site.yaml" "custom content" && pass

# ── init with subdirectory argument ──────────────────────────

"$SSSSG_BIN" init "$WORK_DIR/newproject" >/dev/null 2>&1

begin_test "init creates new subdirectory"
assert_dir_exists "$WORK_DIR/newproject" && pass

begin_test "init populates subdirectory"
assert_file_exists "$WORK_DIR/newproject/site.yaml" && pass

summary
