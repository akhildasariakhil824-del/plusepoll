#!/bin/bash
set -e

echo "=================================================="
echo "      🚀 PulsePoll Deployment & Live Tunnel       "
echo "=================================================="

# Check for Docker
if command -v docker >/dev/null 2>&1; then
    echo "✔ Docker detected. Starting containers..."
    docker compose down --remove-orphans 2>/dev/null || true
    docker compose up --build -d
    echo "✔ Local services started successfully!"
    echo "  - Frontend: http://localhost:3000"
    echo "  - Backend:  http://localhost:8080"
    echo "  - Health:   http://localhost:8080/api/health"
else
    echo "ℹ Docker not detected or not running. You can run backend & frontend directly."
fi

echo ""
echo "--------------------------------------------------"
echo "Generating Public Host Link for live demo..."
echo "--------------------------------------------------"

if command -v cloudflared >/dev/null 2>&1; then
    echo "Starting Cloudflare Quick Tunnel on port 3000..."
    cloudflared tunnel --url http://localhost:3000
elif command -v lt >/dev/null 2>&1; then
    echo "Starting localtunnel on port 3000..."
    lt --port 3000
elif command -v ngrok >/dev/null 2>&1; then
    echo "Starting ngrok on port 3000..."
    ngrok http 3000
else
    echo "To generate a live public URL immediately on your machine, run:"
    echo "  npx localtunnel --port 3000"
    echo "Or install cloudflared:"
    echo "  brew install cloudflared && cloudflared tunnel --url http://localhost:3000"
    echo ""
    echo "For permanent 24/7 cloud hosting:"
    echo "  1. Push this repo to GitHub."
    echo "  2. Connect to Render.com (via render.yaml) or Railway.app."
    echo "  3. Use free-tier MongoDB Atlas & Upstash Redis."
fi
