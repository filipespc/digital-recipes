# Digital Recipes

A modern digital recipe management application.

## Overview

Digital Recipes is a comprehensive platform for managing, organizing, and sharing your favorite recipes. Built with modern web technologies to provide an intuitive and efficient cooking companion.

## Features

- Recipe management and organization
- Search and filtering capabilities
- User-friendly interface
- Recipe sharing functionality

## Getting Started

### Prerequisites
- Docker and Docker Compose
- Go 1.21+ (for local development)
- Python 3.11+ (for local development)

### Environment Setup

1. Copy the environment template:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` file with your configuration:
   ```bash
   # Required
   POSTGRES_PASSWORD=your_secure_password_here
   
   # Optional (defaults provided)
   POSTGRES_DB=digital_recipes
   POSTGRES_USER=dbuser
   GIN_MODE=release
   ```

### Running the Application

Start all services:
```bash
make run
# or
docker-compose up --build
```

Services will be available at:
- API Service: http://localhost:8080
- Parser Service: http://localhost:8081
- PostgreSQL: localhost:5432

### Development

Build services locally:
```bash
make build
```

Run tests:
```bash
make test
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License.