#!/bin/bash
# Create persistent volume for token storage

echo "Creating Fly.io volume for persistent token storage..."

# Create volume (10MB is plenty for tokens)
fly volumes create cowpilot_data --size 1 --region ewr

echo "Volume created. Deploy with: fly deploy"
echo "Tokens will persist at /data/tokens.db"
