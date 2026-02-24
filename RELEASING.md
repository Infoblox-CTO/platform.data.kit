# Releasing DP CLI

This document describes the process for releasing new versions of the DP CLI.

## Prerequisites

Before creating a release, ensure:

1. **GitHub Secrets are configured** (one-time setup):
   - `HOMEBREW_TAP_TOKEN` - Personal Access Token with `repo` scope for [Infoblox-CTO/homebrew-tap](https://github.com/Infoblox-CTO/homebrew-tap)
   - `SCOOP_BUCKET_TOKEN` - Personal Access Token with `repo` scope for [Infoblox-CTO/scoop-bucket](https://github.com/Infoblox-CTO/scoop-bucket)

2. **External repositories exist** (one-time setup):
   - [Infoblox-CTO/homebrew-tap](https://github.com/Infoblox-CTO/homebrew-tap) - Homebrew tap with `Formula/` directory
   - [Infoblox-CTO/scoop-bucket](https://github.com/Infoblox-CTO/scoop-bucket) - Scoop bucket repository

3. **All tests pass**:
   ```bash
   make test
   ```

4. **GoReleaser configuration is valid**:
   ```bash
   make release-local
   ```

## Creating a Release

### 1. Prepare the Release

Ensure your main branch is up to date:

```bash
git checkout main
git pull origin main
```

### 2. Create a Version Tag

Use [Semantic Versioning](https://semver.org/):

- **MAJOR**: Breaking changes
- **MINOR**: New features, backward compatible
- **PATCH**: Bug fixes, backward compatible

```bash
# Create an annotated tag
git tag -a v1.2.3 -m "Release v1.2.3"

# Push the tag to trigger the release workflow
git push origin v1.2.3
```

### 3. Monitor the Release

1. Go to [GitHub Actions](https://github.com/Infoblox-CTO/platform.data.kit/actions)
2. Watch the "Release" workflow
3. Release builds typically complete in 5-10 minutes

### 4. Verify the Release

After the workflow completes:

1. Check [GitHub Releases](https://github.com/Infoblox-CTO/platform.data.kit/releases) for the new version
2. Verify all artifacts are present:
   - `dp_X.Y.Z_darwin_amd64.tar.gz`
   - `dp_X.Y.Z_darwin_arm64.tar.gz`
   - `dp_X.Y.Z_linux_amd64.tar.gz`
   - `dp_X.Y.Z_windows_amd64.zip`
   - `checksums.txt`

3. Verify the Homebrew formula was updated in [homebrew-tap](https://github.com/Infoblox-CTO/homebrew-tap/blob/main/Formula/dp.rb)

4. Verify the Scoop manifest was updated in [scoop-bucket](https://github.com/Infoblox-CTO/scoop-bucket/blob/main/dp.json)

## Testing Installation Methods

### macOS (Homebrew)

```bash
# Update tap and upgrade
brew update
brew upgrade dp

# Or install fresh
brew install Infoblox-CTO/tap/dp

# Verify
dp version
```

### Linux (Install Script)

```bash
# Set token for private repo access
export GITHUB_TOKEN=your_token

# Run install script
curl -sSfL https://raw.githubusercontent.com/Infoblox-CTO/platform.data.kit/main/scripts/install.sh | sh

# Verify
dp version
```

### Windows (Scoop)

```powershell
# Update and install
scoop update dp

# Or install fresh
scoop bucket add infoblox https://github.com/Infoblox-CTO/scoop-bucket
scoop install dp

# Verify
dp version
```

## Troubleshooting

### Release Workflow Fails

1. **Missing secrets**: Ensure `HOMEBREW_TAP_TOKEN` and `SCOOP_BUCKET_TOKEN` are configured in repository settings
2. **Invalid token permissions**: PATs need `repo` scope for the respective repositories
3. **GoReleaser config error**: Run `make release-local` to validate locally

### Homebrew Formula Not Updated

1. Check the `HOMEBREW_TAP_TOKEN` secret has write access to homebrew-tap
2. Verify the token hasn't expired
3. Check the homebrew-tap repository exists and has a `Formula/` directory

### Scoop Manifest Not Updated

1. Check the `SCOOP_BUCKET_TOKEN` secret has write access to scoop-bucket
2. Verify the token hasn't expired
3. Check the scoop-bucket repository exists

## Pre-release Versions

For pre-release versions (alpha, beta, rc), use appropriate version suffixes:

```bash
git tag -a v1.0.0-alpha.1 -m "Release v1.0.0-alpha.1"
git tag -a v1.0.0-beta.1 -m "Release v1.0.0-beta.1"
git tag -a v1.0.0-rc.1 -m "Release v1.0.0-rc.1"
```

GoReleaser automatically marks these as pre-releases on GitHub.

## Local Release Testing

To test the release process locally without publishing:

```bash
# Validate GoReleaser configuration
goreleaser check

# Build all binaries locally (snapshot mode)
make release-local

# Binaries are created in dist/
ls -la dist/
```
