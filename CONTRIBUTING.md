# Contributing to Poltergeist 👻

First off, thank you for considering contributing to Poltergeist! 

## How Can I Contribute?

### 🐛 Reporting Bugs

- Use the GitHub issue tracker
- Describe the bug clearly
- Include steps to reproduce
- Add your Go version and OS

### 💡 Suggesting Features

- Open an issue with `[Feature]` prefix
- Explain the use case
- Provide examples if possible

### 🔧 Pull Requests

1. Fork the repo
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Ensure all tests pass (`go test ./...`)
5. Run linter (`golangci-lint run`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

### 📝 Code Style

- Follow standard Go conventions
- Run `go fmt` before committing
- Add comments for exported functions
- Keep functions small and focused

### 🧪 Testing

```bash
# Run all tests
go test ./...

# Run with race detection
go test -race ./...

# Run with coverage
go test -cover ./...
```

## Development Setup

```bash
git clone https://github.com/poltergeist-framework/poltergeist.git
cd poltergeist
go mod download
go build ./...
```

## Questions?

Feel free to open an issue with `[Question]` prefix!

---

Thank you for making Poltergeist better! 👻

