# Water-Bot Code Style Guide

This document outlines the code style and architecture principles that all developers and AI agents should follow when working on the water-bot project.

## Project Structure

The project follows a clean architecture approach with clear separation of concerns:

```
water-bot/
├── cmd/                  # Executable applications
│   ├── bot/              # Telegram bot executable
│   └── scrapper/         # Data scraping executable
├── internal/             # Private application code
│   ├── api/              # API handlers and interfaces
│   ├── db/               # Database common utilities
│   ├── integration/      # External service integrations
│   ├── repository/       # Data access and persistence
│   ├── entities/         # Core domain entities
│   └── usecases/         # Business logic
├── data/                 # Data storage
│   └── riverdata.db      # SQLite database
└── docs/                 # Documentation files
```

## Architecture Principles

1. **Separation of Concerns**: 
   - Each component should have a single responsibility
   - Keep executables thin, focusing on wiring dependencies

2. **Dependency Flow**:
   - Dependencies should point inward
   - Outer layers can import from inner layers, but not vice versa
   - Flow: API → Usecase → Repository → Entities

3. **Code Organization**:
   - Package by feature, not by layer
   - Keep related code together

## Coding Standards

### General

- Use meaningful variable, function, and package names
- Write short, focused functions (aim for < 20 lines)
- Include comments for complex logic
- Use error handling consistently (no panics in production code)
- Follow the Go proverb: "Don't communicate by sharing memory; share memory by communicating"

### File Structure

- Place related types, functions, and methods in the same file
- Organize methods on types alphabetically
- Start with package declarations, followed by imports, constants, variables, and then functions

### Interfaces

- Keep interfaces small and focused
- Define interfaces where they're used, not where they're implemented
- Follow the Go proverb: "Accept interfaces, return structs"

### Error Handling

- Check all errors
- Use meaningful error messages
- Consider using wrapped errors for context
- Return errors rather than logging and continuing

### Testing

- Write tests for all public functions
- Use table-driven tests where appropriate
- Mock external dependencies for unit tests
- Include integration tests for critical paths

## Commit Guidelines

- Use meaningful commit messages
- Start with an imperative verb (Add, Fix, Update)
- Reference issues where appropriate
- Keep commits focused on a single change

## Documentation

- Document all exported functions, types, and methods
- Include examples for non-trivial APIs
- Update documentation when changing functionality

By following these guidelines, we ensure that the water-bot codebase remains clean, maintainable, and easy to understand for all contributors.