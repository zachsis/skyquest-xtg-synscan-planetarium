#!/bin/zsh
# retry-claude.sh
while true; do
  claude --resume 24656658-0e21-43ee-ac1e-4cfc63a910ba && break
  echo "Limit hit, retrying in 5 minutes..."
  sleep 300
done
