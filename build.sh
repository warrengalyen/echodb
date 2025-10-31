#!/bin/bash

# Build script for echodb

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "Building echodb..."

mkdir -p bin

go build -o bin/echodb ./cmd

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Build successful!${NC}"
    echo "Executable created at: bin/echodb"
else
    echo -e "${RED}✗ Build failed!${NC}"
    exit 1
fi

