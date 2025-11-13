# Release Process

This repository uses GitHub Actions to automatically build, publish, and release new versions.

## Automated Release Workflow

When you push a version tag, the workflow will:

1. **Build multi-arch Docker image** (amd64 + arm64)
2. **Push to GitHub Container Registry** (ghcr.io)
3. **Update Helm chart** with new version
4. **Publish Helm chart** to GitHub Pages
5. **Create GitHub Release** with installation instructions

## Creating a Release

### 1. Ensure Code is Ready

```bash
# Make sure tests pass
make test

# Build locally to verify
make build
```

### 2. Create and Push Version Tag

```bash
# Create a version tag (e.g., v1.0.0, v1.2.3, v2.0.0-beta.1)
git tag -a v1.0.0 -m "Release version 1.0.0"

# Push the tag (this triggers the release workflow)
git push origin v1.0.0
```

### 3. Monitor Workflow

- Go to **Actions** tab in GitHub
- Watch the "Release" workflow execute
- It takes ~5-10 minutes to complete

### 4. Verify Release

After workflow completes:

**Docker Image:**
```bash
docker pull ghcr.io/enriquemanuel/eth-validator-watcher:1.0.0
```

**Helm Chart:**
```bash
helm repo add eth-validator-watcher https://enriquemanuel.github.io/eth-validator-watcher
helm repo update
helm search repo eth-validator-watcher
```

**GitHub Release:**
- Check the [Releases](https://github.com/enriquemanuel/eth-validator-watcher/releases) page
- Release notes are auto-generated

## Version Naming

Follow [Semantic Versioning](https://semver.org/):

- **Major** (v2.0.0): Breaking changes
- **Minor** (v1.1.0): New features, backward compatible
- **Patch** (v1.0.1): Bug fixes, backward compatible
- **Pre-release** (v1.0.0-beta.1): Alpha/Beta versions

## Troubleshooting

**Workflow fails on Docker build:**
- Check Dockerfile syntax
- Verify Go dependencies compile

**Workflow fails on Helm chart:**
- Check Chart.yaml syntax with `helm lint charts/eth-validator-watcher`

**Docker image not accessible:**
- Ensure repository has `packages: write` permission
- Check if ghcr.io token is valid

**Helm chart not appearing:**
- GitHub Pages must be enabled in repository settings
- Set Pages source to "gh-pages" branch

## Manual Publishing (Emergency)

If automated workflow fails:

```bash
# Build and push Docker manually
docker build -t ghcr.io/enriquemanuel/eth-validator-watcher:1.0.0 .
docker push ghcr.io/enriquemanuel/eth-validator-watcher:1.0.0

# Package Helm chart manually
helm package charts/eth-validator-watcher
mv eth-validator-watcher-*.tgz .helm-repo/
helm repo index .helm-repo --url https://enriquemanuel.github.io/eth-validator-watcher
```

## First-Time Setup

### Enable GitHub Packages

1. Go to repository Settings
2. Actions → General
3. Workflow permissions → Read and write permissions

### Enable GitHub Pages

1. Go to repository Settings
2. Pages → Source → "gh-pages" branch
3. Save

### Repository Secrets

No secrets needed! Workflow uses `GITHUB_TOKEN` automatically.
