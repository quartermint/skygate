#!/bin/bash
# Non-interactive Pi-hole installation
# Uses setupVars.conf for pre-seeded configuration
set -euo pipefail

if command -v pihole &>/dev/null; then
    echo "Pi-hole already installed, skipping installation"
    exit 0
fi

# Install Pi-hole non-interactively
curl -sSL https://install.pi-hole.net | bash /dev/stdin --unattended
