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

### Full Stack Startup
```bash
# Start backend services first
docker-compose up -d

# Then start frontend
cd frontend && npm run dev > ../frontend.log 2>&1 &
```