#!/bin/bash

# Data Splitter - Cross-platform Build Script
# Builds binaries for Linux, Windows, and macOS

set -e

PROJECT_DIR=$(pwd)
OUTPUT_DIR="dist"

echo "🔨 Building data-splitter for multiple platforms..."

# Create output directory
mkdir -p $OUTPUT_DIR

# Build for Linux (amd64)
echo "📦 Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.projectDir=$PROJECT_DIR" -o $OUTPUT_DIR/data-splitter-linux-amd64 ./cmd

# Build for Windows (amd64)
echo "📦 Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -ldflags "-X main.projectDir=$PROJECT_DIR" -o $OUTPUT_DIR/data-splitter-windows-amd64.exe ./cmd

# Build for macOS (amd64)
echo "📦 Building for macOS (amd64)..."
GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.projectDir=$PROJECT_DIR" -o $OUTPUT_DIR/data-splitter-darwin-amd64 ./cmd

# Build for macOS (arm64 - Apple Silicon)
echo "📦 Building for macOS (arm64)..."
GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.projectDir=$PROJECT_DIR" -o $OUTPUT_DIR/data-splitter-darwin-arm64 ./cmd

echo "✅ Build complete!"
echo ""
echo "📁 Binaries created in $OUTPUT_DIR/:"
ls -la $OUTPUT_DIR/
echo ""
echo "🚀 To install on your platform:"
echo "  Linux:   sudo cp $OUTPUT_DIR/data-splitter-linux-amd64 /usr/local/bin/data-splitter"
echo "  Windows: Copy $OUTPUT_DIR/data-splitter-windows-amd64.exe to a folder in PATH"
echo "  macOS:   sudo cp $OUTPUT_DIR/data-splitter-darwin-amd64 /usr/local/bin/data-splitter"