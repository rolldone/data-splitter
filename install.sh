#!/bin/bash

# Data Splitter - Global Installation Script
# This script builds and installs data-splitter globally

set -e

echo "🔨 Building data-splitter..."

# Build with embedded project directory
go build -ldflags "-X main.projectDir=$(pwd)" -o data-splitter ./cmd

echo "📦 Installing to /usr/local/bin..."

# Install to global location
sudo cp data-splitter /usr/local/bin/
sudo chmod +x /usr/local/bin/data-splitter

echo "✅ Installation complete!"
echo ""
echo "🚀 You can now run 'data-splitter' from anywhere:"
echo "   data-splitter --info"
echo "   data-splitter --config /path/to/config.yaml"
echo ""
echo "📁 Make sure config.yaml and .env are in your working directory"