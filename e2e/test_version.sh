#!/usr/bin/env bash
# e2e: ssssg version command
source "$(dirname "$0")/helpers.sh"
echo "=== e2e: version ==="

output=$("$SSSSG_BIN" version 2>&1)

begin_test "version shows Version field"
assert_output_contains "$output" "Version:" && pass

begin_test "version shows Commit field"
assert_output_contains "$output" "Commit:" && pass

begin_test "version shows Built field"
assert_output_contains "$output" "Built:" && pass

begin_test "version shows Go version"
assert_output_contains "$output" "Go version:" && pass

begin_test "version shows OS/Arch"
assert_output_contains "$output" "OS/Arch:" && pass

summary
