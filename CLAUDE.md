# Digital Recipes - AI-Powered Recipe Hub

## Project Overview
Digital Recipes is an AI-Powered Recipe Hub MVP designed to solve the problem of scattered and disorganized recipe collections. The application automates recipe data entry from images and provides intelligent search capabilities.

## Core Documentation

### Key Project Documents

#### **PRD.md** - Product Requirements Document
- **Purpose**: Defines the complete product vision, user persona, and feature specifications for the MVP
- **Contents**: 
  - Problem statement and user persona ("The Busy Planner")
  - Job-to-be-Done: "Help me consolidate my recipes into one structured, searchable place"
  - MVP scope with guiding principles: "Automate by Default, Allow Correction by Exception"
  - Detailed feature specifications for the core `Save -> Find -> Decide` loop
- **Key Features Defined**:
  - Add Recipe from Image(s) with AI-powered extraction
  - Review & Edit Recipe workflow with structured ingredient linking
  - View Recipe List & Details for consumption

#### **ADR.md** - Architectural Decision Record  
- **Purpose**: Documents key technical and architectural decisions for the MVP implementation
- **Contents**:
  - High-level system design using decoupled services architecture
  - Technology stack decisions (Go + Gin for API, Python for AI parsing, PostgreSQL for data)
  - Core AI workflow: OCR → LLM pipeline for image-to-data conversion
  - Relational data model with structured ingredient management
- **Key Architectural Decisions**:
  - Backend API Service + Recipe Parser Service with async message queue
  - Two-step AI pipeline: OCR for text extraction → LLM for structuring
  - Relational database schema supporting ingredient linking and review workflow

## Development Principles

### Core Loop Focus
The MVP is exclusively focused on the `Save -> Find -> Decide` recipe management loop. All development should prioritize:

1. **Automated Data Entry**: AI handles the heavy lifting of recipe extraction and structuring
2. **Structured Data Foundation**: Every recipe stored in structured format from day one
3. **Review & Correction Workflow**: Users review and correct AI output rather than manual entry

### Technical Approach
- **Decoupled Architecture**: Separate user-facing API from resource-intensive AI processing
- **Async Processing**: Long-running AI tasks don't block user experience
- **Cost-Effective AI Pipeline**: OCR for transcription + text LLM for understanding
- **Relational Data Integrity**: PostgreSQL for recipe-ingredient relationships

## Out of Scope for MVP
The following features are explicitly parked for future versions:
- Smart Search through natural language
- Add recipes from URL
- Manual recipe creation form
- Recipe tagging
- Shopping list generation
- Ingredient usage prediction

## File Structure Context
- `PRD.md`: Complete product specification and user requirements
- `ADR.md`: Technical architecture and implementation decisions  
- `TODO.md`: Implementation roadmap with phase-by-phase development plan and current progress tracking
- `README.md`: Basic project overview and getting started guide
- `CLAUDE.md`: This file - project context and documentation guide

## Development Guidance
When implementing features, always reference both the PRD for user requirements and the ADR for technical implementation approach. The core workflow of ingredient extraction, linking, and review is central to the user experience and technical architecture.

## Development Process
- Test-Driven Development (TDD) Approach:
  - We want to test often to see if we are on track
  - Active participation in test definition is crucial
  - Always start by defining and implementing tests before writing implementation code

## Running the Application

### Backend Services
**IMPORTANT**: Use Docker Compose to run all backend services (API, database, parser):

```bash
# Start all backend services
docker-compose up -d

# Check service status
docker-compose ps

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

**Backend URLs:**
- API Service: http://localhost:8080
- Database: localhost:5432 (PostgreSQL)

### Frontend Server Startup Protocol
**IMPORTANT**: To avoid timeout issues when starting the Next.js development server, always use background execution:

```bash
# Correct way to start the frontend server (avoids timeout)
cd frontend && npm run dev > dev.log 2>&1 &

# Wait for server to start, then test
sleep 3 && curl http://localhost:3000

# Check server logs if needed
tail -f dev.log

# Kill background server when done
pkill -f "npm run dev"
```

**Frontend URLs:**
- Local: http://localhost:3000
- Network: http://192.168.15.107:3000 (if localhost doesn't work)

**IMPORTANT CORS Configuration:**
- Always use ONLY `http://localhost:3000` for ALLOWED_ORIGINS
- Never add additional ports (3001, 3002, etc.) to CORS configuration
- If Next.js starts on a different port due to conflicts, kill the conflicting process instead of adding new origins

### Full Stack Restart Protocol

#### Quick Method (Using Scripts)
**Recommended**: Use the comprehensive restart script that handles everything automatically:

```bash
# Complete restart with health checks and verification
./restart-all.sh

# Or to just stop everything
./stop-all.sh
```

These scripts provide:
- Colored output for better visibility
- Graceful shutdown before force killing
- Automatic environment variable loading from .env
- PID tracking for clean process management
- Comprehensive health checks
- Log file management in `logs/` directory
- Automatic retry logic for service startup
- Port verification and cleanup

#### Manual Method
If you need to restart services manually or the scripts aren't available:

```bash
# 1. Stop all services
docker-compose down
pkill -f "npm run dev"
pkill -f "node.*next.*dev"

# 2. Force kill anything using port 3000
netstat -tulpn 2>/dev/null | grep :3000 | awk '{print $7}' | cut -d'/' -f1 | xargs -r kill -9

# 3. Start backend services first (with env var)
INTERNAL_SERVICE_SECRET=dev-secret-key docker-compose up -d

# 4. Start frontend on port 3000 (force if needed)
cd frontend && npm run dev -- --port 3000 > ../frontend.log 2>&1 &

# 5. Verify services
sleep 3
curl -f http://localhost:8080/health || echo "Backend not ready"
curl -f http://localhost:3000 | head -10 || echo "Frontend not ready"
```

### Service Management Scripts

#### restart-all.sh
Full service restart with comprehensive checks:
1. **Stops all frontend processes** - npm, Next.js, frees ports 3000-3002
2. **Stops backend services** - docker-compose down with orphan cleanup
3. **Loads environment variables** - from .env with fallback defaults
4. **Starts backend services** - with health check waiting
5. **Verifies backend** - checks individual service health
6. **Starts frontend** - ensures port 3000, with PID tracking
7. **Final verification** - checks all critical ports

#### stop-all.sh
Clean shutdown of all services:
- Stops frontend using tracked PID
- Kills any remaining frontend processes
- Stops all Docker services
- Cleans up PID files

### Service URLs
- **Frontend**: http://localhost:3000
- **Backend API**: http://localhost:8080
- **Parser Service**: http://localhost:8081
- **PostgreSQL**: localhost:5432
- **Redis**: localhost:6379

### Log Locations
- **Frontend**: `logs/frontend.log` or check with `tail -f frontend.log`
- **Backend**: `docker-compose logs -f api-service`
- **Parser**: `docker-compose logs -f parser-service`
- **Docker startup**: `logs/docker-up.log`
- **Docker shutdown**: `logs/docker-down.log`

### Troubleshooting

#### Frontend won't start on port 3000
```bash
# Force kill anything on port 3000
lsof -ti :3000 | xargs -r kill -9

# Or use fuser
fuser -k 3000/tcp
```

#### Backend services won't start
```bash
# Check for env variables
cat .env | grep -E "JWT_SECRET|INTERNAL_SERVICE_SECRET|GEMINI_API_KEY"

# Set minimum required env vars
export INTERNAL_SERVICE_SECRET=dev-secret-key
export JWT_SECRET=dev-jwt-secret

# Restart with env vars
INTERNAL_SERVICE_SECRET=dev-secret-key docker-compose up -d
```

#### Check service health
```bash
# Backend health check
curl http://localhost:8080/health

# Check all Docker services
docker-compose ps

# View recent logs
docker-compose logs --tail=50
```