# Digital Recipes MVP - Implementation Todo List

## High Priority Foundation

- [x] **Setup project structure** with Go API service and Python parser service directories
- [x] **Create PostgreSQL schema migrations** for users, recipes, canonical_ingredients, recipe_ingredients tables
- [ ] **Write database integration tests** for schema validation and basic CRUD operations

## Medium Priority Core APIs

### Recipe Endpoints
- [ ] **Write tests for GET /recipes endpoint** (recipe list functionality)
- [ ] **Implement GET /recipes endpoint** after tests pass
- [ ] **Write tests for GET /recipes/:id endpoint** (recipe details functionality)
- [ ] **Implement GET /recipes/:id endpoint** after tests pass
- [ ] **Write tests for POST /recipes endpoint** (recipe creation)
- [ ] **Implement POST /recipes endpoint** after tests pass

### AI Processing Pipeline
- [ ] **Write tests for OCR service integration** (text extraction from images)
- [ ] **Implement OCR service integration** in Python parser service
- [ ] **Write tests for LLM processing service** (structured data extraction from text)
- [ ] **Implement LLM processing service** for recipe structuring

## Lower Priority Advanced Features

### Ingredient Management
- [ ] **Write tests for ingredient linking logic** (canonical ingredient matching/creation)
- [ ] **Implement ingredient linking** and canonical ingredient management
- [ ] **Write tests for ingredient override API** (user corrections to ingredient links)

### Recipe Workflow
- [ ] **Write tests for recipe status management** (processing/review_required/published states)
- [ ] **Write tests for recipe update API** (saving reviewed recipes)

### Integration Testing
- [ ] **Write end-to-end integration tests** for complete image → structured recipe pipeline

---

## Notes
- Follow test-first approach: write tests before implementation
- Each checkbox represents a complete, testable unit of work
- Focus on one task at a time, completing tests before moving to implementation