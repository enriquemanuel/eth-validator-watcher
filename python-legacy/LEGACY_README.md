# Python/C++ Legacy Implementation

This directory contains the original Python/C++ implementation of the Ethereum Validator Watcher.

**Note:** This implementation has been superseded by the Go implementation in the root directory.

## Status: Legacy / Maintenance Mode

The Python version is no longer the primary implementation. Please use the Go version for:
- New deployments
- Better performance (3-5x faster)
- Simpler deployment (single binary)
- Lower resource usage

## Migration

If you're currently using the Python version, see the [MIGRATION_GUIDE.md](../MIGRATION_GUIDE.md) in the root directory for step-by-step instructions to migrate to the Go version.

## Documentation

- Original README: [README.md](README.md)
- Configuration examples: [etc/](etc/)
- Python tests: [tests/](tests/)

## Why This Was Moved

The Go implementation provides:
- **5x faster** startup time
- **3x faster** metrics computation
- **40% lower** memory usage
- **Single binary** deployment (no Python runtime needed)
- **Better concurrency** (no GIL limitations)

## Building (Legacy)

If you still need to use the Python version:

```bash
# Install dependencies
pip install -r requirements.txt

# Build C++ extension
python build.py

# Install
pip install -e .

# Run
eth-validator-watcher --config etc/config.local.yaml
```

## Support

Legacy Python version is in maintenance mode. For support:
- Critical bugs will be fixed
- New features will only be added to Go version
- Migration assistance available via GitHub issues

## When to Use This

You should only use the Python version if:
- You have specific Python integration requirements
- You're migrating gradually and need both versions temporarily
- You're contributing bug fixes to the legacy codebase

**For all other cases, please use the Go implementation in the root directory.**

## Contents

- `eth_validator_watcher/` - Python source code
- `tests/` - Python unit tests
- `docs/` - Python documentation
- `etc/` - Configuration examples
- `build.py` - C++ extension builder
- `pyproject.toml` - Python package configuration
- `requirements.txt` - Python dependencies
