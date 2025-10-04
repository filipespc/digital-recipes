#!/bin/bash

# Stop all services
PROJECT_DIR="/home/filipe-carneiro/projects/digital-recipes"

echo "Stopping all Digital Recipes services..."

# Stop frontend if PID file exists
if [ -f "${PROJECT_DIR}/.frontend.pid" ]; then
    PID=$(cat "${PROJECT_DIR}/.frontend.pid")
    if kill -0 $PID 2>/dev/null; then
        kill $PID
        echo "Stopped frontend (PID: $PID)"
    fi
    rm "${PROJECT_DIR}/.frontend.pid"
fi

# Kill any remaining frontend processes
pkill -f "npm run dev" 2>/dev/null || true
pkill -f "next-server" 2>/dev/null || true

# Stop backend services
cd "$PROJECT_DIR"
docker-compose down

echo "All services stopped."
