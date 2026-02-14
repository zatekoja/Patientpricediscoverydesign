#!/bin/bash
# Setup script for Pulumi infrastructure

set -e

echo "🚀 Setting up Open Health Initiative Infrastructure..."

# Navigate to pulumi directory
cd "$(dirname "$0")/../pulumi"

echo "📦 Installing Node.js dependencies..."
npm install

echo "✅ Pulumi TypeScript compilation check..."
npm run typecheck

echo "🧪 Running tests..."
npm test

echo ""
echo "✅ Setup complete!"
echo ""
echo "Next steps:"
echo "1. Configure AWS credentials: aws configure"
echo "2. Login to Pulumi: pulumi login"
echo "3. Select stack: pulumi stack select dev"
echo "4. Preview changes: pulumi preview"
echo "5. Deploy: pulumi up"
