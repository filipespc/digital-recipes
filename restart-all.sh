#!/bin/bash

# Digital Recipes - Robust Service Restart Script
# This script ensures all services are properly stopped and restarted

set -e  # Exit on error
set -u  # Exit on undefined variable

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Project directory
PROJECT_DIR="/home/filipe-carneiro/projects/digital-recipes"
FRONTEND_DIR="${PROJECT_DIR}/frontend"
LOG_DIR="${PROJECT_DIR}/logs"

# Create logs directory if it doesn't exist
mkdir -p "${LOG_DIR}"

# Function to print colored messages
print_msg() {
    local color=$1
    local msg=$2
    echo -e "${color}${msg}${NC}"
}

print_msg "$YELLOW" "======================================"
print_msg "$YELLOW" "Digital Recipes - Service Restart Tool"
print_msg "$YELLOW" "======================================"

# Function to kill processes safely
kill_process_pattern() {
    local pattern=$1
    local desc=$2

    print_msg "$YELLOW" "Stopping ${desc}..."

    # Find PIDs matching pattern
    local pids=$(pgrep -f "$pattern" 2>/dev/null || true)

    if [ -n "$pids" ]; then
        # Try graceful shutdown first
        for pid in $pids; do
            kill $pid 2>/dev/null || true
        done

        # Wait a bit for graceful shutdown
        sleep 2

        # Force kill if still running
        pids=$(pgrep -f "$pattern" 2>/dev/null || true)
        if [ -n "$pids" ]; then
            for pid in $pids; do
                kill -9 $pid 2>/dev/null || true
            done
            print_msg "$RED" "  Force killed ${desc}"
        else
            print_msg "$GREEN" "  Gracefully stopped ${desc}"
        fi
    else
        print_msg "$GREEN" "  ${desc} not running"
    fi
}

# Function to kill processes on specific port
kill_port() {
    local port=$1
    local desc=$2

    print_msg "$YELLOW" "Freeing port ${port} (${desc})..."

    # Get PIDs using the port
    local pids=$(lsof -ti :$port 2>/dev/null || true)

    if [ -n "$pids" ]; then
        # Kill all processes using the port
        echo $pids | xargs -r kill -9 2>/dev/null || true
        print_msg "$GREEN" "  Port ${port} freed"
    else
        print_msg "$GREEN" "  Port ${port} already free"
    fi
}

# Function to wait for service
wait_for_service() {
    local url=$1
    local service_name=$2
    local max_attempts=30
    local attempt=0

    print_msg "$YELLOW" "Waiting for ${service_name}..."

    while [ $attempt -lt $max_attempts ]; do
        if curl -f -s "$url" > /dev/null 2>&1; then
            print_msg "$GREEN" "  ${service_name} is ready!"
            return 0
        fi

        attempt=$((attempt + 1))
        echo -n "."
        sleep 1
    done

    echo ""
    print_msg "$RED" "  ${service_name} failed to start after ${max_attempts} seconds"
    return 1
}

# Step 1: Stop all frontend processes
print_msg "$YELLOW" "\n=== Step 1: Stopping Frontend Services ==="
kill_process_pattern "npm run dev" "npm dev processes"
kill_process_pattern "next-server" "Next.js server"
kill_process_pattern "node.*next" "Next.js node processes"
kill_port 3000 "Frontend port 3000"
kill_port 3001 "Frontend port 3001"
kill_port 3002 "Frontend port 3002"

# Step 2: Stop all backend services via Docker Compose
print_msg "$YELLOW" "\n=== Step 2: Stopping Backend Services ==="
cd "$PROJECT_DIR"

# Stop and remove all containers
print_msg "$YELLOW" "Stopping Docker Compose services..."
docker-compose down --remove-orphans 2>&1 | tee "${LOG_DIR}/docker-down.log" > /dev/null || {
    print_msg "$RED" "  Warning: docker-compose down had issues (see ${LOG_DIR}/docker-down.log)"
}

# Clean up any orphaned containers
print_msg "$YELLOW" "Cleaning up orphaned containers..."
docker ps -aq --filter "name=digital-recipes" 2>/dev/null | xargs -r docker rm -f 2>/dev/null || true

# Step 3: Source environment variables
print_msg "$YELLOW" "\n=== Step 3: Loading Environment Variables ==="
if [ -f "${PROJECT_DIR}/.env" ]; then
    # Export variables from .env file
    set -a  # Mark all new variables for export
    source "${PROJECT_DIR}/.env"
    set +a  # Turn off auto-export
    print_msg "$GREEN" "  Environment variables loaded from .env"

    # Verify critical variables
    if [ -z "${JWT_SECRET:-}" ]; then
        print_msg "$RED" "  Warning: JWT_SECRET not set, using default"
        export JWT_SECRET="dev-jwt-secret"
    fi

    if [ -z "${INTERNAL_SERVICE_SECRET:-}" ]; then
        print_msg "$RED" "  Warning: INTERNAL_SERVICE_SECRET not set, using default"
        export INTERNAL_SERVICE_SECRET="dev-secret-key"
    fi
else
    print_msg "$RED" "  Warning: .env file not found, using defaults"
    export JWT_SECRET="dev-jwt-secret"
    export INTERNAL_SERVICE_SECRET="dev-secret-key"
    export GEMINI_API_KEY="${GEMINI_API_KEY:-}"
fi

# Step 4: Start backend services
print_msg "$YELLOW" "\n=== Step 4: Starting Backend Services ==="
cd "$PROJECT_DIR"

print_msg "$YELLOW" "Starting Docker Compose services..."
docker-compose up -d 2>&1 | tee "${LOG_DIR}/docker-up.log" || {
    print_msg "$RED" "  Failed to start Docker services"
    print_msg "$RED" "  Check ${LOG_DIR}/docker-up.log for details"
    exit 1
}

# Wait for backend to be ready
wait_for_service "http://localhost:8080/health" "Backend API" || {
    print_msg "$RED" "\nBackend failed to start. Checking logs..."
    docker-compose logs --tail=20 api-service
    exit 1
}

# Step 5: Verify backend services
print_msg "$YELLOW" "\n=== Step 5: Verifying Backend Services ==="
docker-compose ps

# Check individual service health
HEALTH_RESPONSE=$(curl -s http://localhost:8080/health)
if echo "$HEALTH_RESPONSE" | grep -q "healthy"; then
    print_msg "$GREEN" "  Backend API is healthy"
else
    print_msg "$RED" "  Backend API health check failed: $HEALTH_RESPONSE"
fi

# Step 6: Start frontend service
print_msg "$YELLOW" "\n=== Step 6: Starting Frontend Service ==="
cd "$FRONTEND_DIR"

# Check if node_modules exists
if [ ! -d "node_modules" ]; then
    print_msg "$YELLOW" "Installing frontend dependencies..."
    npm install 2>&1 | tee "${LOG_DIR}/npm-install.log" || {
        print_msg "$RED" "  Failed to install dependencies"
        exit 1
    }
fi

# Start frontend on port 3000 specifically
print_msg "$YELLOW" "Starting frontend on port 3000..."
nohup npm run dev -- --port 3000 > "${LOG_DIR}/frontend.log" 2>&1 &
FRONTEND_PID=$!

# Store the PID for later reference
echo $FRONTEND_PID > "${PROJECT_DIR}/.frontend.pid"
print_msg "$GREEN" "  Frontend started with PID: $FRONTEND_PID"

# Wait for frontend to be ready
wait_for_service "http://localhost:3000" "Frontend" || {
    print_msg "$RED" "\nFrontend failed to start. Last 20 lines of log:"
    tail -20 "${LOG_DIR}/frontend.log"
    exit 1
}

# Step 7: Final verification
print_msg "$YELLOW" "\n=== Step 7: Final Verification ==="

# Check all critical ports
for port in 3000 8080 8081 5432 6379; do
    if lsof -i :$port > /dev/null 2>&1; then
        print_msg "$GREEN" "  ✓ Port $port is active"
    else
        print_msg "$RED" "  ✗ Port $port is not active"
    fi
done

# Display service URLs
print_msg "$GREEN" "\n======================================"
print_msg "$GREEN" "All services started successfully!"
print_msg "$GREEN" "======================================"
print_msg "$GREEN" "\nService URLs:"
print_msg "$GREEN" "  Frontend:       http://localhost:3000"
print_msg "$GREEN" "  Backend API:    http://localhost:8080"
print_msg "$GREEN" "  Parser Service: http://localhost:8081"
print_msg "$GREEN" "\nLogs:"
print_msg "$GREEN" "  Frontend:       ${LOG_DIR}/frontend.log"
print_msg "$GREEN" "  Backend:        docker-compose logs -f api-service"
print_msg "$GREEN" "  Parser:         docker-compose logs -f parser-service"
print_msg "$GREEN" "\nTo stop all services, run:"
print_msg "$GREEN" "  ${PROJECT_DIR}/stop-all.sh"

# Create a companion stop script
cat > "${PROJECT_DIR}/stop-all.sh" << 'EOF'
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
EOF

chmod +x "${PROJECT_DIR}/stop-all.sh"

print_msg "$GREEN" "\nRestart complete!"