#!/bin/bash

set -e

VERSION="$(python3 -c "import json,sys; print(json.load(open('wails.json'))['info']['productVersion'])")"
APP_NAME="NANITE"
DIST_DIR="dist"
OS="$(uname -s)"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo "${BLUE}========================================${NC}"
echo "${BLUE}  ${APP_NAME} v${VERSION} — Beta Distribution Build${NC}"
echo "${BLUE}  Platform: ${OS}${NC}"
echo "${BLUE}========================================${NC}"
echo ""

# Clean and create dist/
echo "${BLUE}Cleaning previous dist artifacts...${NC}"
rm -rf "${DIST_DIR}"
mkdir -p "${DIST_DIR}"
rm -rf build/bin/*

# ═════════════════════════════════════════
# macOS — Universal Binary + DMG
#         + Windows cross-compile + NSIS
# ═════════════════════════════════════════
if [ "$OS" = "Darwin" ]; then

    # ─────────────────────────────────────────
    # macOS — Universal Binary + DMG
    # ─────────────────────────────────────────
    echo ""
    echo "${BLUE}[1/2] Building macOS universal binary (Intel + Apple Silicon)...${NC}"

    wails build -clean -platform darwin/universal

    echo "${GREEN}macOS build succeeded.${NC}"

    MAC_APP="build/bin/${APP_NAME}.app"

    # Bust macOS icon cache so the new icon appears immediately after install
    touch "${MAC_APP}"
    /System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister \
        -f "${MAC_APP}" 2>/dev/null || true

    if command -v create-dmg &> /dev/null; then
        echo "${BLUE}Creating DMG with create-dmg...${NC}"

        DMG_OUT="${DIST_DIR}/${APP_NAME}-${VERSION}-macOS.dmg"

        # create-dmg exits 2 when the app is unsigned (codesign skipped) — that is expected
        set +e
        create-dmg \
            --volname "${APP_NAME} ${VERSION}" \
            --volicon "${MAC_APP}/Contents/Resources/iconfile.icns" \
            --window-pos 200 120 \
            --window-size 600 400 \
            --icon-size 100 \
            --icon "${APP_NAME}.app" 175 190 \
            --hide-extension "${APP_NAME}.app" \
            --app-drop-link 425 190 \
            "${DMG_OUT}" \
            "${MAC_APP}"
        CREATE_DMG_EXIT=$?
        set -e

        if [ $CREATE_DMG_EXIT -eq 0 ] || [ $CREATE_DMG_EXIT -eq 2 ]; then
            echo "${GREEN}Created ${DMG_OUT}${NC}"
            echo "  Size: $(du -h "${DMG_OUT}" | cut -f1)"
        else
            echo "${YELLOW}create-dmg exited with code ${CREATE_DMG_EXIT}; falling back to zip.${NC}"
            ZIP_OUT="${DIST_DIR}/${APP_NAME}-${VERSION}-macOS.zip"
            (cd build/bin && zip -r "../../${ZIP_OUT}" "${APP_NAME}.app")
            echo "${GREEN}Created ${ZIP_OUT} (fallback)${NC}"
            echo "  Size: $(du -h "${ZIP_OUT}" | cut -f1)"
        fi
    else
        echo "${YELLOW}create-dmg not found. Install with: brew install create-dmg${NC}"
        echo "${YELLOW}Falling back to zip...${NC}"
        ZIP_OUT="${DIST_DIR}/${APP_NAME}-${VERSION}-macOS.zip"
        (cd build/bin && zip -r "../../${ZIP_OUT}" "${APP_NAME}.app")
        echo "${GREEN}Created ${ZIP_OUT} (fallback)${NC}"
        echo "  Size: $(du -h "${ZIP_OUT}" | cut -f1)"
    fi

    # ─────────────────────────────────────────
    # Windows — Cross-compile + NSIS installer
    # ─────────────────────────────────────────
    echo ""
    echo "${BLUE}[2/2] Building Windows amd64 installer...${NC}"

    if command -v x86_64-w64-mingw32-gcc &> /dev/null; then
        echo "${BLUE}mingw-w64 found. Cross-compiling...${NC}"

        export CC=x86_64-w64-mingw32-gcc
        export CXX=x86_64-w64-mingw32-g++

        set +e
        wails build -platform windows/amd64 -nsis
        WIN_EXIT=$?
        set -e

        if [ $WIN_EXIT -eq 0 ]; then
            WIN_SETUP=$(find build/bin -name "*.exe" | grep -i "setup\|installer" | head -1)
            [ -z "$WIN_SETUP" ] && WIN_SETUP=$(find build/bin -name "*.exe" | head -1)

            if [ -n "$WIN_SETUP" ]; then
                DEST="${DIST_DIR}/${APP_NAME}-${VERSION}-Windows-Setup.exe"
                cp "${WIN_SETUP}" "${DEST}"
                echo "${GREEN}Created ${DEST}${NC}"
                echo "  Size: $(du -h "${DEST}" | cut -f1)"
            else
                echo "${YELLOW}Could not locate Windows installer in build/bin — listing contents:${NC}"
                ls -lh build/bin/
            fi
        else
            echo "${RED}Windows build failed (exit ${WIN_EXIT}).${NC}"
            echo "${YELLOW}Falling back to plain exe zip...${NC}"
            WIN_EXE=$(find build/bin -name "*.exe" | head -1)
            if [ -n "$WIN_EXE" ]; then
                ZIP_OUT="${DIST_DIR}/${APP_NAME}-${VERSION}-Windows.zip"
                zip "${ZIP_OUT}" "${WIN_EXE}"
                echo "${GREEN}Created ${ZIP_OUT} (fallback)${NC}"
            fi
        fi
    else
        echo "${YELLOW}mingw-w64 not found — skipping Windows build.${NC}"
        echo "${YELLOW}To enable Windows builds, run:${NC}"
        echo "  brew install mingw-w64"
        echo "  brew install makensis"
    fi

    echo ""
    echo "${YELLOW}Note: Linux builds must be run on a Linux machine.${NC}"
    echo "${YELLOW}  Clone the repo there and run: ./build-for-friends.sh${NC}"

# ═════════════════════════════════════════
# Linux — Native amd64 build
# ═════════════════════════════════════════
elif [ "$OS" = "Linux" ]; then

    echo ""
    echo "${BLUE}[1/1] Building Linux amd64...${NC}"

    wails build -clean -platform linux/amd64

    echo "${GREEN}Linux build succeeded.${NC}"

    LINUX_BIN="build/bin/nanite"

    if command -v appimagetool &> /dev/null; then
        echo "${BLUE}appimagetool found — creating AppImage...${NC}"

        APPDIR="$(pwd)/build/bin/NANITE.AppDir"
        rm -rf "${APPDIR}"
        mkdir -p "${APPDIR}/usr/bin"

        cp "${LINUX_BIN}" "${APPDIR}/usr/bin/nanite"
        chmod +x "${APPDIR}/usr/bin/nanite"

        cat > "${APPDIR}/AppRun" << 'APPRUN'
#!/bin/bash
exec "${APPDIR}/usr/bin/nanite" "$@"
APPRUN
        chmod +x "${APPDIR}/AppRun"

        cat > "${APPDIR}/nanite.desktop" << 'DESKTOP'
[Desktop Entry]
Name=NANITE
Exec=nanite
Icon=nanite
Type=Application
Categories=Productivity;
DESKTOP

        cp build/appicon.png "${APPDIR}/nanite.png"

        APPIMAGE_OUT="${DIST_DIR}/${APP_NAME}-${VERSION}-Linux.AppImage"

        set +e
        appimagetool "${APPDIR}" "${APPIMAGE_OUT}"
        APPIMAGE_EXIT=$?
        set -e

        rm -rf "${APPDIR}"

        if [ $APPIMAGE_EXIT -eq 0 ]; then
            chmod +x "${APPIMAGE_OUT}"
            echo "${GREEN}Created ${APPIMAGE_OUT}${NC}"
            echo "  Size: $(du -h "${APPIMAGE_OUT}" | cut -f1)"
        else
            echo "${YELLOW}appimagetool failed (exit ${APPIMAGE_EXIT}); falling back to tar.gz.${NC}"
            TAR_OUT="${DIST_DIR}/${APP_NAME}-${VERSION}-Linux.tar.gz"
            tar czf "${TAR_OUT}" -C build/bin nanite
            echo "${GREEN}Created ${TAR_OUT} (fallback)${NC}"
            echo "  Size: $(du -h "${TAR_OUT}" | cut -f1)"
        fi
    else
        echo "${YELLOW}appimagetool not found — falling back to tar.gz.${NC}"
        echo "${YELLOW}To get appimagetool: https://github.com/AppImage/AppImageKit/releases${NC}"
        TAR_OUT="${DIST_DIR}/${APP_NAME}-${VERSION}-Linux.tar.gz"
        tar czf "${TAR_OUT}" -C build/bin nanite
        echo "${GREEN}Created ${TAR_OUT}${NC}"
        echo "  Size: $(du -h "${TAR_OUT}" | cut -f1)"
    fi

else
    echo "${RED}Unsupported OS: ${OS}${NC}"
    exit 1
fi

# ─────────────────────────────────────────
# Summary
# ─────────────────────────────────────────
echo ""
echo "${GREEN}========================================${NC}"
echo "${GREEN}  Build complete! Artifacts in ${DIST_DIR}/  ${NC}"
echo "${GREEN}========================================${NC}"
echo ""
ls -lh "${DIST_DIR}/" 2>/dev/null || echo "(no artifacts produced)"
echo ""
echo "Share the files above along with INSTALL.md."
