#!/usr/bin/env bash
# Common helpers for e2e tests

set -euo pipefail

# Binary path — set by Makefile or default to project bin/
SSSSG_BIN="${SSSSG_BIN:-$(cd "$(dirname "$0")/.." && pwd)/bin/ssssg}"

if [[ ! -x "$SSSSG_BIN" ]]; then
  echo "FATAL: ssssg binary not found at $SSSSG_BIN (run 'make build' first)" >&2
  exit 1
fi

# Track test results
_TESTS_RUN=0
_TESTS_PASSED=0
_TESTS_FAILED=0
_CURRENT_TEST=""

# Create a temp directory that is cleaned up on exit
WORK_DIR="$(mktemp -d)"
cleanup() { rm -rf "$WORK_DIR"; }
trap cleanup EXIT

# ── Assertions ───────────────────────────────────────────────

begin_test() {
  _CURRENT_TEST="$1"
  _TESTS_RUN=$((_TESTS_RUN + 1))
  printf "  %-50s " "$_CURRENT_TEST"
}

pass() {
  _TESTS_PASSED=$((_TESTS_PASSED + 1))
  echo "PASS"
}

fail() {
  _TESTS_FAILED=$((_TESTS_FAILED + 1))
  echo "FAIL"
  echo "    $1" >&2
}

assert_file_exists() {
  local path="$1"
  if [[ ! -f "$path" ]]; then
    fail "file does not exist: $path"
    return 1
  fi
}

assert_dir_exists() {
  local path="$1"
  if [[ ! -d "$path" ]]; then
    fail "directory does not exist: $path"
    return 1
  fi
}

assert_file_not_exists() {
  local path="$1"
  if [[ -f "$path" ]]; then
    fail "file should not exist: $path"
    return 1
  fi
}

assert_file_contains() {
  local path="$1"
  local expected="$2"
  if ! grep -qF "$expected" "$path"; then
    fail "file $path does not contain: $expected"
    return 1
  fi
}

assert_file_not_contains() {
  local path="$1"
  local unexpected="$2"
  if grep -qF "$unexpected" "$path"; then
    fail "file $path should not contain: $unexpected"
    return 1
  fi
}

assert_output_contains() {
  local output="$1"
  local expected="$2"
  if ! echo "$output" | grep -qF "$expected"; then
    fail "output does not contain: $expected"
    return 1
  fi
}

assert_exit_code() {
  local expected="$1"
  local actual="$2"
  if [[ "$actual" -ne "$expected" ]]; then
    fail "expected exit code $expected, got $actual"
    return 1
  fi
}

# Print final summary and exit with appropriate code
summary() {
  echo ""
  echo "  Results: $_TESTS_PASSED/$_TESTS_RUN passed, $_TESTS_FAILED failed"
  if [[ $_TESTS_FAILED -gt 0 ]]; then
    exit 1
  fi
}
