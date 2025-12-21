#!/bin/bash
# Manual test script for the TUI recorder
# Run this in an actual terminal (not through Claude Code)

set -e

echo "Building tend..."
make build

echo ""
echo "Building test TUI fixture..."
cd tests/e2e/fixtures/list-tui
go build -o /tmp/list-tui main.go
cd -

echo ""
echo "====================================================================="
echo "Starting TUI recording test"
echo "====================================================================="
echo ""
echo "Instructions:"
echo "1. The list-tui will appear"
echo "2. Press 'j' or down arrow a few times to move cursor"
echo "3. Press 'enter' to select an item"
echo "4. Press 'q' to quit"
echo ""
echo "The session will be saved to test-recording.html"
echo ""
read -p "Press ENTER to start recording..."

./bin/tend tui record --out test-recording.html -- /tmp/list-tui

echo ""
echo "====================================================================="
echo "Recording complete! Check test-recording.html"
echo "====================================================================="
echo ""
echo "To view the recording, open test-recording.html in a browser:"
echo "  open test-recording.html"
echo ""
