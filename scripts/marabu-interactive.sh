#!/bin/bash

echo "Select UI mode:"
echo "1) TUI"
echo "2) CLI"
echo "3) Headless"
read -rp "Enter choice [1-3] (default: 1): " ui_choice
ui_choice=${ui_choice:-1}

echo "Select bootstrap mode:"
echo "1) Standard"
echo "2) No-bootstrap"
echo "3) Bootstrap Only"
read -rp "Enter choice [1-3] (default: 1): " boot_choice
boot_choice=${boot_choice:-1}

if [ "$ui_choice" = "1" ]; then
    ui="tui"
elif [ "$ui_choice" = "2" ]; then
    ui="cli"
else 
    ui="headless"
fi

if [ "$boot_choice" = "1" ]; then
    boot="standard"
elif [ "$boot_choice" = "2" ]; then
    boot="no-bootstrap"
else 
    boot="bootstrap-only"
fi

# construct the binary path
marabu="./bin/marabu-${boot}-${ui}"

# Make sure were running inside kitty
if [ -z "$KITTY_WINDOW_ID" ]; then
    echo "Error: This script should be run inside a kitty terminal. Fallback to CLI only."
    exec "$marabu"
    # exec replaces the bash process, so the script ends here on fallback
fi

FILE="${HOME}/dev/blockchain/marabu/logs/latest.log"
> "$FILE"

# Launch a new kitty window for logs and capture the window ID
if [ "$ui" = "headless" ]; then
     logsWindowID=$(kitty @ launch \
      --type=window \
      --title "Test Window" \
      bash -c "cd ~/dev/blockchain/marabu/; exec bash")
else
    logsWindowID=$(kitty @ launch \
          --type=window \
          --title "Marabu Logs" \
          --keep-focus \
          bash -c "sleep 0.2; exec tail -n +1 -F '$FILE' 2>/dev/null")
fi

# Ensure the extra window is closed when we quit the main node
trap "kitty @ close-window --match id:$logsWindowID 2>/dev/null" EXIT

# Run marabu in current terminal (blocks until user quits)
clear
"$marabu"
