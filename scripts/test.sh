#!/bin/bash

cd ../compiler

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Initialize counters
passed=0
failed=0
skipped=0

echo "Running tests..."

# Run tests and capture output
test_output=$(go test ./... -v 2>&1)
exit_code=$?

# Parse the output
while IFS= read -r line; do
    if [[ "$line" == *"--- PASS:"* ]]; then
        ((passed++))
        echo -e "${GREEN}PASS${NC}: ${line#*--- PASS: }"
    elif [[ "$line" == *"--- FAIL:"* ]]; then
        ((failed++))
        echo -e "${RED}FAIL${NC}: ${line#*--- FAIL: }"
    elif [[ "$line" == *"--- SKIP:"* ]]; then
        ((skipped++))
        echo -e "${YELLOW}SKIP${NC}: ${line#*--- SKIP: }"
    elif [[ "$line" == *"PASS"* ]] && [[ "$line" == *"ok"* ]]; then
        echo -e "${GREEN}✓${NC} $line"
    elif [[ "$line" == *"FAIL"* ]]; then
        echo -e "${RED}✗${NC} $line"
    fi
done <<< "$test_output"

total=$((passed + failed + skipped))

echo
echo -e "Passed : ${GREEN}$passed${NC}"
echo -e "Failed : ${RED}$failed${NC}"
echo -e "Skipped: ${YELLOW}$skipped${NC}"
echo "Total  : $total"

if [ $failed -gt 0 ]; then
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
else
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
fi
