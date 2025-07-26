#!/bin/bash
# Track test performance and alert on significant changes

METRICS_FILE=".test-metrics.json"
THRESHOLD_PERCENT=20  # Alert if test time changes by more than 20%

# Initialize metrics file if it doesn't exist
if [ ! -f "$METRICS_FILE" ]; then
    echo "{}" > "$METRICS_FILE"
fi

# Parse current test results
if [ ! -f test-results.json ]; then
    echo "No test-results.json found"
    exit 0
fi

# Create temp file for new metrics
TMP_METRICS=$(mktemp)
cp "$METRICS_FILE" "$TMP_METRICS"

echo "Test Performance Report"
echo "======================"
echo ""

# Track changes
ALERTS=0

# Parse each test result
jq -c 'select(.Action == "pass" and .Test != null and .Elapsed != null)' test-results.json | while read -r line; do
    TEST_NAME=$(echo "$line" | jq -r '.Test')
    CURRENT_TIME=$(echo "$line" | jq -r '.Elapsed')
    
    # Get previous time
    PREV_TIME=$(jq -r ".[\"$TEST_NAME\"].avg // 0" "$METRICS_FILE")
    PREV_COUNT=$(jq -r ".[\"$TEST_NAME\"].count // 0" "$METRICS_FILE")
    
    # Calculate new average (rolling average)
    if [ "$PREV_COUNT" -eq 0 ]; then
        NEW_AVG=$CURRENT_TIME
        NEW_COUNT=1
    else
        NEW_AVG=$(echo "scale=3; ($PREV_TIME * $PREV_COUNT + $CURRENT_TIME) / ($PREV_COUNT + 1)" | bc)
        NEW_COUNT=$((PREV_COUNT + 1))
    fi
    
    # Check for significant change
    if [ "$PREV_COUNT" -gt 0 ]; then
        CHANGE=$(echo "scale=2; (($CURRENT_TIME - $PREV_TIME) / $PREV_TIME) * 100" | bc 2>/dev/null || echo "0")
        ABS_CHANGE=$(echo "$CHANGE" | sed 's/-//')
        
        if (( $(echo "$ABS_CHANGE > $THRESHOLD_PERCENT" | bc -l) )); then
            ALERTS=$((ALERTS + 1))
            if (( $(echo "$CHANGE > 0" | bc -l) )); then
                echo "⚠️  SLOWER: $TEST_NAME: ${CURRENT_TIME}s (was ${PREV_TIME}s, +${CHANGE}%)"
            else
                echo "✅ FASTER: $TEST_NAME: ${CURRENT_TIME}s (was ${PREV_TIME}s, ${CHANGE}%)"
            fi
        else
            echo "  $TEST_NAME: ${CURRENT_TIME}s"
        fi
    else
        echo "  $TEST_NAME: ${CURRENT_TIME}s (new test)"
    fi
    
    # Update metrics
    jq ".[\"$TEST_NAME\"] = {avg: $NEW_AVG, count: $NEW_COUNT, last: $CURRENT_TIME, updated: \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}" "$TMP_METRICS" > "${TMP_METRICS}.new"
    mv "${TMP_METRICS}.new" "$TMP_METRICS"
done

# Save updated metrics
mv "$TMP_METRICS" "$METRICS_FILE"

# Tool-specific performance tracking
echo ""
echo "Tool Performance Metrics"
echo "========================"
if [ -f test-results.json ]; then
    # Extract tool-specific tests (assuming naming convention)
    jq -r 'select(.Action == "pass" and .Test != null and (.Test | contains("Tool"))) | "\(.Test): \(.Elapsed)s"' test-results.json | sort -k2 -nr
fi

# Summary
echo ""
if [ $ALERTS -gt 0 ]; then
    echo "🔔 $ALERTS tests showed significant performance changes (>${THRESHOLD_PERCENT}%)"
else
    echo "✓ No significant performance changes detected"
fi

# Show top 5 slowest tests
echo ""
echo "Top 5 Slowest Tests:"
jq -r 'select(.Action == "pass" and .Test != null) | "\(.Test): \(.Elapsed)s"' test-results.json | sort -t: -k2 -nr | head -5
