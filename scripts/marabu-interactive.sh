#!/bin/bash

marabu="./bin/marabu-no-bootstrap-headless"

echo "Select UI mode:"
echo "1) Headless"
echo "2) CLI"
read -rp "Enter choice [1-2] (default: 1): " ui_choice
ui_choice=${ui_choice:-1}

echo "Select bootstrap mode:"
echo "1) No-bootstrap"
echo "2) Standard"
echo "3) Bootstrap Only"
read -rp "Enter choice [1-2] (default: 1): " boot_choice
boot_choice=${boot_choice:-1}

if [ "$ui_choice" = "1" ]; then
    ui="headless"
else
    ui="cli"
fi

if [ "$boot_choice" = "1" ]; then
    boot="no-bootstrap"
elif [ "$boot_choice" = "2" ]; then
    boot="standard"
else 
    boot="bootstrap-only"
fi

# Determine executable
if [ "$ui" = "cli" ]; then
  if [ "$boot" = "standard" ]; then
    marabu="./bin/marabu-standard-cli"
  elif [ "$boot" = "bootstrap-only" ]; then
    marabu="./bin/marabu-bootstrap-only-cli"
  else
    marabu="./bin/marabu-no-bootstrap-cli"
  fi
else
  if [ "$boot" = "standard" ]; then
    marabu="./bin/marabu-standard-headless"
  elif [ "$boot" = "bootstrap-only" ]; then
    marabu="./bin/marabu-bootstrap-only-cli"
  else
    marabu="./bin/marabu-no-bootstrap-headless"
  fi
fi

# Make sure we're running inside kitty
if [ -z "$KITTY_WINDOW_ID" ]; then
  echo "Error: This script should be run inside a kitty terminal. Fallback to CLI only."
  exec $marabu
  exit 1
fi

FILE="${HOME}/dev/blockchain/marabu/logs/latest.log"

> $FILE

# Launch a new kitty window for logs and capture the window ID
if [ "$ui" = "cli" ]; then
    logsWindowID=$(kitty @ launch \
          --type=window \
          --title "Marabu Logs" \
          --keep-focus \
          bash -c "sleep 0.2; exec tail -n +1 -F '$FILE' 2>/dev/null")
    else
    logsWindowID=$(kitty @ launch \
      --type=window \
      --title "Test Window" \
      bash -c "cd ~/dev/blockchain/marabu/; exec bash")
fi

trap "kitty @ close-window --match id:$logsWindowID" EXIT

# Run marabu in current terminal
clear
$marabu
PID=$!
wait $PID
