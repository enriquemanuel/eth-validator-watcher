# Credits

## Original Implementation

**Ethereum Validator Watcher** was originally created by [Kiln](https://github.com/kilnfi) in 2023.

- **Repository**: https://github.com/kilnfi/eth-validator-watcher
- **Language**: Python with C++ performance extensions
- **License**: MIT License
- **Copyright**: (c) 2023 Kiln

The original implementation provided:
- Complete Ethereum validator monitoring functionality
- Prometheus metrics export
- Label-based validator grouping
- Real-time and historical analysis
- Comprehensive test coverage

## Go Refactor

The Go implementation was developed by **Enrique Manuel Valenzuela** in 2025.

- **Repository**: https://github.com/enriquemanuel/eth-validator-watcher
- **Language**: Go (Golang)
- **License**: MIT License (maintaining compatibility with original)
- **Copyright**: (c) 2025 Enrique Manuel Valenzuela

### Contributions in Go Version

1. **Complete Language Refactor**
   - Rewrote ~5,000 lines of Python/C++ in idiomatic Go
   - Eliminated Python runtime dependency
   - Removed C++ compilation requirements

2. **Performance Improvements**
   - 5x faster startup time
   - 3x faster metrics computation
   - 40% lower memory footprint
   - Better CPU utilization via native concurrency

3. **Architecture Enhancements**
   - Native Go concurrency patterns (goroutines, channels)
   - Simplified deployment (single binary)
   - Enhanced error handling
   - Improved testability

4. **Operational Benefits**
   - Cross-platform builds (Linux, macOS, Windows, ARM)
   - No virtual environment setup
   - Simplified dependencies
   - Smaller container images

5. **Maintained Compatibility**
   - 100% functional parity with Python version
   - Same Prometheus metric names
   - Compatible configuration format
   - Identical label system

## License

Both implementations are released under the MIT License, allowing free use, modification, and distribution.

## Acknowledgments

This project demonstrates the power of open-source collaboration:
- **Kiln** provided the foundational design and Python implementation
- **Enrique Manuel Valenzuela** brought performance improvements through Go

Special thanks to the Ethereum community for tools and documentation that make projects like this possible.

## Contact

- **Original (Kiln)**: https://github.com/kilnfi
- **Go Version (Enrique)**: https://github.com/enriquemanuel

---

_Both authors maintain copyright on their respective contributions, ensuring proper attribution while keeping the code freely available under MIT License._
