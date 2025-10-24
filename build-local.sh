#!/bin/bash

set -e

echo "🔧 Building PLANCK for Local Testing"
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Build for macOS
echo -e "${BLUE}🍎 Building PLANCK...${NC}"
wails build

if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✅ Build successful!${NC}"
    echo ""
    echo -e "${YELLOW}📍 Location:${NC} build/bin/PLANCK.app"
    echo -e "${YELLOW}📏 Size:${NC} $(du -sh build/bin/PLANCK.app | cut -f1)"
    echo ""
    echo -e "${GREEN}🚀 Launch commands:${NC}"
    echo "   Normal:  open build/bin/PLANCK.app"
    echo "   Debug:   DEBUG=1 open build/bin/PLANCK.app"
    echo "   Dev:     wails dev"
    echo ""
    
    # Ask if user wants to launch
    read -p "Launch PLANCK now? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        open build/bin/PLANCK.app
        echo -e "${GREEN}✅ PLANCK launched!${NC}"
    fi
else
    echo ""
    echo -e "${RED}❌ Build failed${NC}"
    exit 1
fi
