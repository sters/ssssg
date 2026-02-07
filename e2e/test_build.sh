#!/usr/bin/env bash
# e2e: ssssg build command
source "$(dirname "$0")/helpers.sh"
echo "=== e2e: build ==="

# ── Helper: create a minimal project ─────────────────────────

make_project() {
  local dir="$1"
  mkdir -p "$dir/templates" "$dir/static"
}

# ── Basic build with data ────────────────────────────────────

PROJECT="$WORK_DIR/basic"
make_project "$PROJECT"

cat > "$PROJECT/site.yaml" <<'YAML'
global:
  data:
    site_name: "E2E Test"
pages:
  - template: "index.html"
    output: "index.html"
    data:
      title: "Home"
      greeting: "Hello E2E"
YAML

cat > "$PROJECT/templates/index.html" <<'TMPL'
<html><body><h1>{{ .Page.greeting }}</h1><p>{{ .Global.site_name }}</p></body></html>
TMPL

"$SSSSG_BIN" build --config "$PROJECT/site.yaml" --timeout 10s >/dev/null 2>&1

begin_test "basic build creates output file"
assert_file_exists "$PROJECT/public/index.html" && pass

begin_test "basic build renders page data"
assert_file_contains "$PROJECT/public/index.html" "<h1>Hello E2E</h1>" && pass

begin_test "basic build renders global data"
assert_file_contains "$PROJECT/public/index.html" "<p>E2E Test</p>" && pass

# ── Build with layout + partials ─────────────────────────────

PROJECT="$WORK_DIR/layout"
make_project "$PROJECT"

cat > "$PROJECT/site.yaml" <<'YAML'
global:
  layout: "_layout.html"
  data:
    site_name: "Layout Test"
pages:
  - template: "index.html"
    output: "index.html"
    data:
      title: "Home"
      body: "Welcome"
YAML

cat > "$PROJECT/templates/_layout.html" <<'TMPL'
<!DOCTYPE html>
<html>
<head><title>{{ .Page.title }}</title></head>
<body>
{{ template "_header.html" . }}
{{ block "content" . }}{{ end }}
</body>
</html>
TMPL

cat > "$PROJECT/templates/_header.html" <<'TMPL'
<header>{{ .Global.site_name }}</header>
TMPL

cat > "$PROJECT/templates/index.html" <<'TMPL'
{{ define "content" }}<main>{{ .Page.body }}</main>{{ end }}
TMPL

"$SSSSG_BIN" build --config "$PROJECT/site.yaml" --timeout 10s >/dev/null 2>&1

begin_test "layout build renders title"
assert_file_contains "$PROJECT/public/index.html" "<title>Home</title>" && pass

begin_test "layout build renders header partial"
assert_file_contains "$PROJECT/public/index.html" "<header>Layout Test</header>" && pass

begin_test "layout build renders content block"
assert_file_contains "$PROJECT/public/index.html" "<main>Welcome</main>" && pass

# ── Build with local file fetch ──────────────────────────────

PROJECT="$WORK_DIR/fetch"
make_project "$PROJECT"

echo "body { margin: 0; }" > "$PROJECT/static/style.css"

cat > "$PROJECT/site.yaml" <<'YAML'
global:
  fetch:
    css: "static/style.css"
pages:
  - template: "index.html"
    output: "index.html"
    data:
      title: "Fetch"
YAML

cat > "$PROJECT/templates/index.html" <<'TMPL'
<html><head><style>{{ .Global.css | rawCSS }}</style></head><body>ok</body></html>
TMPL

"$SSSSG_BIN" build --config "$PROJECT/site.yaml" --timeout 10s >/dev/null 2>&1

begin_test "local fetch injects file content"
assert_file_contains "$PROJECT/public/index.html" "body { margin: 0; }" && pass

# ── Build with page-level layout override ────────────────────

PROJECT="$WORK_DIR/override"
make_project "$PROJECT"

cat > "$PROJECT/site.yaml" <<'YAML'
global:
  layout: "_layout.html"
  data:
    site_name: "Override"
pages:
  - template: "page.html"
    output: "page.html"
    layout: "_alt.html"
    data:
      body: "alt content"
YAML

cat > "$PROJECT/templates/_layout.html" <<'TMPL'
<div class="default">{{ block "content" . }}{{ end }}</div>
TMPL

cat > "$PROJECT/templates/_alt.html" <<'TMPL'
<div class="alt">{{ block "content" . }}{{ end }}</div>
TMPL

cat > "$PROJECT/templates/page.html" <<'TMPL'
{{ define "content" }}{{ .Page.body }}{{ end }}
TMPL

"$SSSSG_BIN" build --config "$PROJECT/site.yaml" --timeout 10s >/dev/null 2>&1

begin_test "page-level layout override works"
assert_file_contains "$PROJECT/public/page.html" '<div class="alt">' && pass

begin_test "page-level layout does not use default"
assert_file_not_contains "$PROJECT/public/page.html" '<div class="default">' && pass

# ── Static file copying ─────────────────────────────────────

PROJECT="$WORK_DIR/static"
make_project "$PROJECT"
mkdir -p "$PROJECT/static/img"
echo "body{}" > "$PROJECT/static/app.css"
echo "PNG_DATA" > "$PROJECT/static/img/logo.png"
echo "" > "$PROJECT/static/.gitkeep"

cat > "$PROJECT/site.yaml" <<'YAML'
pages:
  - template: "index.html"
    output: "index.html"
YAML

cat > "$PROJECT/templates/index.html" <<'TMPL'
<html><body>ok</body></html>
TMPL

"$SSSSG_BIN" build --config "$PROJECT/site.yaml" --timeout 10s >/dev/null 2>&1

begin_test "static files are copied"
assert_file_exists "$PROJECT/public/app.css" && pass

begin_test "static subdirectory files are copied"
assert_file_exists "$PROJECT/public/img/logo.png" && pass

begin_test "dotfiles are not copied"
assert_file_not_exists "$PROJECT/public/.gitkeep" && pass

# ── --clean flag removes stale files ─────────────────────────

PROJECT="$WORK_DIR/clean"
make_project "$PROJECT"

cat > "$PROJECT/site.yaml" <<'YAML'
pages:
  - template: "index.html"
    output: "index.html"
YAML

cat > "$PROJECT/templates/index.html" <<'TMPL'
<html><body>ok</body></html>
TMPL

# Create stale file
mkdir -p "$PROJECT/public"
echo "stale" > "$PROJECT/public/old.html"

"$SSSSG_BIN" build --config "$PROJECT/site.yaml" --timeout 10s --clean >/dev/null 2>&1

begin_test "--clean removes stale files"
assert_file_not_exists "$PROJECT/public/old.html" && pass

begin_test "--clean still generates new output"
assert_file_exists "$PROJECT/public/index.html" && pass

# ── Subdirectory output paths ────────────────────────────────

PROJECT="$WORK_DIR/subdir"
make_project "$PROJECT"

cat > "$PROJECT/site.yaml" <<'YAML'
pages:
  - template: "page.html"
    output: "blog/2024/post.html"
YAML

cat > "$PROJECT/templates/page.html" <<'TMPL'
<html><body>nested</body></html>
TMPL

"$SSSSG_BIN" build --config "$PROJECT/site.yaml" --timeout 10s >/dev/null 2>&1

begin_test "nested output directories are created"
assert_file_exists "$PROJECT/public/blog/2024/post.html" && pass

begin_test "nested output has correct content"
assert_file_contains "$PROJECT/public/blog/2024/post.html" "nested" && pass

# ── HTML auto-escaping (XSS protection) ─────────────────────

PROJECT="$WORK_DIR/escape"
make_project "$PROJECT"

cat > "$PROJECT/site.yaml" <<'YAML'
pages:
  - template: "index.html"
    output: "index.html"
    data:
      content: "<script>alert('xss')</script>"
YAML

cat > "$PROJECT/templates/index.html" <<'TMPL'
<html><body>{{ .Page.content }}</body></html>
TMPL

"$SSSSG_BIN" build --config "$PROJECT/site.yaml" --timeout 10s >/dev/null 2>&1

begin_test "html/template escapes dangerous content"
assert_file_not_contains "$PROJECT/public/index.html" "<script>" && pass

begin_test "html/template produces escaped output"
assert_file_contains "$PROJECT/public/index.html" "&lt;script&gt;" && pass

# ── Multiple pages in single build ───────────────────────────

PROJECT="$WORK_DIR/multi"
make_project "$PROJECT"

cat > "$PROJECT/site.yaml" <<'YAML'
pages:
  - template: "a.html"
    output: "a.html"
    data:
      name: "PageA"
  - template: "b.html"
    output: "b.html"
    data:
      name: "PageB"
  - template: "c.html"
    output: "c.html"
    data:
      name: "PageC"
YAML

for p in a b c; do
  cat > "$PROJECT/templates/${p}.html" <<TMPL
<html><body>{{ .Page.name }}</body></html>
TMPL
done

"$SSSSG_BIN" build --config "$PROJECT/site.yaml" --timeout 10s >/dev/null 2>&1

begin_test "multiple pages: a.html generated"
assert_file_contains "$PROJECT/public/a.html" "PageA" && pass

begin_test "multiple pages: b.html generated"
assert_file_contains "$PROJECT/public/b.html" "PageB" && pass

begin_test "multiple pages: c.html generated"
assert_file_contains "$PROJECT/public/c.html" "PageC" && pass

# ── Custom --output flag ─────────────────────────────────────

PROJECT="$WORK_DIR/customout"
make_project "$PROJECT"

cat > "$PROJECT/site.yaml" <<'YAML'
pages:
  - template: "index.html"
    output: "index.html"
YAML

cat > "$PROJECT/templates/index.html" <<'TMPL'
<html><body>custom output</body></html>
TMPL

CUSTOM_OUT="$WORK_DIR/customout_dist"
"$SSSSG_BIN" build --config "$PROJECT/site.yaml" --output "$CUSTOM_OUT" --timeout 10s >/dev/null 2>&1

begin_test "--output flag directs output to custom dir"
assert_file_exists "$CUSTOM_OUT/index.html" && pass

begin_test "--output custom dir has correct content"
assert_file_contains "$CUSTOM_OUT/index.html" "custom output" && pass

summary
