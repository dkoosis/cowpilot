#!/bin/bash
cd /Users/vcto/Projects/cowpilot
echo "Running make all..."
make all 2>&1 | tee build_output.log
exit_code=${PIPESTATUS[0]}
echo "Build exit code: $exit_code"
exit $exit_code
