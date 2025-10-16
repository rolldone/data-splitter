#!/bin/bash

# Data Splitter - Uninstall Script
# This script removes the globally installed data-splitter

echo "🗑️  Uninstalling data-splitter..."

# Remove from global location
sudo rm -f /usr/local/bin/data-splitter

echo "✅ Uninstallation complete!"
echo ""
echo "Note: This only removes the binary. Your config files and logs remain untouched."