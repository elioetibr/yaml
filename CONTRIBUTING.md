# Contributing to github.com/elioetibr/yaml (Fork)

Thank you for your interest in contributing to this YAML library fork!

## Development Setup

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/yaml.git
   cd yaml
   ```
3. Install development tools:
   ```bash
   make install-tools
   ```

## Making Changes

### Branch Naming

- Feature branches: `feature/your-feature-name`
- Bug fixes: `fix/issue-description`
- Documentation: `docs/what-you-updated`

### Commit Messages

We follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` New feature (triggers minor version bump)
- `fix:` Bug fix (triggers patch version bump)
- `docs:` Documentation changes
- `test:` Test updates
- `refactor:` Code refactoring
- `perf:` Performance improvements
- `ci:` CI/CD changes
- `chore:` Maintenance tasks

Breaking changes: Add `!` after type or include `BREAKING CHANGE:` in footer.

Examples:
```
feat: add blank line preservation feature
fix: correct parser edge case handling
feat!: change API for Node structure
```

### Testing

Run tests before submitting:
```bash
# All tests
make test

# Specific feature tests
make test-feature

# Benchmarks
make bench

# Full CI simulation
make ci-local
```

### Code Quality

Ensure your code passes all checks:
```bash
# Format code
make fmt

# Run linters
make lint

# Run vet
make vet
```

## Pull Request Process

1. **Create a PR** using the template
2. **Fill out all sections** of the PR template
3. **Ensure CI passes** - All GitHub Actions must be green
4. **Wait for review** - A maintainer will review your PR
5. **Address feedback** - Make requested changes
6. **Merge** - Once approved, your PR will be merged

### PR Title Format

Your PR title should follow the commit message format:
- `feat: add new feature`
- `fix: resolve issue with...`
- `docs: update README`

## Version Bumping

Version bumping is **automatic** based on your commit messages:

- `feat:` → Minor version (3.x.0)
- `fix:` → Patch version (3.0.x)
- Breaking changes → Major version (4.0.0)

For manual version bumping (maintainers only):
```bash
make version-patch  # 3.0.x
make version-minor  # 3.x.0
make version-major  # x.0.0
```

## CI/CD Pipeline

Our CI/CD pipeline runs:

1. **Linting** - Code style and quality checks
2. **Testing** - Across multiple Go versions and OS
3. **Security** - Vulnerability scanning
4. **Coverage** - Code coverage reporting
5. **Version Check** - Determines if version bump needed
6. **Release** - Automatic release on tag push

### Local CI Testing

Test the full CI pipeline locally:
```bash
make ci-local
```

## Feature Development

For the blank line preservation feature:

1. **Enable the feature**:
   ```go
   yaml.PreserveBlankLines = true
   ```

2. **Test your changes**:
   ```bash
   make test-blank-lines
   ```

3. **Benchmark impact**:
   ```bash
   make bench-compare
   ```

## Documentation

- Update code comments for exported functions
- Update README if adding new features
- Add examples for new functionality
- Update CHANGELOG.md (automated on release)

## Getting Help

- Open an issue for bugs
- Start a discussion for features
- Check existing issues before creating new ones

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help others learn and grow

## Recognition

Contributors will be recognized in:
- Release notes
- CONTRIBUTORS file
- GitHub insights

Thank you for contributing!