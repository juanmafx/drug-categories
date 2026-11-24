#!/usr/bin/env bash
# Runs both test suites and writes real results to reports/tests/.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
report_dir="$root/reports/tests"
mkdir -p "$report_dir"

backend_status=0
frontend_status=0

(cd "$root/backend" && go test -race -count=1 -cover -v ./...) >"$report_dir/backend-go-test.txt" 2>&1 || backend_status=$?
(cd "$root/frontend" && npx vitest run --reporter=default --reporter=junit --outputFile.junit="$report_dir/frontend-junit.xml") >"$report_dir/frontend-vitest.txt" 2>&1 || frontend_status=$?

sed -i -E 's/\x1b\[[0-9;]*m//g' "$report_dir/frontend-vitest.txt"
sed -i "s#$root#.#g" "$report_dir/backend-go-test.txt" "$report_dir/frontend-vitest.txt" "$report_dir/frontend-junit.xml"

backend_passed="$(grep -cE '^ *--- PASS' "$report_dir/backend-go-test.txt" || true)"
backend_failed="$(grep -cE '^ *--- FAIL' "$report_dir/backend-go-test.txt" || true)"
frontend_total="$(grep -oE '<testsuites [^>]*tests="[0-9]+"' "$report_dir/frontend-junit.xml" | grep -oE 'tests="[0-9]+"' | grep -oE '[0-9]+')"
frontend_failed="$(grep -oE '<testsuites [^>]*failures="[0-9]+"' "$report_dir/frontend-junit.xml" | grep -oE 'failures="[0-9]+"' | grep -oE '[0-9]+')"

{
  echo "# Test Report"
  echo
  echo "- Generated (UTC): $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "- Commit base: $(git -C "$root" rev-parse --short HEAD)"
  echo "- Go: $(cd "$root/backend" && go env GOVERSION)"
  echo "- Node: $(node --version)"
  echo
  echo "| Suite | Passed | Failed | Exit code |"
  echo "| --- | --- | --- | --- |"
  echo "| Backend (go test -race, including subtests) | $backend_passed | $backend_failed | $backend_status |"
  echo "| Frontend (Vitest) | $((frontend_total - frontend_failed)) | $frontend_failed | $frontend_status |"
  echo
  echo "## Backend package coverage"
  echo
  echo '```text'
  grep -E '^(ok|FAIL|\?)' "$report_dir/backend-go-test.txt" | sed -E 's/\t[0-9.]+s//' || true
  echo '```'
} >"$report_dir/SUMMARY.md"

cat "$report_dir/SUMMARY.md"
exit $((backend_status + frontend_status))
