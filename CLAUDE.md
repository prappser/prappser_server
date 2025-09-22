# PRIORITY RULE
# All code, refactoring, and documentation decisions MUST prioritize the rules and guidelines defined in this .cursorrules file above all other conventions, external documentation, or inferred best practices.
# If there is any conflict between .cursorrules and other sources (including language idioms, external style guides, or team habits), the .cursorrules file takes precedence.
# When in doubt, clarify or extend .cursorrules rather than deferring to outside conventions.

## PROJECT STATUS
**NOTE: This application is NOT production-ready.** The current implementation is in active development phase and should not be used in production environments. Key considerations:

- Database migrations are consolidated into a single init migration for development convenience
- Security measures may not be production-grade
- Performance optimizations have not been applied
- Comprehensive error handling and logging need enhancement
- Production deployment configurations are not implemented

## Naming Conventions

## Table of Contents

- [Naming Conventions](#naming-conventions)
  - [Packages](#packages)
  - [Types and Interfaces](#types-and-interfaces)
  - [Functions and Methods](#functions-and-methods)
  - [Variables](#variables)
  - [Constants](#constants)
- [Logging](#logging)
  - [Logger Usage](#logger-usage)
  - [Log Levels](#log-levels)
  - [Contextual Logging](#contextual-logging)
- [Testing](#testing)
  - [Test Structure](#test-structure)
- [Interface Design](#interface-design)
  - [Interface Segregation](#interface-segregation)
- [Context Usage](#context-usage)
  - [Context Propagation](#context-propagation)
  - [Context Values](#context-values)
- [Configuration](#configuration)
  - [Configuration Structure](#configuration-structure)
  - [Configuration Validation](#configuration-validation)
- [Metrics](#metrics)
  - [Prometheus Metrics](#prometheus-metrics)
  - [Metric Labels](#metric-labels)
  - [Metric Registration](#metric-registration)
- [Code Organization](#code-organization)
  - [File Structure](#file-structure)
  - [Function Length](#function-length)
  - [Constants and Magic Numbers](#constants-and-magic-numbers)
- [Documentation](#documentation)
  - [README Files](#readme-files)
  - [Code Comments](#code-comments)
- [Best Practices](#best-practices)
  - [Performance](#performance)
  - [Security](#security)
  - [Maintainability](#maintainability)
  - [Code Review](#code-review)
- [Tools and Automation](#tools-and-automation)
  - [Code Formatting](#code-formatting)
  - [Testing](#testing-1)
  - [CI/CD](#cicd)

### Packages
- Use lowercase names
- Underscores are allowed but try to think about single word name mostly
- Examples: `auction`, `config`, `exchange`, `bid_filtering`

### Types and Interfaces
- Use PascalCase for exported types
- Use descriptive names that indicate purpose
- Interface names should end with the capability they provide
- Examples: `AuctionService`, `BidderClient`, `CacheClient`

### Functions and Methods
- Use PascalCase for exported functions
- Use camelCase for unexported functions
- Method names should be descriptive and action-oriented
- Examples: `CreateAuctionService`, `RequestBid`, `ValidateRequest`

### Variables
- Use camelCase
- Use descriptive names
- Avoid abbreviations unless widely understood
- Examples: `auctionRequest`, `bidResponse`, `cacheClient`

### Constants
- Use PascalCase for exported constants
- Use lowerCase for internal constants
- Examples: `TypeJSON`, `TypeXML`, `defaultTimeout`

## Testing

### Test Structure
- Place tests in `*_test.go` files in the same package
- Use descriptive test names that explain the scenario
- Follow the pattern: `Test[FunctionName]_[Scenario]`
- Try to create multiple separate test cases instead of using `t.Run`

```go
func TestAuctionService_ProcessRequest_ShouldProcessValidRequest(t *testing.T) {
    // given
    service := NewAuctionService()
    request := &Request{ID: "test-123"}
    
    // when
    result, err := service.ProcessRequest(context.Background(), request)
    
    // then
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

## Interface Design

### Interface Segregation
- Keep interfaces small and focused
- Define interfaces close to where they are used
- By default avoid interfaces and only introduce them if there is a need other than for testing

```go
type AuctionService interface {
    RunExchange(ctx context.Context, request *Request) (*Response, error)
    ValidateRequest(request *Request) error
}

type CacheClient interface {
    Cache(ctx context.Context, items []Cacheable) []string
}
```

## Context Usage

### Context Propagation
- Pass `context.Context` as the first parameter to functions
- Use context for cancellation, timeouts, and request-scoped data
- Don't store context in structs

```go
func (s *Service) ProcessRequest(ctx context.Context, req *Request) error {
    // Use context for cancellation
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // Use context for timeouts
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    return s.doProcess(ctx, req)
}
```

### Context Values
- Use context for request-scoped data (trace ID, user ID, etc.)
- Don't use context for optional parameters
- Use typed context keys

```go
type contextKey string

const (
    userIDKey contextKey = "user_id"
    traceIDKey contextKey = "trace_id"
)

func WithUserID(ctx context.Context, userID string) context.Context {
    return context.WithValue(ctx, userIDKey, userID)
}

func UserIDFromContext(ctx context.Context) (string, bool) {
    userID, ok := ctx.Value(userIDKey).(string)
    return userID, ok
}
```

## Configuration

### Configuration Structure
- Use `mapstructure` tags for configuration binding
- Provide sensible defaults
- Use nested structures for complex configurations
- Try to put config structs in a packages related to particular config

```go
type Config struct {
    Host string `mapstructure:"host"`
    Port int    `mapstructure:"port"`
    
    Cache struct {
        Host   string `mapstructure:"host"`
        Scheme string `mapstructure:"scheme"`
    } `mapstructure:"cache"`
    
    Adapters map[string]AdapterConfig `mapstructure:"adapters"`
}
```

### Configuration Validation
- Validate configuration on startup
- Return meaningful error messages
- Use environment variables for sensitive data

```go
func (c *Config) validate() []error {
    var errors []error
    
    if c.Host == "" {
        errors = append(errors, errors.New("host is required"))
    }
    
    if c.Port <= 0 {
        errors = append(errors, errors.New("port must be positive"))
    }
    
    return errors
}
```

## Code Organization

### File Structure
- Keep files focused on a single responsibility
- Use descriptive file names
- Group related functionality together
- Create subpackages only if there is too many files in a package

### Function Length
- Keep functions short and focused
- Extract complex logic into separate functions
- Use descriptive function names

### Constants and Magic Numbers
- Define constants for magic numbers and strings
- Use descriptive constant names
- Group related constants together

```go
const (
    DefaultTimeout = 30 * time.Second
    MaxBidCount    = 100
    CacheTTL       = 24 * time.Hour
)
```

## Documentation

### README Files
- Include setup instructions
- Document configuration options

### Code Comments
- Put comments **ONLY** for non-obvious code decisions
- Avoid comments when possible even for exported functions or types

## Best Practices

### Performance
- Use appropriate data structures
- Avoid unnecessary allocations
- Use goroutines for concurrent operations
- Profile code for performance bottlenecks

### Security
- Validate all input data
- Use HTTPS for external communications
- Follow security best practices for handling sensitive data

### Maintainability
- Write self-documenting code
- Use consistent formatting (gofmt)
- Follow the single responsibility principle
- Keep dependencies minimal and up to date

### Code Review
- All code changes require review
- Focus on correctness, performance, and maintainability
- Ensure adequate test coverage
- Check for security vulnerabilities

## Tools and Automation

### Code Formatting
- Use `gofmt` for code formatting
- Configure your editor to format on save

### Testing
- Run tests with `go test ./...`
- Use `-race` flag for race condition detection

### CI/CD
- All changes must pass CI checks
- Include automated testing
- Use semantic versioning for releases