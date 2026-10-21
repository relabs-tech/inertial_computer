#!/usr/bin/env bash
# NavComputer.sh — Navigation Computer mode
#   Onboard Wi-Fi -> access point "Navigation Computer" (192.168.50.1, DHCP + NAT for clients)
#   USB dongle    -> home network (uplink for AP clients, when plugged in)
#   Ethernet      -> untouched
# Persists across reboots until WifiNormal.sh is run.
#
# Usage: sudo ./NavComputer.sh
#        First run asks for the AP passphrase (or: sudo AP_PSK='...' ./NavComputer.sh)

set -euo pipefail

# --- configuration -----------------------------------------------------------
AP_CON="pi-ap"
AP_SSID="Navigation Computer"
AP_ADDR="192.168.50.1/24"
AP_BAND="a"               # "a" = 5 GHz, "bg" = 2.4 GHz
AP_CHANNEL="36"           # 36/40/44/48 avoid DFS under ETSI
DONGLE_CON="relabs-dongle"
DONGLE_METRIC="700"       # dongle is last choice for outbound traffic (eth0 ~100)
HOME_SSID="Palacio De Relabs 2025"
ONBOARD_MAC="DC:A6:32:0C:E1:8D"
DONGLE_MAC="90:DE:80:58:14:A1"
COUNTRY="ES"
LOG="/var/log/wifi-mode.log"
# ------------------------------------------------------------------------------

[[ $EUID -eq 0 ]] || exec sudo --preserve-env=AP_PSK "$0" "$@"

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

  # Dongle profile: a copy of the home profile, bound to the dongle.
  if ! con_exists "$DONGLE_CON"; then
    echo "Creating \"$DONGLE_CON\" from \"$HOME_CON\""
    nmcli con clone "$HOME_CON" "$DONGLE_CON" >/dev/null
  fi
  nmcli con mod "$DONGLE_CON" \
    connection.interface-name "" 802-11-wireless.mac-address "$DONGLE_MAC" \
    ipv4.route-metric "$DONGLE_METRIC" ipv6.route-metric "$DONGLE_METRIC" \
    connection.autoconnect yes
  if [[ -z $(nmcli -s -g 802-11-wireless-security.psk con show "$DONGLE_CON") ]]; then
    read -rsp "Passphrase for \"$HOME_SSID\": " psk; echo
    nmcli con mod "$DONGLE_CON" wifi-sec.psk "$psk"
  fi

  # AP profile, bound to the onboard radio.
  props=(
    802-11-wireless.mode ap
    802-11-wireless.band "$AP_BAND"
    802-11-wireless.channel "$AP_CHANNEL"
    802-11-wireless.mac-address "$ONBOARD_MAC"
    connection.interface-name ""
    wifi-sec.key-mgmt wpa-psk
    wifi-sec.proto rsn
    wifi-sec.pairwise ccmp
    wifi-sec.group ccmp
    wifi-sec.pmf disable          # brcmfmac AP can fail to start with PMF enabled
    ipv4.method shared            # DHCP for clients + NAT out via any uplink
    ipv4.addresses "$AP_ADDR"
    ipv6.method disabled
    connection.autoconnect yes
    connection.autoconnect-priority 10
  )
  if ! con_exists "$AP_CON" && [[ -z "${AP_PSK:-}" ]]; then
    read -rsp "Passphrase for \"$AP_SSID\" (8-63 chars): " AP_PSK; echo
  fi
  if [[ -n "${AP_PSK:-}" ]]; then
    (( ${#AP_PSK} >= 8 && ${#AP_PSK} <= 63 )) || { echo "Passphrase must be 8-63 characters." >&2; exit 1; }
    props+=(wifi-sec.psk "$AP_PSK")
  fi
  if con_exists "$AP_CON"; then
    nmcli con mod "$AP_CON" 802-11-wireless.ssid "$AP_SSID" "${props[@]}"
  else
    nmcli con add type wifi con-name "$AP_CON" ssid "$AP_SSID" "${props[@]}" >/dev/null
  fi

  # Home profile stays bound to the onboard radio but doesn't autoconnect in this mode.
  nmcli con mod "$HOME_CON" \
    connection.interface-name "" 802-11-wireless.mac-address "$ONBOARD_MAC" \
    connection.autoconnect no

  # ---- Switching: run detached so a dropped SSH session can't interrupt it ----
  : > "$LOG"
  WIFI_DETACHED=1 setsid "$0" >>"$LOG" 2>&1 </dev/null &
  pid=$!
  echo "Switching to Navigation Computer mode (log: $LOG)."
  echo "SSH over onboard Wi-Fi will drop; eth0 is unaffected."
  tail -n +1 -f --pid="$pid" "$LOG"
  exit 0
fi

echo "== $(date) — Navigation Computer mode"

# 5 GHz AP mode needs the regulatory domain set.
if ! iw reg get | grep -q "country $COUNTRY"; then
  echo "Setting Wi-Fi country to $COUNTRY"
  raspi-config nonint do_wifi_country "$COUNTRY"
  sleep 2
fi

# Dongle first, so the uplink is in place before the onboard radio changes role.
if [[ -n $(dev_for_mac "$DONGLE_MAC") ]]; then
  nmcli con up "$DONGLE_CON" || echo "WARNING: dongle failed to join \"$HOME_SSID\"; AP will run without an uplink."
else
  echo "Dongle not plugged in; AP will run without an uplink."
fi

nmcli con up "$AP_CON"

echo
nmcli -f DEVICE,TYPE,STATE,CONNECTION dev status
echo
ip -4 -br addr
