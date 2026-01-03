#!/bin/bash
set -e

echo "Running post-clone script..."

echo "Trusting swift-openapi-generator plugin..."
defaults write com.apple.dt.Xcode IDESkipPackagePluginFingerprintValidatation -bool YES

if [ -z "$API_BASE_URL" ]; then
    echo "API_BASE_URL not set, using default localhost configuration"
    exit 0
fi

echo "Setting API_BASE_URL to: $API_BASE_URL"

CONFIG_FILE="${CI_PRIMARY_REPOSITORY_PATH}/koditon/Source/BuildConfig.swift"

cat > "$CONFIG_FILE" << EOF
import Foundation

enum BuildConfig {
    static let apiBaseURL = URL(string: "$API_BASE_URL")!
}
EOF

echo "Generated $CONFIG_FILE"
