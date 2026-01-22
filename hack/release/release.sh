#!/bin/bash
# DP Release Script - Multi-module tagging
# This script creates version tags for all modules in the monorepo

set -e

VERSION=""
DRY_RUN=false
PUSH=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

usage() {
    echo "Usage: $0 -v <version> [options]"
    echo ""
    echo "Options:"
    echo "  -v, --version    Version to release (e.g., 0.1.0)"
    echo "  -p, --push       Push tags to remote"
    echo "  -d, --dry-run    Show what would be done"
    echo "  -h, --help       Show this help"
    echo ""
    echo "Examples:"
    echo "  $0 -v 0.1.0 --dry-run"
    echo "  $0 -v 0.1.0 --push"
    exit 1
}

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -p|--push)
            PUSH=true
            shift
            ;;
        -d|--dry-run)
            DRY_RUN=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            ;;
    esac
done

# Validate version
if [ -z "$VERSION" ]; then
    log_error "Version is required"
    usage
fi

# Validate version format (semver)
if ! [[ $VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?$ ]]; then
    log_error "Invalid version format. Expected: X.Y.Z or X.Y.Z-suffix"
    exit 1
fi

# Modules to tag (order matters for dependency resolution)
MODULES=(
    "contracts"
    "sdk"
    "cli"
    "platform/controller"
)

# Get repo root
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "$REPO_ROOT"

log_info "================================================"
log_info "  DP Release v${VERSION}"
log_info "================================================"
echo ""

# Check for uncommitted changes
if [ -n "$(git status --porcelain)" ]; then
    log_warn "You have uncommitted changes. Commit or stash them first."
    if [ "$DRY_RUN" = false ]; then
        exit 1
    fi
fi

# Check we're on main/master
BRANCH=$(git branch --show-current)
if [ "$BRANCH" != "main" ] && [ "$BRANCH" != "master" ]; then
    log_warn "Not on main/master branch (current: $BRANCH)"
    read -p "Continue anyway? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Build and test before tagging
log_info "Running tests..."
if [ "$DRY_RUN" = false ]; then
    make test || {
        log_error "Tests failed. Fix before releasing."
        exit 1
    }
fi

log_info "Running lint..."
if [ "$DRY_RUN" = false ]; then
    make lint || {
        log_error "Linting failed. Fix before releasing."
        exit 1
    }
fi

# Create tags for each module
log_info ""
log_info "Creating tags:"
for module in "${MODULES[@]}"; do
    TAG="${module}/v${VERSION}"
    
    if git rev-parse "$TAG" >/dev/null 2>&1; then
        log_warn "  Tag $TAG already exists, skipping"
        continue
    fi
    
    if [ "$DRY_RUN" = true ]; then
        log_info "  [DRY RUN] Would create tag: $TAG"
    else
        log_info "  Creating tag: $TAG"
        git tag -a "$TAG" -m "Release ${module} v${VERSION}"
    fi
done

# Create root version tag
ROOT_TAG="v${VERSION}"
if git rev-parse "$ROOT_TAG" >/dev/null 2>&1; then
    log_warn "  Tag $ROOT_TAG already exists, skipping"
else
    if [ "$DRY_RUN" = true ]; then
        log_info "  [DRY RUN] Would create tag: $ROOT_TAG"
    else
        log_info "  Creating tag: $ROOT_TAG"
        git tag -a "$ROOT_TAG" -m "Release v${VERSION}"
    fi
fi

# Push tags if requested
if [ "$PUSH" = true ]; then
    log_info ""
    log_info "Pushing tags to remote..."
    
    if [ "$DRY_RUN" = true ]; then
        log_info "  [DRY RUN] Would push tags"
    else
        git push origin --tags
        log_info "  Tags pushed successfully"
    fi
fi

log_info ""
log_info "================================================"
if [ "$DRY_RUN" = true ]; then
    log_info "  Dry run complete"
else
    log_info "  Release v${VERSION} complete!"
fi
log_info "================================================"
echo ""

# Show created tags
log_info "Tags created:"
for module in "${MODULES[@]}"; do
    echo "  • ${module}/v${VERSION}"
done
echo "  • v${VERSION}"
echo ""

if [ "$PUSH" = false ] && [ "$DRY_RUN" = false ]; then
    log_info "To push tags, run:"
    echo "  git push origin --tags"
fi
