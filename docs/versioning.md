# Versioning and Changelog Management

This document describes the versioning strategy, changelog management, and implementation approach for GoReminder.

## Table of Contents

1. [Versioning Strategy](#versioning-strategy)
2. [Changelog Management](#changelog-management)
3. [Implementation Approaches](#implementation-approaches)
4. [Release Process](#release-process)
5. [CI/CD Integration](#cicd-integration)

---

## Versioning Strategy

### Semantic Versioning (SemVer)

GoReminder follows [Semantic Versioning 2.0.0](https://semver.org/):

**Format**: `MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]`

- **MAJOR**: Breaking changes (incompatible API changes)
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Pre-release Versions

For MVP and early development, use pre-release versions:

- **Release Candidate**: `v1.0.0-rc.1`, `v1.0.0-rc.2`, etc.
- **Alpha**: `v1.0.0-alpha.1` (early development)
- **Beta**: `v1.0.0-beta.1` (feature complete, testing)

### When to Use Release Candidates

**Use RC when:**
- ✅ Approaching MVP milestone
- ✅ Feature complete but needs testing
- ✅ Breaking changes that need validation
- ✅ Preparing for first stable release

**Skip RC when:**
- ❌ Internal development builds
- ❌ Hotfixes for stable releases
- ❌ Minor patches

### Version Progression Example

```
v0.1.0-rc.1  → First release candidate
v0.1.0-rc.2  → Bug fixes in RC
v0.1.0       → First stable release (MVP)
v0.1.1       → Patch release (bug fixes)
v0.2.0       → Minor release (new features)
v1.0.0       → Major release (breaking changes)
```

### Starting Fresh for MVP

Since you're approaching MVP, recommended approach:

1. **Current state**: You have `v0.6.0-rc.1` in CHANGELOG
2. **Recommendation**: Continue with `v0.7.0-rc.1`, `v0.8.0-rc.1`, etc. until MVP
3. **MVP release**: `v1.0.0` (first stable release)
4. **Post-MVP**: Continue with `v1.0.1`, `v1.1.0`, etc.

**Alternative (if you want to reset):**
- Start from `v0.1.0-rc.1` for MVP track
- Use `v1.0.0` for MVP release

---

## Changelog Management

### Format: Keep a Changelog

Follow [Keep a Changelog](https://keepachangelog.com/) format (already implemented):

```markdown
## [Unreleased]

## [v1.0.0] - 2025-01-15
### Added
- Feature description

### Changed
- Change description

### Fixed
- Bug fix description

### Removed
- Removed feature description

### Security
- Security fix description
```

### Changelog Sections

1. **Unreleased**: Changes in development (not yet released)
2. **Version tags**: Released versions with dates
3. **Categories**:
   - `Added`: New features
   - `Changed`: Changes in existing functionality
   - `Deprecated`: Soon-to-be removed features
   - `Removed`: Removed features
   - `Fixed`: Bug fixes
   - `Security`: Security vulnerabilities

### Changelog Workflow

1. **During development**: Add changes to `[Unreleased]` section
2. **Before release**: Move `[Unreleased]` to version section, add date
3. **After release**: Create new `[Unreleased]` section

**Example workflow:**

```markdown
## [Unreleased]
### Added
- New feature X
- New feature Y

## [v1.0.0] - 2025-01-15
### Added
- New feature X
- New feature Y
```

---

## Implementation Approaches

### Approach 1: VERSION File + Build-time Injection (Recommended)

**Pros:**
- ✅ Single source of truth
- ✅ Works with CI/CD
- ✅ Version visible in git
- ✅ Easy to automate

**Cons:**
- ⚠️ Requires build flags

**Implementation:**

1. Create `VERSION` file:
```bash
# VERSION
0.7.0-rc.1
```

2. Create version package:
```go
// pkg/version/version.go
package version

var (
    Version   = "dev"
    BuildTime = "unknown"
    GitCommit = "unknown"
    GitTag    = "unknown"
)
```

3. Build with ldflags:
```makefile
# Makefile
VERSION := $(shell cat VERSION 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_TAG := $(shell git describe --tags --exact-match 2>/dev/null || echo "unknown")

build:
	@echo "Building version $(VERSION)..."
	go build -ldflags "\
		-X github.com/boskuv/goreminder/pkg/version.Version=$(VERSION) \
		-X github.com/boskuv/goreminder/pkg/version.BuildTime=$(BUILD_TIME) \
		-X github.com/boskuv/goreminder/pkg/version.GitCommit=$(GIT_COMMIT) \
		-X github.com/boskuv/goreminder/pkg/version.GitTag=$(GIT_TAG)" \
		-o $(BINARY) $(MAIN)
```

4. Use in main.go:
```go
// cmd/core/main.go
import "github.com/boskuv/goreminder/pkg/version"

func main() {
    appVersion := version.Version
    if appVersion == "dev" {
        // Try to read from VERSION file as fallback
        if data, err := os.ReadFile("VERSION"); err == nil {
            appVersion = strings.TrimSpace(string(data))
        }
    }
    
    docs.SwaggerInfo.Version = appVersion
    routes.RegisterSystemRoutes(router, appVersion)
}
```

### Approach 2: Git Tags Only

**Pros:**
- ✅ No file to maintain
- ✅ Automatic from git

**Cons:**
- ⚠️ Requires git repository
- ⚠️ Harder to get version in development

**Implementation:**

```makefile
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
```

### Approach 3: Go Generate (Not Recommended)

**Pros:**
- ✅ Version in code

**Cons:**
- ❌ Requires manual generation
- ❌ Version in git history
- ❌ Easy to forget

### Approach 4: Environment Variable

**Pros:**
- ✅ Simple for Docker/Kubernetes

**Cons:**
- ❌ Not versioned
- ❌ Requires manual setting

**Use case**: Docker builds, Kubernetes deployments

```dockerfile
ARG VERSION=dev
ENV APP_VERSION=$VERSION
```

---

## Recommended Implementation

### Step 1: Create Version Package

```go
// pkg/version/version.go
package version

import (
    "fmt"
    "os"
    "strings"
)

var (
    // Version is the application version (set via ldflags)
    Version = "dev"
    
    // BuildTime is the build timestamp (set via ldflags)
    BuildTime = "unknown"
    
    // GitCommit is the git commit hash (set via ldflags)
    GitCommit = "unknown"
    
    // GitTag is the git tag (set via ldflags)
    GitTag = "unknown"
)

// GetVersion returns the application version
func GetVersion() string {
    if Version == "dev" {
        // Fallback: try to read from VERSION file
        if data, err := os.ReadFile("VERSION"); err == nil {
            return strings.TrimSpace(string(data))
        }
    }
    return Version
}

// GetFullVersion returns full version information
func GetFullVersion() map[string]string {
    return map[string]string{
        "version":   GetVersion(),
        "buildTime": BuildTime,
        "gitCommit": GitCommit,
        "gitTag":    GitTag,
    }
}

// String returns version as string
func String() string {
    v := GetVersion()
    if GitTag != "unknown" && GitTag != "" {
        return fmt.Sprintf("%s (tag: %s, commit: %s)", v, GitTag, GitCommit)
    }
    return fmt.Sprintf("%s (commit: %s)", v, GitCommit)
}
```

### Step 2: Create VERSION File

```bash
# VERSION
0.7.0-rc.1
```

### Step 3: Update Makefile

```makefile
# Version variables
VERSION := $(shell cat VERSION 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_TAG := $(shell git describe --tags --exact-match 2>/dev/null || echo "")

# Build with version
build:
	@echo "Building version $(VERSION)..."
	$(GO) build \
		-ldflags "\
			-X github.com/boskuv/goreminder/pkg/version.Version=$(VERSION) \
			-X github.com/boskuv/goreminder/pkg/version.BuildTime=$(BUILD_TIME) \
			-X github.com/boskuv/goreminder/pkg/version.GitCommit=$(GIT_COMMIT) \
			-X github.com/boskuv/goreminder/pkg/version.GitTag=$(GIT_TAG)" \
		-o $(BINARY) $(MAIN)
	@echo "Copying the configuration file..."
	cp $(CONFIG_FILEPATH) $(BUILD_DIR)/

# Show version
version:
	@echo "Version: $(VERSION)"
	@echo "Git commit: $(GIT_COMMIT)"
	@echo "Git tag: $(GIT_TAG)"
```

### Step 4: Update main.go

```go
// cmd/core/main.go
import (
    "github.com/boskuv/goreminder/pkg/version"
)

func main() {
    // ... existing code ...
    
    appVersion := version.GetVersion()
    
    // Setup swagger info
    docs.SwaggerInfo.Title = "Task Management API"
    docs.SwaggerInfo.Description = "API documentation for the Task Management system"
    docs.SwaggerInfo.Version = appVersion
    docs.SwaggerInfo.Host = "localhost:8080"
    docs.SwaggerInfo.Schemes = []string{"http"}
    
    // ... existing code ...
    
    // Register system routes with version
    routes.RegisterSystemRoutes(router, appVersion)
    
    // ... rest of code ...
}
```

### Step 5: Enhance Version Handler

```go
// internal/api/handlers/system_handler.go
func (h *VersionHandler) Version(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "version":   h.AppVersion,
        "buildTime": version.BuildTime,
        "gitCommit": version.GitCommit,
        "gitTag":    version.GitTag,
    })
}
```

---

## Release Process

### Pre-Release Checklist

1. ✅ All features implemented
2. ✅ Tests passing
3. ✅ Documentation updated
4. ✅ CHANGELOG updated
5. ✅ Version bumped in VERSION file

### Release Steps

#### 1. Update CHANGELOG

```markdown
## [Unreleased]

## [v1.0.0] - 2025-01-15
### Added
- Feature 1
- Feature 2

### Changed
- Improvement 1

### Fixed
- Bug fix 1
```

#### 2. Update VERSION File

```bash
# VERSION
1.0.0
```

#### 3. Commit Changes

```bash
git add CHANGELOG.md VERSION
git commit -m "chore: release v1.0.0"
```

#### 4. Create Git Tag

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

#### 5. Build Release

```bash
make build
# Binary will have version 1.0.0
```

#### 6. Create GitHub Release (if using GitHub)

- Title: `v1.0.0`
- Description: Copy from CHANGELOG
- Attach binaries

### Release Candidate Process

For RC releases:

```bash
# 1. Update VERSION
echo "1.0.0-rc.1" > VERSION

# 2. Update CHANGELOG
# Move [Unreleased] to [v1.0.0-rc.1] - 2025-01-15

# 3. Commit and tag
git add CHANGELOG.md VERSION
git commit -m "chore: release v1.0.0-rc.1"
git tag -a v1.0.0-rc.1 -m "Release candidate v1.0.0-rc.1"
git push origin v1.0.0-rc.1
```

### Stable Release from RC

```bash
# 1. Update VERSION (remove -rc.1)
echo "1.0.0" > VERSION

# 2. Update CHANGELOG
# Change [v1.0.0-rc.1] to [v1.0.0], update date

# 3. Commit and tag
git add CHANGELOG.md VERSION
git commit -m "chore: release v1.0.0"
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

---

## CI/CD Integration

### GitHub Actions Example

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Read version
        id: version
        run: |
          VERSION=$(cat VERSION)
          echo "VERSION=$VERSION" >> $GITHUB_OUTPUT
      
      - name: Build
        run: |
          make build
      
      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          tag_name: ${{ steps.version.outputs.VERSION }}
          name: Release ${{ steps.version.outputs.VERSION }}
          body_path: CHANGELOG.md
          files: bin/goreminder
```

### GitLab CI Example

```yaml
# .gitlab-ci.yml
release:
  stage: release
  only:
    - tags
  script:
    - VERSION=$(cat VERSION)
    - make build
    - echo "Building release $VERSION"
  artifacts:
    paths:
      - bin/goreminder
```

---

## Pre-commit Hooks (Optional)

### Validate VERSION File

```bash
# .git/hooks/pre-commit
#!/bin/bash

# Check if VERSION file matches CHANGELOG
VERSION=$(cat VERSION 2>/dev/null)
if [ -z "$VERSION" ]; then
    echo "Error: VERSION file not found"
    exit 1
fi

# Check if version in CHANGELOG matches VERSION file
if ! grep -q "\[$VERSION\]" CHANGELOG.md; then
    echo "Warning: Version $VERSION not found in CHANGELOG.md"
    echo "Consider updating CHANGELOG.md before committing"
fi
```

### Auto-update CHANGELOG (Advanced)

Use tools like:
- [git-chglog](https://github.com/git-chglog/git-chglog) - Generate changelog from git commits
- [standard-version](https://github.com/conventional-changelog/standard-version) - Automate versioning

---

## Best Practices

### ✅ Do

1. **Keep VERSION file in sync with CHANGELOG**
2. **Use semantic versioning consistently**
3. **Tag releases in git**
4. **Update CHANGELOG before release**
5. **Use release candidates for major changes**
6. **Include build metadata (commit, build time)**

### ❌ Don't

1. **Don't hardcode version in code**
2. **Don't skip CHANGELOG updates**
3. **Don't use same version twice**
4. **Don't commit without updating version for releases**
5. **Don't forget to tag releases**

---

## Summary

### Recommended Setup

1. **VERSION file**: Single source of truth
2. **Build-time injection**: Use ldflags
3. **Version package**: Centralized version info
4. **CHANGELOG**: Keep a Changelog format
5. **Git tags**: Tag all releases
6. **CI/CD**: Automate release builds

### Version Flow

```
Development → Update CHANGELOG [Unreleased]
           ↓
       Ready for release
           ↓
    Update VERSION file
           ↓
    Update CHANGELOG (move to version)
           ↓
    Commit + Tag
           ↓
    Build with version
           ↓
    Release
```

### For MVP

- Continue with `v0.7.0-rc.1`, `v0.8.0-rc.1`, etc.
- Use `v1.0.0` for MVP release
- Post-MVP: `v1.0.1`, `v1.1.0`, etc.

---

## Quick Start

1. Create `VERSION` file with `0.7.0-rc.1`
2. Create `pkg/version/version.go` (see code above)
3. Update `Makefile` with ldflags
4. Update `main.go` to use `version.GetVersion()`
5. Update `CHANGELOG.md` with new version
6. Build: `make build`
7. Check: `curl http://localhost:8080/version`

