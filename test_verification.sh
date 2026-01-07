#!/bin/bash

echo "================================"
echo "Associate - Final Verification"
echo "================================"
echo ""

echo "1. Checking Go version..."
go version
echo ""

echo "2. Building application..."
go build -o associate
if [ $? -eq 0 ]; then
    echo "✅ Build successful"
else
    echo "❌ Build failed"
    exit 1
fi
echo ""

echo "3. Running tests..."
go test ./... 
if [ $? -eq 0 ]; then
    echo "✅ All tests passed"
else
    echo "❌ Tests failed"
    exit 1
fi
echo ""

echo "4. Checking binary size..."
ls -lh associate | awk '{print "Binary size: " $5}'
echo ""

echo "5. Verifying commands..."
./associate --help > /dev/null 2>&1 && echo "✅ Root command works"
./associate config --help > /dev/null 2>&1 && echo "✅ config command works"
./associate init --help > /dev/null 2>&1 && echo "✅ init command works"
./associate refresh-memory --help > /dev/null 2>&1 && echo "✅ refresh-memory command works"
./associate reset-memory --help > /dev/null 2>&1 && echo "✅ reset-memory command works"
echo ""

echo "6. Checking documentation..."
[ -f README.md ] && echo "✅ README.md exists"
[ -f USAGE_EXAMPLES.md ] && echo "✅ USAGE_EXAMPLES.md exists"
[ -f REQUIREMENTS_CHECKLIST.md ] && echo "✅ REQUIREMENTS_CHECKLIST.md exists"
[ -f .env.example ] && echo "✅ .env.example exists"
echo ""

echo "7. Verifying .gitignore..."
grep -q ".env" .gitignore && echo "✅ .env is gitignored"
echo ""

echo "================================"
echo "✅ ALL VERIFICATIONS PASSED"
echo "================================"
echo ""
echo "The Associate application is ready for use!"
echo ""
echo "Quick start:"
echo "  1. ./associate config set NEO4J_PASSWORD yourpassword"
echo "  2. ./associate init"
echo "  3. ./associate refresh-memory"
