#!/bin/bash
set -e

echo "========================================"
echo "kubecrsh Enhanced Monitoring Stack"
echo "========================================"
echo ""

if [ ! -f "./bin/kubecrsh" ]; then
    echo "Building kubecrsh..."
    make build
    echo ""
fi

echo "[1/4] Starting Prometheus + Loki + Grafana..."
docker-compose -f monitoring/docker-compose.monitoring.yml up -d

echo ""
echo "Waiting for services (10s)..."
sleep 10

echo ""
echo "[2/4] Starting kubecrsh daemon..."
./bin/kubecrsh daemon --http-addr :8080 &
DAEMON_PID=$!

sleep 3

echo ""
echo "[3/4] Running health checks..."

if curl -s http://localhost:8080/metrics | grep -q kubecrsh; then
    echo "   kubecrsh metrics OK"
else
    echo "   kubecrsh metrics failed"
fi

if curl -s http://localhost:9090/-/healthy > /dev/null 2>&1; then
    echo "   Prometheus OK"
else
    echo "   Prometheus not ready"
fi

if curl -s http://localhost:3100/ready > /dev/null 2>&1; then
    echo "   Loki OK"
else
    echo "   Loki not ready"
fi

if curl -s http://localhost:3000/api/health > /dev/null 2>&1; then
    echo "   Grafana OK"
else
    echo "   Grafana not ready (may need more time)"
fi

echo ""
echo "[4/4] Generating test crash data..."
echo "   Creating 8 test pods..."

for i in {1..8}; do
    REASON=$((i % 3))
    case $REASON in
        0) EXIT_CODE=137; REASON_NAME="OOMKilled" ;;
        1) EXIT_CODE=1; REASON_NAME="Error" ;;
        2) EXIT_CODE=2; REASON_NAME="Error" ;;
    esac
    
    kubectl run test-crash-$i --image=busybox --restart=Never -- sh -c "echo 'Fatal error in test pod $i'; exit $EXIT_CODE" 2>/dev/null || true
    echo "   - test-crash-$i ($REASON_NAME)"
    sleep 1
done

echo ""
echo "Waiting for crash detection (15s)..."
sleep 15

CRASHES=$(ls -1 reports/*.json 2>/dev/null | wc -l | tr -d ' ')
echo "   Detected $CRASHES crashes"

echo ""
echo "========================================"
echo "Monitoring Stack Ready"
echo "========================================"
echo ""
echo "Grafana Dashboard:"
echo "   URL:      http://localhost:3000"
echo "   Username: admin"
echo "   Password: admin"
echo "   Dashboard: kubecrsh Pod Crash Analytics"
echo ""
echo "Components:"
echo "   Prometheus: http://localhost:9090"
echo "   Loki:       http://localhost:3100"
echo "   Metrics:    http://localhost:8080/metrics"
echo ""
echo "Press Ctrl+C to stop all services..."
echo ""

cleanup() {
    echo ""
    echo "Shutting down..."
    echo "   Stopping kubecrsh daemon..."
    kill $DAEMON_PID 2>/dev/null || true
    echo "   Stopping Docker containers..."
    docker-compose -f docker-compose.monitoring.yml down
    echo "   Cleaning up test pods..."
    kubectl delete pod --selector=run 2>/dev/null || true
    echo "Stopped"
    exit 0
}

trap cleanup EXIT INT TERM

wait $DAEMON_PID
