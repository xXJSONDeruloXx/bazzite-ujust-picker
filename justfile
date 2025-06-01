# Development tasks for bazzite-ujust-picker

default:
    @just --list

# Build the application
build:
    go build -o ujust-picker

# Run the application
run:
    go run .

# Clean build artifacts
clean:
    rm -f ujust-picker

# Format code
fmt:
    go fmt ./...

# Create a new release tag and push to GitHub
release version:
    git tag -a {{version}} -m "Release {{version}}"
    git push origin {{version}}

# Local test of GoReleaser without publishing
test-release:
    goreleaser release --snapshot --clean

# Check for GoReleaser configuration issues
check-release:
    goreleaser check

# Run tests
test:
    go test ./...

# Install locally
install: build
    sudo cp ujust-picker /usr/local/bin/

# Create a new tagged release
release version:
    git tag -a v{{version}} -m "Release v{{version}}"
    git push origin v{{version}}
