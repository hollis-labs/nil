#!/bin/bash

set -e

echo "🚀 Building PLANCK Todo App for Distribution"
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Clean previous builds
echo -e "${BLUE}📦 Cleaning previous builds...${NC}"
rm -rf build/bin/*
rm -f PLANCK-*.zip PLANCK-*.dmg

# Build for macOS
echo -e "${BLUE}🍎 Building for macOS...${NC}"
wails build -clean

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ macOS build successful!${NC}"
    
    # Create zip for easy sharing
    cd build/bin
    zip -r ../../PLANCK-macOS.zip PLANCK.app
    cd ../..
    
    echo -e "${GREEN}✅ Created PLANCK-macOS.zip${NC}"
    echo ""
    echo "📍 Location: $(pwd)/PLANCK-macOS.zip"
    echo "📏 Size: $(du -h PLANCK-macOS.zip | cut -f1)"
else
    echo "❌ macOS build failed"
    exit 1
fi

# Optional: Build for Windows (requires mingw-w64)
read -p "Build for Windows? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    if command -v x86_64-w64-mingw32-gcc &> /dev/null; then
        echo -e "${BLUE}🪟 Building for Windows...${NC}"
        wails build -platform windows/amd64
        
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}✅ Windows build successful!${NC}"
            
            cd build/bin
            zip ../../PLANCK-Windows.zip planck.exe
            cd ../..
            
            echo -e "${GREEN}✅ Created PLANCK-Windows.zip${NC}"
            echo ""
            echo "📍 Location: $(pwd)/PLANCK-Windows.zip"
            echo "📏 Size: $(du -h PLANCK-Windows.zip | cut -f1)"
        else
            echo "❌ Windows build failed"
        fi
    else
        echo "⚠️  mingw-w64 not found. Install with: brew install mingw-w64"
    fi
fi

echo ""
echo -e "${GREEN}🎉 Build complete!${NC}"
echo ""
echo "📤 Share these files with your friends:"
ls -lh PLANCK-*.zip 2>/dev/null || echo "No zip files created"
echo ""
echo "📖 For detailed distribution instructions, see DISTRIBUTION.md"
