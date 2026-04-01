#!/bin/bash
# TUI Launcher - 在真实终端中运行 TUI

echo "🚀 Starting gClawCoder TUI..."
echo ""
echo "If the TUI doesn't start properly, try:"
echo "  1. Make sure you're in a real terminal (not IDE console)"
echo "  2. If using SSH, use: ssh -t user@host ./gclaw tui"
echo "  3. Try REPL mode instead: ./gclaw repl"
echo ""
echo "Press Ctrl+C to exit TUI mode"
echo ""

./gclaw tui
