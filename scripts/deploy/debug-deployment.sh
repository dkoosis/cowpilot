#!/bin/bash
# Debug fly.io deployment issues

set -e

echo "1. Checking fly.io app status..."
fly status -a cowpilot

echo -e "\n2. Recent logs (last 50 lines)..."
fly logs -a cowpilot | tail -n 50

echo -e "\n3. Checking secrets..."
fly secrets list -a cowpilot

echo -e "\n4. Testing local Docker build..."
docker build -t cowpilot-test . || {
    echo "Docker build failed!"
    exit 1
}

echo -e "\n5. Running local Docker test..."
docker run -d --name cowpilot-test -p 8080:8080 -e PORT=8080 -e FLY_APP_NAME=test cowpilot-test

sleep 3

echo -e "\n6. Checking if container is running..."
docker ps | grep cowpilot-test || {
    echo "Container crashed! Checking logs..."
    docker logs cowpilot-test
    docker rm -f cowpilot-test
    exit 1
}

echo -e "\n7. Testing health endpoint..."
curl -f http://localhost:8080/health || {
    echo "Health check failed!"
    docker logs cowpilot-test
    docker rm -f cowpilot-test
    exit 1
}

echo -e "\n✓ Local test passed!"
docker rm -f cowpilot-test

echo -e "\n8. Deploying to fly.io with updated Dockerfile..."
fly deploy -a cowpilot
