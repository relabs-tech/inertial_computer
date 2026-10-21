#!/usr/bin/env bash
# WifiNormal.sh — normal mode
#   Onboard Wi-Fi -> home network
#   USB dongle    -> nothing
#   Ethernet      -> untouched
# The AP and dongle profiles are kept (disabled) so NavComputer.sh can bring them back.
# Persists across reboots until NavComputer.sh is run.
#
# Usage: sudo ./WifiNormal.sh

set -euo pipefail

# --- configuration -----------------------------------------------------------
AP_CON="pi-ap"
DONGLE_CON="relabs-dongle"
HOME_SSID="Palacio De Relabs 2025"
ONBOARD_MAC="DC:A6:32:0C:E1:8D"
DONGLE_MAC="90:DE:80:58:14:A1"
LOG="/var/log/wifi-mode.log"
# ------------------------------------------------------------------------------

[[ $EUID -eq 0 ]] || exec sudo "$0" "$@"

con_exists()  { nmcli -g NAME con show | grep -Fxq "$1"; }
dev_for_mac() { ip -br link | awk -v m="${1,,}" 'tolower($3)==m {print $1}'; }

find_home_con() {         # the original home profile (not the dongle clone, not the AP)
  local name ssid mode
  while IFS= read -r name; do
    [[ $name == "$AP_CON" || $name == "$DONGLE_CON" ]] && continue
    [[ $(nmcli -g connection.type con show "$name") == 802-11-wireless ]] || continue
    ssid=$(nmcli -g 802-11-wireless.ssid con show "$name")
    mode=$(nmcli -g 802-11-wireless.mode con show "$name")
    if [[ $ssid == "$HOME_SSID" && $mode != ap ]]; then echo "$name"; return 0; fi
  done < <(nmcli -g NAME con show)
  return 1
}

if [[ -z "${WIFI_DETACHED:-}" ]]; then
  # ---- Preparation: profile edits only; nothing changes on the air yet ----
  HOME_CON=$(find_home_con) || { echo "ERROR: no saved connection for \"$HOME_SSID\"." >&2; exit 1; }
  nmcli con mod "$HOME_CON" \
    connection.interface-name "" 802-11-wireless.mac-address "$ONBOARD_MAC" \
    connection.autoconnect yes
  if con_exists "$AP_CON";     then nmcli con mod "$AP_CON"     connection.autoconnect no; fi
  if con_exists "$DONGLE_CON"; then nmcli con mod "$DONGLE_CON" connection.autoconnect no; fi

  # ---- Switching: run detached so a dropped SSH session can't interrupt it ----
  : > "$LOG"
  WIFI_DETACHED=1 setsid "$0" >>"$LOG" 2>&1 </dev/null &
  pid=$!
  echo "Switching to normal Wi-Fi mode (log: $LOG)."
  echo "SSH over the AP or the dongle will drop; eth0 is unaffected."
  tail -n +1 -f --pid="$pid" "$LOG"
  exit 0
fi

echo "== $(date) — normal Wi-Fi mode"

HOME_CON=$(find_home_con)

# Onboard radio back on the home network (this replaces the AP on that radio).
nmcli con up "$HOME_CON"

# Dongle: disconnect whatever it's on.
DEV=$(dev_for_mac "$DONGLE_MAC")
if [[ -n $DEV ]]; then
  nmcli dev disconnect "$DEV" 2>/dev/null || true
fi

echo
nmcli -f DEVICE,TYPE,STATE,CONNECTION dev status
echo
ip -4 -br addr
