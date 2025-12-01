# LibreSeed Testing Documentation

This directory contains testing strategies and documentation for LibreSeed.

## Documents

### [LIBRESEED_TEST_STRATEGY.md](LIBRESEED_TEST_STRATEGY.md)
Comprehensive test strategy covering:
- Testing approach and methodology
- Component-specific test plans (Seeder, Publisher, CLI, Gateways)
- Integration testing strategies
- Performance testing requirements
- Security testing considerations
- Test automation and CI/CD integration

## Testing Levels

LibreSeed employs multiple testing levels:

1. **Unit Tests** - Individual component and function testing
2. **Integration Tests** - Component interaction testing
3. **System Tests** - End-to-end workflow validation
4. **Performance Tests** - Scalability and performance benchmarks
5. **Security Tests** - Vulnerability assessment and penetration testing

## Quick Start

To run tests for specific components:

```bash
# Seeder tests
cd seeder && go test ./...

# Publisher tests
cd publisher && go test ./...

# CLI tests
cd cli && go test ./...

# Gateway tests
cd gateways/npm && npm test
```

## Test Coverage Goals

- **Core Protocol**: 90%+ coverage
- **Seeder**: 85%+ coverage
- **Publisher**: 85%+ coverage
- **CLI**: 80%+ coverage
- **Gateways**: 80%+ coverage

## Related Documentation

- [Protocol Specification](../../spec/LIBRESEED-SPEC-v1.2.md)
- [Architecture Review](../architecture/ARCHITECTURE-REVIEW.md)
- [Implementation Guide](../IMPLEMENTATION_GUIDE.md)
