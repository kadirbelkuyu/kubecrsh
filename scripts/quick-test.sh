#!/bin/bash
set -e

echo "Quick local test for kubecrsh"
echo ""

echo "[1/7] Building binary..."
go build -o bin/kubecrsh ./cmd/kubecrsh
echo "Build successful"
echo ""

echo "[2/7] Starting daemon..."
./bin/kubecrsh daemon --http-addr :8080 &
DAEMON_PID=$!

sleep 2

echo "[3/7] Checking health endpoints..."
if curl -s http://localhost:8080/health > /dev/null; then
    echo "/health OK"
else
    echo "/health failed"
    kill $DAEMON_PID 2>/dev/null || true
    exit 1
fi

if curl -s http://localhost:8080/ready > /dev/null; then
    echo "/ready OK"
else
    echo "/ready failed"
    kill $DAEMON_PID 2>/dev/null || true
    exit 1
fi

echo "[4/7] Checking metrics..."
METRICS=$(curl -s http://localhost:8080/metrics | grep kubecrsh | wc -l)
if [ "$METRICS" -gt 0 ]; then
    echo "Metrics endpoint working ($METRICS metrics)"
else
    echo "No metrics found"
    kill $DAEMON_PID 2>/dev/null || true
    exit 1
fi

echo ""
echo "[5/7] Creating test crash pod..."
kubectl run quick-test-crash --image=busybox --restart=Never -- sh -c "exit 1" 2>/dev/null || true

echo ""
echo "Waiting 5 seconds for crash detection..."
sleep 5

echo "[6/7] Checking crash reports..."
REPORTS=$(ls -1 reports/*.json 2>/dev/null | wc -l)
if [ "$REPORTS" -gt 0 ]; then
    echo "Crash detected and reported ($REPORTS reports)"
    echo ""
    echo "Latest report:"
    ls -t reports/*.json | head -1 | xargs cat | jq -r '.Crash | "Pod: \(.PodName), Reason: \(.Reason), ExitCode: \(.ExitCode)"'
else
    echo "Warning: No reports generated (check if pod crashed)"
fi

echo ""
echo "[7/7] Cleaning up..."
kill $DAEMON_PID 2>/dev/null || true
kubectl delete pod quick-test-crash --ignore-not-found 2>/dev/null

echo ""
echo "Quick test complete."
echo ""
echo "To test watch mode:"
echo "  ./bin/kubecrsh watch"
echo ""
echo "To test with Slack:"
echo "  ./bin/kubecrsh daemon --slack-webhook 'YOUR_WEBHOOK_URL'"
