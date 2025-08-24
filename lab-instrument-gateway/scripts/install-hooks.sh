#!/bin/bash

# Script to install git hooks for security checks

set -e

echo "🔧 Installing git hooks for security checks..."

# Create .git/hooks directory if it doesn't exist
mkdir -p .git/hooks

# Copy pre-commit hook
if [ -f ".githooks/pre-commit" ]; then
    cp .githooks/pre-commit .git/hooks/pre-commit
    chmod +x .git/hooks/pre-commit
    echo "✅ Pre-commit hook installed"
else
    echo "❌ Pre-commit hook file not found"
    exit 1
fi

# Test the hook
echo "🧪 Testing pre-commit hook..."
if .git/hooks/pre-commit; then
    echo "✅ Pre-commit hook test passed"
else
    echo "⚠️  Pre-commit hook test failed (this is normal if there are issues to fix)"
fi

echo
echo "🎉 Git hooks installation complete!"
echo
echo "The pre-commit hook will now:"
echo "  • Check for sensitive data patterns"
echo "  • Warn about large files"
echo "  • Check Go code formatting"
echo "  • Look for TODO/FIXME comments"
echo
echo "To bypass the hook (not recommended):"
echo "  git commit --no-verify"
echo