# ssssg - Super Simple Static Site Generator

[![go](https://github.com/sters/ssssg/workflows/Go/badge.svg)](https://github.com/sters/ssssg/actions?query=workflow%3AGo)
[![coverage](docs/coverage.svg)](https://github.com/sters/ssssg)
[![go-report](https://goreportcard.com/badge/github.com/sters/ssssg)](https://goreportcard.com/report/github.com/sters/ssssg)

Define data and URLs in YAML, write HTML with Go templates, and build to generate static HTML.

## Install

```shell
go install github.com/sters/ssssg@latest
```

or use specific version from [Releases](https://github.com/sters/ssssg/releases).

## Quick Start

```shell
# Initialize a new project
ssssg init mysite
cd mysite

# Build the site
ssssg build
```

## Usage

```shell
ssssg build                       # Build with defaults (site.yaml)
ssssg build --config site.yaml    # Specify config file
ssssg build --templates templates/
ssssg build --static static/
ssssg build --output public/
ssssg build --timeout 30s

ssssg init                        # Initialize in current directory
ssssg init mysite                 # Initialize in specified directory

ssssg version                     # Show version info
```

## Project Structure

```
my-site/
  site.yaml          # Site definition (data, URLs, pages)
  templates/
    _layout.html      # Shared layout (_ prefix = shared file)
    _header.html      # Partial
    _footer.html      # Partial
    index.html        # Page template
  static/             # Static files (copied to output as-is)
  public/             # Output directory (generated)
```

## site.yaml

```yaml
global:
  layout: "_layout.html"
  data:
    site_name: "My Site"
  fetch:
    reset_css: "https://cdn.example.com/reset.css"
    custom_css: "static/style.css"

pages:
  - template: "index.html"
    output: "index.html"
    data:
      title: "Home"
      greeting: "Welcome!"
    fetch:
      projects: "https://api.example.com/projects.json"

  - template: "about.html"
    output: "about/index.html"
    layout: "_other_layout.html"
    data:
      title: "About"
```

## Templates

Templates use Go's `html/template` syntax. Data is accessed via `.Global`, `.Page`, and `.Static`:

```html
{{ .Global.site_name }}
{{ .Page.title }}
{{ .Global.reset_css | raw }}
```

Use `| raw` for fetched HTML/CSS content that should not be escaped.

### Static File Metadata

`.Static` provides metadata for all files in the output directory (scanned after pipeline processing). Each entry is a `StaticFileInfo` with these fields:

| Field | Type | Description |
|-------|------|-------------|
| `Path` | string | Forward-slash relative path (e.g. `img/photo.png`) |
| `Size` | int64 | File size in bytes |
| `Width` | int | Image width in px (0 if not an image) |
| `Height` | int | Image height in px (0 if not an image) |

Supported image formats: JPEG, PNG, GIF, WebP.

```html
{{ $img := index .Static "img/photo.png" }}
{{ if $img.Path }}
  <img src="/img/photo.png" width="{{ $img.Width }}" height="{{ $img.Height }}">
{{ end }}
```

Accessing a non-existent key returns a zero-value struct (no error), so you can safely check `$img.Path` for existence.

## Static File Pipelines

By default, files in `static/` are copied to the output directory as-is. You can define pipelines to process matched files with shell commands:

```yaml
static:
  pipelines:
    - match: "*.jpg"
      commands:
        - "cp {{.Src}} {{.Dest}}"
        - "mogrify -resize 800x600 {{.Dest}}"
        - "jpegoptim --strip-all {{.Dest}}"
    - match: "*.png"
      commands:
        - "cp {{.Src}} {{.Dest}}"
        - "optipng -o2 {{.Dest}}"
    - match: "images/*.webp"
      commands:
        - "cwebp -q 80 {{.Src}} -o {{.Dest}}"
```

### Matching rules

- Pattern without `/`: matches against the **basename** (e.g. `*.jpg` matches `images/photo.jpg`)
- Pattern with `/`: matches against the **relative path** from static directory (e.g. `images/*.webp`)
- First matching pipeline wins
- Unmatched files are copied as-is

### Template variables

Commands are processed with Go `text/template`. Available variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `{{.Src}}` | Source absolute path | `/path/to/static/photo.jpg` |
| `{{.Dest}}` | Destination absolute path | `/path/to/public/photo.jpg` |
| `{{.Dir}}` | Destination directory | `/path/to/public` |
| `{{.Name}}` | File name | `photo.jpg` |
| `{{.Ext}}` | Extension | `.jpg` |
| `{{.Base}}` | Name without extension | `photo` |
