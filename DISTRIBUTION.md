# PLANCK Todo App - Distribution Guide

## Overview

This app uses **Wails v2**, which is App Store compliant and does **NOT** use private APIs. The build system creates native desktop applications for macOS and Windows.

## Private API Status

✅ **No private APIs detected**

The Info.plist only contains standard macOS bundle keys:
- CFBundleIdentifier
- CFBundleName
- CFBundleExecutable
- LSMinimumSystemVersion (10.13.0)
- NSHighResolutionCapable

The development version (Info.dev.plist) includes `NSAllowsLocalNetworking` for Vite dev server, but this is **NOT** included in production builds.

---

## Building for Distribution

### For Mac (Development/Friends)

#### 1. Build the App
```bash
cd /Users/chrispian/Downloads/todo-app-starter/todo-app
wails build -clean
```

The app will be created at:
```
build/bin/todo-app.app
```

#### 2. Create a DMG for Distribution (Optional)
```bash
# Install create-dmg if you don't have it
brew install create-dmg

# Create DMG
create-dmg \
  --volname "PLANCK" \
  --window-pos 200 120 \
  --window-size 800 400 \
  --icon-size 100 \
  --icon "todo-app.app" 200 190 \
  --hide-extension "todo-app.app" \
  --app-drop-link 600 185 \
  "PLANCK-Installer.dmg" \
  "build/bin/todo-app.app"
```

#### 3. Share with Friends
You can share the app in three ways:

**Option A: Share the .app directly**
- Zip the `build/bin/todo-app.app` folder
- Send to friends
- They unzip and drag to Applications folder

**Option B: Share the DMG**
- Send the `PLANCK-Installer.dmg`
- They open it and drag to Applications

**Option C: Notarize for macOS (recommended for wider distribution)**
See "Notarization" section below.

---

### For Windows

#### 1. Build for Windows (from Mac using cross-compilation)
```bash
# Install mingw-w64 for cross-compilation
brew install mingw-w64

# Build for Windows
wails build -platform windows/amd64
```

The app will be created at:
```
build/bin/todo-app.exe
```

#### 2. Create Windows Installer (Optional)
Wails supports NSIS for Windows installers, but requires Windows or Wine.

For friends, you can just zip the .exe and share it.

For proper installer:
```bash
# Install NSIS (on Windows or via Wine)
wails build -platform windows/amd64 -nsis
```

---

## Code Signing & Notarization (For Public Distribution)

### macOS Notarization

To distribute outside of friends/family without "Unknown Developer" warnings:

#### Prerequisites
1. Apple Developer Account ($99/year)
2. Developer ID Application Certificate
3. App-specific password for notarization

#### Steps

1. **Get your Developer ID**
```bash
security find-identity -v -p codesigning
```

2. **Sign the app**
```bash
codesign --deep --force --verify --verbose --sign "Developer ID Application: Your Name (TEAM_ID)" \
  build/bin/todo-app.app
```

3. **Create a signed DMG**
```bash
create-dmg \
  --volname "PLANCK" \
  --window-pos 200 120 \
  --window-size 800 400 \
  --icon-size 100 \
  --icon "todo-app.app" 200 190 \
  --app-drop-link 600 185 \
  "PLANCK-Installer.dmg" \
  "build/bin/todo-app.app"

codesign --sign "Developer ID Application: Your Name (TEAM_ID)" PLANCK-Installer.dmg
```

4. **Notarize with Apple**
```bash
# Create app-specific password at appleid.apple.com
# Store credentials
xcrun notarytool store-credentials "notarytool-profile" \
  --apple-id "your@email.com" \
  --team-id "TEAM_ID"

# Submit for notarization
xcrun notarytool submit PLANCK-Installer.dmg \
  --keychain-profile "notarytool-profile" \
  --wait

# Staple the notarization ticket
xcrun stapler staple PLANCK-Installer.dmg
```

5. **Verify**
```bash
spctl -a -t open --context context:primary-signature -v build/bin/todo-app.app
```

---

## Quick Distribution for Friends (No Signing)

### macOS

1. **Build:**
   ```bash
   wails build -clean
   ```

2. **Zip:**
   ```bash
   cd build/bin
   zip -r PLANCK-macOS.zip todo-app.app
   ```

3. **Share:**
   - Send `PLANCK-macOS.zip`
   - Friends need to: Right-click → Open (first time only to bypass Gatekeeper)

### Windows

1. **Build:**
   ```bash
   wails build -platform windows/amd64
   ```

2. **Zip:**
   ```bash
   cd build/bin
   zip PLANCK-Windows.zip todo-app.exe
   ```

3. **Share:**
   - Send `PLANCK-Windows.zip`
   - Friends unzip and run

---

## Build Configurations

### Production Build (Optimized)
```bash
wails build -clean
```

### Development Build (with DevTools)
```bash
wails build -devtools
```

### Universal macOS Build (Intel + Apple Silicon)
```bash
wails build -platform darwin/universal
```

### All Platforms
```bash
wails build -platform darwin/universal,windows/amd64
```

---

## Customizing App Info

Edit `wails.json` to customize:
```json
{
  "name": "todo-app",
  "outputfilename": "PLANCK",
  "author": {
    "name": "Your Name",
    "email": "your@email.com"
  },
  "info": {
    "companyName": "Your Company",
    "productName": "PLANCK",
    "productVersion": "1.0.0",
    "copyright": "Copyright © 2025 Your Name",
    "comments": "A minimalist todo app"
  }
}
```

Then rebuild:
```bash
wails build -clean
```

---

## App Store Distribution (Future)

The app is **App Store compliant** and ready for submission once you:

1. Add proper bundle identifier in wails.json
2. Set up provisioning profiles
3. Add required Info.plist keys (privacy descriptions if needed)
4. Follow Apple's App Store Review Guidelines
5. Submit via App Store Connect

---

## Troubleshooting

### macOS "App is damaged" error
This happens with unsigned apps downloaded from the internet.

**Solution for users:**
```bash
xattr -cr /Applications/todo-app.app
```

Or right-click → Open (instead of double-click) the first time.

### Windows SmartScreen warning
Unsigned Windows apps show SmartScreen warnings.

**Solution:**
- For friends: "More info" → "Run anyway"
- For public: Get a code signing certificate from a CA like DigiCert

### Missing dependencies
Make sure you have all dependencies:
```bash
wails doctor
```

---

## File Size

Typical build sizes:
- **macOS**: ~20-30 MB (arm64), ~50-60 MB (universal)
- **Windows**: ~25-35 MB (amd64)

To reduce size:
```bash
# Install UPX
brew install upx

# Build with compression
wails build -upx
```

---

## Next Steps

1. ✅ Build works without private APIs
2. For friends: Use unsigned builds with simple zip distribution
3. For wider distribution: Set up code signing & notarization
4. For commercial release: Consider App Store submission

---

## Support & Issues

- Wails Documentation: https://wails.io/docs/introduction
- Code signing guide: https://wails.io/docs/guides/mac-appstore
- Windows installer: https://wails.io/docs/guides/windows-installer
