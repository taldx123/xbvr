#!/bin/bash
set -e

# Default registry tag if none is provided
# Examples of usage:
# ./build-local.sh
# ./build-local.sh xbvr:latest
IMAGE_TAG=${1:-"xbvr:latest"}

echo "Building multi-stage docker image..."
echo "This will download all node and go dependencies inside the container."
docker build -f Dockerfile.local -t "$IMAGE_TAG" .

echo "Build successful! The image '$IMAGE_TAG' is now available in your local Docker daemon."
echo "You can run it with: docker run -p 9999:9999 $IMAGE_TAG"
