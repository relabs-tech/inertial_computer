#!/bin/bash
# Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
# SPDX-License-Identifier: MIT
# See LICENSE file for full license text
#
# Manual start/stop/status helper for all inertial-computer producers and
# servers, for development/bench use without systemd. Builds binaries into
# bin/ (the repo root itself cannot be used: it already contains web/ and
# celestial/ asset directories that the web/celestial servers serve from),
# launches them with the repo root as working directory so those relative
# asset paths still resolve, and tracks PIDs/logs under .run/ and logs/.
#
# Usage:
#   scripts/startup.sh [start|stop|restart|status|build] [--with-console]
#
# start   - build (if needed) and launch all components (default command)
# stop    - stop all running components
# restart - stop then start
# status  - show running/stopped state for each component
# build   - just (re)build all binaries, don't launch anything
#
# --with-console  also start console_mqtt (debugging aid, prints MQTT feed;
#                 normally left out since it duplicates the web dashboard)
set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_DIR="$REPO_DIR/bin"
PID_DIR="$REPO_DIR/.run"
LOG_DIR="$REPO_DIR/logs"
RUN_USER="${SUDO_USER:-$(whoami)}"

# name -> needs root for hardware access (SPI/I2C/GPIO/serial)
declare -A NEEDS_SUDO=(
    [imu_producer]=1
    [gps_producer]=1
    [hmc5983_producer]=1
    [power_producer]=1
    [display]=1
    [web]=0
    [celestial]=0
    [console_mqtt]=0
)

COMPONENTS=(imu_producer gps_producer hmc5983_producer power_producer display web celestial)

WITH_CONSOLE=0
COMMAND="start"
for arg in "$@"; do
    case "$arg" in
        start|stop|restart|status|build) COMMAND="$arg" ;;
        --with-console) WITH_CONSOLE=1 ;;
        *) echo "Unknown argument: $arg" >&2; exit 1 ;;
    esac
done
if [[ "$WITH_CONSOLE" -eq 1 ]]; then
    COMPONENTS+=(console_mqtt)
fi

mkdir -p "$BIN_DIR" "$PID_DIR" "$LOG_DIR"

build_component() {
    local name="$1"
    echo "Building $name..."
    (cd "$REPO_DIR" && go build -o "$BIN_DIR/$name" "./cmd/$name/")
}

build_all() {
    for name in "${COMPONENTS[@]}"; do
        build_component "$name"
    done
}

is_running() {
    local name="$1"
    local pid_file="$PID_DIR/$name.pid"
    [[ -f "$pid_file" ]] || return 1
    local pid
    pid="$(cat "$pid_file" 2>/dev/null)"
    # /proc/<pid> is readable regardless of owner, unlike kill -0 (which
    # returns EPERM for root-owned processes and falsely looks "stopped")
    [[ -n "$pid" && -d "/proc/$pid" ]] || return 1
    grep -q "$name" "/proc/$pid/cmdline" 2>/dev/null
}

check_mosquitto() {
    if ! (systemctl is-active --quiet mosquitto 2>/dev/null || pgrep -x mosquitto >/dev/null 2>&1); then
        echo "Warning: mosquitto MQTT broker does not appear to be running." >&2
        echo "  Start it with: sudo systemctl start mosquitto" >&2
    fi
}

start_component() {
    local name="$1"
    if is_running "$name"; then
        echo "$name already running (pid $(cat "$PID_DIR/$name.pid"))"
        return
    fi
    if [[ ! -x "$BIN_DIR/$name" ]]; then
        build_component "$name"
    fi

    local pid_file="$PID_DIR/$name.pid"
    rm -f "$pid_file"
    local bin="$BIN_DIR/$name"
    local log="$LOG_DIR/$name.log"
    if [[ "${NEEDS_SUDO[$name]}" -eq 1 && "$(id -u)" -ne 0 ]]; then
        echo "Starting $name (sudo)..."
        # Do the backgrounding + pid capture as root in one non-backgrounded
        # 'sudo bash -c' call so $! is the real binary's pid, not a sudo
        # wrapper pid (plain 'sudo ... &' plus '-b' spawns extra sudo/pty
        # layers and the outer $! ends up pointing at the wrong process).
        # 'cd' must be its own statement (not "cd X && nohup Y &"): with &&,
        # '&' backgrounds the whole chain as one subshell and $! would be
        # that subshell's pid instead of nohup's.
        sudo bash -c 'cd "$1" || exit 1; nohup "$2" >>"$3" 2>&1 </dev/null & echo $! > "$4"' \
            _ "$REPO_DIR" "$bin" "$log" "$pid_file"
    else
        echo "Starting $name..."
        (cd "$REPO_DIR" || exit 1
         nohup "$bin" >>"$log" 2>&1 </dev/null &
         echo $! > "$pid_file")
    fi
}

stop_component() {
    local name="$1"
    local pid_file="$PID_DIR/$name.pid"
    if ! is_running "$name"; then
        rm -f "$pid_file"
        echo "$name not running"
        return
    fi
    local pid
    pid="$(cat "$pid_file")"
    echo "Stopping $name (pid $pid)..."
    if [[ "${NEEDS_SUDO[$name]}" -eq 1 && "$(id -u)" -ne 0 ]]; then
        sudo kill "$pid" 2>/dev/null || true
    else
        kill "$pid" 2>/dev/null || true
    fi
    rm -f "$pid_file"
}

status_component() {
    local name="$1"
    if is_running "$name"; then
        echo "  $name: running (pid $(cat "$PID_DIR/$name.pid"))"
    else
        echo "  $name: stopped"
    fi
}

case "$COMMAND" in
    build)
        build_all
        ;;
    start)
        check_mosquitto
        build_all
        for name in "${COMPONENTS[@]}"; do
            start_component "$name"
        done
        echo
        echo "All components started. Logs in $LOG_DIR, PIDs in $PID_DIR."
        echo "Check status with: scripts/startup.sh status"
        ;;
    stop)
        for ((i=${#COMPONENTS[@]}-1; i>=0; i--)); do
            stop_component "${COMPONENTS[i]}"
        done
        ;;
    restart)
        "$0" stop $([[ "$WITH_CONSOLE" -eq 1 ]] && echo --with-console)
        "$0" start $([[ "$WITH_CONSOLE" -eq 1 ]] && echo --with-console)
        ;;
    status)
        echo "Component status (user: $RUN_USER):"
        for name in "${COMPONENTS[@]}"; do
            status_component "$name"
        done
        ;;
esac
