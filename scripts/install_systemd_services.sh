#!/bin/bash
# Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
# SPDX-License-Identifier: MIT

# Installs/updates the inertial-computer systemd units from systemd/*.service and
# systemd/*.target, substituting @REPO_DIR@ and @USER@ with real values so the units
# work regardless of where the repo is checked out or which user built the binaries.
set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUN_USER="${SUDO_USER:-$(whoami)}"

echo "Installing systemd units for inertial-computer"
echo "  repo dir: $REPO_DIR"
echo "  run user: $RUN_USER"

for unit in "$REPO_DIR"/systemd/*.service "$REPO_DIR"/systemd/*.target; do
    name="$(basename "$unit")"
    sed -e "s|@REPO_DIR@|$REPO_DIR|g" -e "s|@USER@|$RUN_USER|g" "$unit" \
        | sudo tee "/etc/systemd/system/$name" > /dev/null
    echo "  installed $name"
done

sudo systemctl daemon-reload

cat <<EOF

Done. Enable the always-on services with:
  sudo systemctl enable --now inertial-imu inertial-gps inertial-hmc5983 \\
      inertial-web inertial-display inertial-celestial inertial-power

Optional (manual debugging aid, not part of inertial.target):
  sudo systemctl enable --now inertial-console-mqtt

Check status/logs with:
  systemctl status inertial-imu
  journalctl -u inertial-imu -f
EOF
