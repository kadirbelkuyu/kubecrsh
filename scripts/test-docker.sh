#!/bin/bash
set -e

echo "Testing Docker build..."

echo "Building image..."
docker build -t kubecrsh:test .

SIZE=$(docker images kubecrsh:test --format "{{.Size}}")
echo "Image built successfully: $SIZE"

echo ""
echo "Testing container..."
docker run --rm kubecrsh:test --help > /dev/null
echo "Container runs successfully"

echo ""
echo "Cleaning up..."
docker rmi kubecrsh:test

echo "Docker build test complete."
