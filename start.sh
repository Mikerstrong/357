#!/usr/bin/env bash
set -e

# Load .env if present
if [ -f .env ]; then
  export $(grep -v '^#' .env | xargs)
fi

# Install frontend deps if needed
if [ ! -d frontend/node_modules ]; then
  echo "Installing frontend dependencies..."
  cd frontend && npm install && cd ..
fi

# Start Go backend in background
echo "Starting Go backend on :${PORT:-8080}..."
cd backend
go run . &
BACKEND_PID=$!
cd ..

# Trap to kill backend when script exits
trap "kill $BACKEND_PID 2>/dev/null" EXIT

echo "Starting Vite dev server on :5173..."
cd frontend
npm run dev
