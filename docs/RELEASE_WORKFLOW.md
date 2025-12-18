# Automated Release Workflow

This repository uses an automated release workflow that creates semantic versioned releases based on conventional commit messages.

## How It Works

When you push commits to the `main` branch, the workflow automatically:

1. **Analyzes commit messages** to determine if a release is needed
2. **Calculates the new version** based on commit types (major, minor, or patch)
3. **Builds and pushes Docker images** to GitHub Container Registry
4. **Updates and publishes Helm charts** to GitHub Pages
5. **Creates a Git tag and GitHub release** with generated release notes

## Conventional Commits

The workflow uses [Conventional Commits](https://www.conventionalcommits.org/) to determine version bumps:

### Commit Types

- **`feat:`** - New feature (triggers **minor** version bump: `v1.2.0` → `v1.3.0`)
- **`fix:`** - Bug fix (triggers **patch** version bump: `v1.2.0` → `v1.2.1`)
- **`BREAKING CHANGE:`** or **`feat!:`** / **`fix!:`** - Breaking change (triggers **major** version bump: `v1.2.0` → `v2.0.0`)
- **`perf:`** - Performance improvement (triggers **patch** version bump)
- **`refactor:`** - Code refactoring (triggers **patch** version bump)

### Other Commit Types (No Release)

These commit types do NOT trigger a release:
- `chore:` - Maintenance tasks
- `docs:` - Documentation changes
- `style:` - Code style changes
- `test:` - Test changes
- `ci:` - CI/CD changes

## Commit Message Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Examples

#### Feature (Minor Version Bump)
```bash
git commit -m "feat: add sync committee tracking"
# v1.2.0 → v1.3.0
```

#### Bug Fix (Patch Version Bump)
```bash
git commit -m "fix: resolve slot skipping issue in clock"
# v1.2.0 → v1.2.1
```

#### Breaking Change (Major Version Bump)
```bash
git commit -m "feat!: change config file format

BREAKING CHANGE: Config format changed from YAML to JSON"
# v1.2.0 → v2.0.0
```

#### With Scope
```bash
git commit -m "feat(metrics): add validator type breakdown metrics"
# v1.2.0 → v1.3.0
```

#### No Release
```bash
git commit -m "docs: update README with new metrics"
# No release created
```

## Release Process

### Automatic Release (Recommended)

1. Make your changes and commit using conventional commit format:
   ```bash
   git add .
   git commit -m "feat: add new feature"
   ```

2. Push to main:
   ```bash
   git push origin main
   ```

3. The workflow automatically:
   - Determines the version bump
   - Builds Docker image with new version
   - Updates Helm chart
   - Creates GitHub release
   - Publishes artifacts

### Manual Release

If you need to create a manual release:

```bash
# Create and push a tag manually
git tag -a v1.3.0 -m "Release v1.3.0"
git push origin v1.3.0
```

This will trigger the existing `release.yaml` workflow.

## Workflow Files

- **`.github/workflows/auto-release.yaml`** - Automatic release on push to main
- **`.github/workflows/release.yaml`** - Manual release on tag push
- **`.github/workflows/build.yaml`** - Build and test on feature branches and PRs

## Version Strategy

The workflow follows [Semantic Versioning](https://semver.org/):

- **MAJOR** version (X.0.0) - Incompatible API changes
- **MINOR** version (0.X.0) - Backward-compatible new features
- **PATCH** version (0.0.X) - Backward-compatible bug fixes

## Artifacts Published

Each release publishes:

1. **Docker Images** to GitHub Container Registry:
   - `ghcr.io/enriquemanuel/eth-validator-watcher:1.3.0`
   - `ghcr.io/enriquemanuel/eth-validator-watcher:latest`

2. **Helm Chart** to GitHub Pages:
   ```bash
   helm repo add eth-validator-watcher https://enriquemanuel.github.io/eth-validator-watcher
   helm install my-watcher eth-validator-watcher/eth-validator-watcher --version 1.3.0
   ```

3. **GitHub Release** with:
   - Release notes generated from commits
   - Installation instructions
   - Links to Docker images and Helm charts

## Skipping Workflow

To push changes without triggering the auto-release workflow, include `[skip ci]` in your commit message:

```bash
git commit -m "chore: update Helm chart to version 1.3.0 [skip ci]"
```

## Troubleshooting

### Workflow Not Running

- Check that you're pushing to `main` branch
- Verify GitHub Actions are enabled in repository settings
- Check workflow permissions in Settings → Actions → General

### Version Not Bumping Correctly

- Ensure commit messages follow conventional commit format
- Check workflow logs in Actions tab for version calculation details
- Verify the latest tag exists: `git describe --tags --abbrev=0`

### Docker Push Fails

- Check GitHub Container Registry permissions
- Verify `GITHUB_TOKEN` has `packages: write` permission
- Ensure multi-platform builds are supported

### Helm Chart Not Publishing

- Verify GitHub Pages is enabled for the repository
- Check that the `gh-pages` branch exists
- Review workflow logs for deployment errors

## Best Practices

1. **Use descriptive commit messages** - Help others understand changes
2. **Group related changes** - Make atomic commits
3. **Test before pushing** - Run tests locally first
4. **Review release notes** - Check generated notes after release
5. **Use scopes** - Add context to commits (e.g., `feat(metrics):`)

## Example Workflow

```bash
# 1. Create a feature branch
git checkout -b feature/new-metrics

# 2. Make changes
# ... edit files ...

# 3. Commit with conventional format
git add .
git commit -m "feat(metrics): add block proposal tracking"

# 4. Push feature branch (builds but doesn't release)
git push origin feature/new-metrics

# 5. Create PR and merge to main
# (via GitHub UI)

# 6. Automatic release happens on merge to main!
# - Version bumped to v1.3.0 (from v1.2.0)
# - Docker image built and pushed
# - Helm chart updated and published
# - GitHub release created
```

## Configuration

The workflow can be customized by editing `.github/workflows/auto-release.yaml`:

- Modify version bump logic in the `Calculate new version` step
- Adjust commit type patterns in the `Check if should release` step
- Change Docker platforms, Helm chart paths, or release note format

## References

- [Conventional Commits](https://www.conventionalcommits.org/)
- [Semantic Versioning](https://semver.org/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
