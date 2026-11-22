#!/bin/bash
set -e
cd "$(dirname "$0")"

echo "=========================================================="
echo "  🚀 Mini Delivery/works Scale Test (1,000,000 Orders)"
echo "=========================================================="

# 1. Build binaries
echo "[1/4] Building engine and loadgen..."
go build -o engine ./cmd/engine
go build -o loadgen ./cmd/loadgen

# 2. Check or generate 1,000,000 orders dataset
DATASET="load/orders-1m.jsonl"
if [ ! -f "$DATASET" ]; then
    echo "[2/4] Generating 1,000,000 orders in $DATASET..."
    ./loadgen -n 1000000 -out "$DATASET"
else
    echo "[2/4] Using existing dataset: $DATASET ($(du -h "$DATASET" | cut -f1))"
fi

# 3. Execute scale benchmark
echo "[3/4] Streaming 1,000,000 orders through engine..."
echo "Command: ./engine data/ < $DATASET | wc -l"

START_TIME=$(python3 -c 'import time; print(time.time())')

# Run engine and count output lines
OUTPUT_LINES=$(./engine data/ < "$DATASET" | wc -l | tr -d ' ')

END_TIME=$(python3 -c 'import time; print(time.time())')
ELAPSED=$(python3 -c "print(f'{$END_TIME - $START_TIME:.2f}')")
THROUGHPUT=$(python3 -c "print(f'{1000000 / ($END_TIME - $START_TIME):,.0f}')")

# 4. Results report
echo ""
echo "=========================================================="
echo "  🏁 Scale Benchmark Results"
echo "=========================================================="
echo "  Total Orders Processed : 1,000,000"
echo "  Output Lines Counted   : $OUTPUT_LINES"
echo "  Total Wall-Clock Time  : ${ELAPSED}s"
echo "  Throughput             : ${THROUGHPUT} orders/second"
echo "  Target Requirement     : < 60.0s (AC-09)"

if [ "$OUTPUT_LINES" -eq 1000000 ] && [ $(python3 -c "print(1 if $ELAPSED < 60.0 else 0)") -eq 1 ]; then
    echo "  Status                 : ✅ PASS (PASSED UNDER 60 SECONDS)"
else
    echo "  Status                 : ❌ FAIL"
    exit 1
fi
echo "=========================================================="
