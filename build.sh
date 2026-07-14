#!/bin/bash
# Build photocop binary and assemble .app bundle with ad-hoc signing.
set -e
cd "$(dirname "$0")"
go build -o photocop .

APP="dist/PhotoCopy.app"
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS"
mkdir -p "$APP/Contents/Resources"

# photocop binary in Resources
cp photocop "$APP/Contents/Resources/photocop"
chmod +x "$APP/Contents/Resources/photocop"

# bash wrapper as main executable
cp dist_template/PhotoCopy_launcher "$APP/Contents/MacOS/PhotoCopy"
chmod +x "$APP/Contents/MacOS/PhotoCopy"

# copy Info.plist
cp dist_template/Info.plist "$APP/Contents/Info.plist"

# Ad-hoc sign everything — binary first, then bundle
codesign --sign - --force "$APP/Contents/Resources/photocop"
codesign --sign - --force --deep "$APP"

echo "Built $APP"
