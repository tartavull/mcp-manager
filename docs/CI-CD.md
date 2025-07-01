# CI/CD Documentation

## Overview

This project uses GitHub Actions for continuous integration and deployment with Nix flakes for reproducible builds.

## Workflows

### 1. CI/CD Workflow (`ci-cd.yml`)

**Triggers:**
- Push to `main` branch
- Pull requests to `main`
- Git tags matching `v*`

**Jobs:**

#### Test Job
- Runs on Ubuntu and macOS
- Uses Nix to set up the environment
- Runs all tests and checks:
  - Protobuf generation
  - Code formatting
  - Go vet
  - Unit tests
  - Integration tests
  - Test coverage

#### Build Job
- Builds binaries for multiple platforms:
  - Linux (amd64, arm64)
  - macOS (amd64, arm64)
  - Windows (amd64)
- Uses Nix for reproducible builds
- Creates compressed archives (tar.gz for Unix, zip for Windows)

#### Release Job
- Only runs on git tags
- Downloads all build artifacts
- Creates checksums
- Publishes GitHub release with:
  - Binary archives
  - Checksums file
  - Auto-generated release notes
  - Installation instructions

### 2. Release Workflow (`release.yml`)

**Purpose:** Simplifies creating new releases

**Usage:**
1. Go to Actions → Create Release
2. Enter version number (e.g., `0.1.0`)
3. Optionally mark as pre-release
4. Workflow creates tag and triggers CI/CD

## Local Testing

### Test CI Locally

Run the provided script to test CI steps locally:

```bash
./scripts/test-ci-locally.sh
```

This script:
- Uses Nix development shell
- Runs all CI checks
- Tests cross-compilation
- Provides immediate feedback

### Manual Testing

Test individual steps:

```bash
# Enter Nix shell
nix develop

# Run specific checks
make fmt-check
make vet
make test
make build
```

## Creating a Release

### Method 1: GitHub Actions (Recommended)

1. Go to [Actions](../../actions) → Create Release
2. Click "Run workflow"
3. Enter version (e.g., `0.1.0`)
4. Submit

The workflow will:
- Create a git tag
- Trigger the CI/CD pipeline
- Build and publish release

### Method 2: Manual Tag

```bash
# Create annotated tag
git tag -a v0.1.0 -m "Release v0.1.0"

# Push tag
git push origin v0.1.0
```

## Nix Builds

### Update Vendor Hash

When dependencies change, update the vendor hash:

```bash
./scripts/update-vendor-hash.sh
```

### Build with Nix

```bash
# Build all packages
nix build

# Build specific package
nix build .#mcp-daemon
nix build .#mcp-manager

# Result will be in ./result/bin/
```

## Secrets Configuration

### Required Secrets

1. **GITHUB_TOKEN**: Automatically provided by GitHub Actions

2. **CACHIX_AUTH_TOKEN** (Optional): For Nix cache
   - Sign up at [cachix.org](https://cachix.org)
   - Create a binary cache
   - Generate auth token
   - Add as repository secret

## Troubleshooting

### Build Failures

1. **Protobuf generation fails**
   ```bash
   make proto
   git add internal/grpc/pb/
   git commit -m "Update generated protobuf code"
   ```

2. **Vendor hash mismatch**
   ```bash
   ./scripts/update-vendor-hash.sh
   git add flake.nix
   git commit -m "Update vendor hash"
   ```

3. **Format check fails**
   ```bash
   make fmt
   git add .
   git commit -m "Format code"
   ```

### Release Issues

1. **Tag already exists**
   - Delete local tag: `git tag -d v0.1.0`
   - Delete remote tag: `git push origin :refs/tags/v0.1.0`
   - Create new tag

2. **Release assets missing**
   - Check build job logs
   - Ensure all platforms built successfully
   - Verify artifact upload steps

## Best Practices

1. **Before Pushing**
   - Run `./scripts/test-ci-locally.sh`
   - Fix any issues locally
   - Commit all changes

2. **Version Numbering**
   - Use semantic versioning (MAJOR.MINOR.PATCH)
   - Examples: `0.1.0`, `1.0.0`, `1.2.3`

3. **Release Notes**
   - GitHub auto-generates from commits
   - Add manual notes for major features
   - Include breaking changes clearly

4. **Testing Releases**
   - Use pre-release flag for beta versions
   - Test installation instructions
   - Verify checksums 