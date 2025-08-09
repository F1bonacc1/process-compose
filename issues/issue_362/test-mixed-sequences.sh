#!/bin/bash
echo "=== Testing Mixed ANSI Sequences ==="

# Initial content
echo "Build process starting..."
echo "Preparing files..."
sleep 1

# Clear and show progress
printf "\033[2J\033[H"
echo "Building project..."
echo ""

# Progress bar with \r updates
for i in {1..10}; do
    printf "\rProgress: [%-10s] %d0%%" $(head -c $i < /dev/zero | tr '\0' '█') $i
    sleep 0.3
done
printf "\n"

sleep 1

# Clear again and show completion
printf "\033[2J\033[H"
echo "✅ Build completed successfully!"
echo "📦 Output: dist/app.js"
echo "🚀 Ready for deployment"
