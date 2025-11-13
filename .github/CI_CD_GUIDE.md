# CI/CD Guide

## Workflows Overview

This repository has 3 automated workflows:

### 1. PR Validation (`pr-validation.yaml`)
**Triggers:** Every pull request to main/develop

**What it does:**
- ✅ **Linting** - golangci-lint, go vet, go fmt check
- ✅ **Testing** - Unit tests with race detection + coverage
- ✅ **Build** - Multi-platform compilation (linux/darwin, amd64/arm64)
- ✅ **Docker** - Test Docker image builds
- ✅ **Helm** - Validate Helm chart syntax
- ✅ **Security** - Trivy vulnerability scan

**Purpose:** Prevent bad code from being merged

### 2. Build & Push (`build.yaml`)
**Triggers:** Push to main/develop branches

**What it does:**
- ✅ Linting
- ✅ Testing
- ✅ Builds multi-arch Docker image
- ✅ Pushes to ghcr.io with branch tags

**Images created:**
- `ghcr.io/enriquemanuel/eth-validator-watcher:main`
- `ghcr.io/enriquemanuel/eth-validator-watcher:develop`
- `ghcr.io/enriquemanuel/eth-validator-watcher:main-sha123abc`

### 3. Release (`release.yaml`)
**Triggers:** Push version tag (v1.0.0, v1.2.3, etc.)

**What it does:**
1. Builds multi-arch Docker image
2. Pushes to ghcr.io with version tags
3. Updates Helm chart version
4. Commits Helm chart changes
5. Publishes Helm chart to GitHub Pages
6. Creates GitHub Release with notes

**Artifacts created:**
- Docker: `ghcr.io/enriquemanuel/eth-validator-watcher:1.0.0`
- Docker: `ghcr.io/enriquemanuel/eth-validator-watcher:1.0`
- Docker: `ghcr.io/enriquemanuel/eth-validator-watcher:1`
- Docker: `ghcr.io/enriquemanuel/eth-validator-watcher:latest`
- Helm: Published to https://enriquemanuel.github.io/eth-validator-watcher
- Release: https://github.com/enriquemanuel/eth-validator-watcher/releases

## Development Workflow

### Making Changes

```bash
# 1. Create feature branch
git checkout -b feature/my-feature

# 2. Make changes, ensure code quality
go fmt ./...
go vet ./...
go test ./...

# 3. Commit and push
git add .
git commit -m "feat: add my feature"
git push origin feature/my-feature

# 4. Create PR on GitHub
# PR validation workflow will run automatically
```

### Releasing a New Version

```bash
# 1. Ensure main branch is up to date
git checkout main
git pull

# 2. Decide version number (Semantic Versioning)
# - v1.0.0 → v1.0.1 (patch - bug fixes)
# - v1.0.0 → v1.1.0 (minor - new features)
# - v1.0.0 → v2.0.0 (major - breaking changes)

# 3. Create annotated tag
git tag -a v1.1.0 -m "Release v1.1.0 - Description of changes"

# 4. Push tag (triggers release workflow)
git push origin v1.1.0

# 5. Monitor workflow
# Go to: https://github.com/enriquemanuel/eth-validator-watcher/actions

# 6. Verify release
# Check: https://github.com/enriquemanuel/eth-validator-watcher/releases
```

## Code Quality Standards

All PRs must pass:

1. **Formatting** - `go fmt ./...` (automatic formatting)
2. **Linting** - golangci-lint (catches common issues)
3. **Vetting** - `go vet ./...` (finds suspicious code)
4. **Tests** - All tests pass with race detection
5. **Build** - Compiles for all platforms
6. **Docker** - Docker image builds successfully
7. **Helm** - Helm chart validates
8. **Security** - No critical vulnerabilities

### Running Checks Locally

```bash
# Format code
go fmt ./...

# Run linter (install first: https://golangci-lint.run/usage/install/)
golangci-lint run ./...

# Run vet
go vet ./...

# Run tests
go test -v -race ./...

# Build
go build -o build/eth-validator-watcher ./cmd/watcher

# Test Docker build
docker build -t eth-validator-watcher:local .

# Validate Helm
helm lint charts/eth-validator-watcher
```

## Workflow Permissions

### Required Repository Settings

**1. Actions Permissions:**
- Settings → Actions → General
- Workflow permissions: "Read and write permissions"
- Allow GitHub Actions to create pull requests: ✅

**2. GitHub Pages:**
- Settings → Pages
- Source: "Deploy from a branch"
- Branch: "gh-pages"

**3. Branch Protection (Recommended):**
- Settings → Branches → Add rule for `main`
- Require pull request before merging: ✅
- Require status checks: ✅
  - lint
  - test
  - build
  - docker-build
  - helm-lint

## Troubleshooting

### Workflow fails on lint

```bash
# Fix locally
go fmt ./...
golangci-lint run --fix ./...
git add .
git commit --amend
git push -f
```

### Tests fail

```bash
# Run tests locally to debug
go test -v ./...

# With race detection
go test -v -race ./...

# Specific package
go test -v ./pkg/metrics
```

### Docker build fails

```bash
# Test locally
docker build -t test .

# Check Dockerfile syntax
docker build --no-cache -t test .
```

### Helm validation fails

```bash
# Lint chart
helm lint charts/eth-validator-watcher

# Test template rendering
helm template test charts/eth-validator-watcher --debug
```

### Release workflow fails

Check:
1. Tag format is correct (v1.2.3, not 1.2.3)
2. GitHub token has proper permissions
3. Helm chart Chart.yaml is valid
4. No merge conflicts in auto-commit

## Monitoring

- **Workflow runs**: https://github.com/enriquemanuel/eth-validator-watcher/actions
- **Packages**: https://github.com/enriquemanuel/eth-validator-watcher/pkgs/container/eth-validator-watcher
- **Releases**: https://github.com/enriquemanuel/eth-validator-watcher/releases
- **Helm repository**: https://enriquemanuel.github.io/eth-validator-watcher

## Quick Links

- [PR Validation Workflow](.github/workflows/pr-validation.yaml)
- [Build Workflow](.github/workflows/build.yaml)
- [Release Workflow](.github/workflows/release.yaml)
- [Release Process Guide](.github/RELEASE.md)
