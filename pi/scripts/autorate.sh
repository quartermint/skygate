#!/bin/bash
# SkyGate CAKE Autorate for Starlink
# Dynamically adjusts CAKE bandwidth based on RTT measurements.
# Simplified from cake-autorate algorithm (OpenWrt) for Raspberry Pi OS.
#
# Dependencies: fping, iproute2 (tc), bc
# Usage: autorate.sh [--dry-run]
#
# Environment overrides (or defaults):
#   INTERFACE, REFLECTOR, MIN_RATE_KBPS, BASE_RATE_KBPS, MAX_RATE_KBPS,
#   BASELINE_RTT_MS, THRESHOLD_MS, INCREASE_STEP_KBPS, DECREASE_FACTOR,
#   STABLE_THRESHOLD, INTERVAL_S

set -euo pipefail

# Configuration (overridable via environment)
INTERFACE="${INTERFACE:-eth0}"
REFLECTOR="${REFLECTOR:-1.1.1.1}"
MIN_RATE_KBPS="${MIN_RATE_KBPS:-5000}"
BASE_RATE_KBPS="${BASE_RATE_KBPS:-20000}"
MAX_RATE_KBPS="${MAX_RATE_KBPS:-100000}"
BASELINE_RTT_MS="${BASELINE_RTT_MS:-40}"
THRESHOLD_MS="${THRESHOLD_MS:-15}"
INCREASE_STEP_KBPS="${INCREASE_STEP_KBPS:-5000}"
DECREASE_FACTOR="${DECREASE_FACTOR:-0.7}"
STABLE_THRESHOLD="${STABLE_THRESHOLD:-5}"
INTERVAL_S="${INTERVAL_S:-2}"

# WireGuard tunnel CAKE (static bandwidth, no autorate)
WG_ENABLED="${WG_ENABLED:-false}"
WG_INTERFACE="${WG_INTERFACE:-wg0}"
WG_CAKE_RATE_KBPS="${WG_CAKE_RATE_KBPS:-15000}"

# Internal state
CURRENT_RATE="$BASE_RATE_KBPS"
STABLE_COUNT=0
DRY_RUN="${DRY_RUN:-false}"

# Parse arguments
for arg in "$@"; do
    case "$arg" in
        --dry-run) DRY_RUN=true ;;
    esac
done

# --- Functions (exported for testing) ---

# Measure RTT to reflector. Returns average RTT in ms (integer).
# Returns 999 on failure.
measure_rtt() {
    local rtt
    rtt=$(fping -c 3 -p 200 -q "$REFLECTOR" 2>&1 | grep -oP 'avg = \K[0-9.]+' || echo "999")
    echo "${rtt%.*}"
}

# Calculate new rate based on RTT measurement.
# Args: $1=current_rate, $2=rtt_ms, $3=stable_count
# Outputs: new_rate new_stable_count
calculate_rate() {
    local current_rate="$1"
    local rtt_ms="$2"
    local stable="$3"
    local excess=$((rtt_ms - BASELINE_RTT_MS))
    local new_rate="$current_rate"
    local new_stable="$stable"

    if [ "$excess" -gt "$THRESHOLD_MS" ]; then
        # Bufferbloat detected: decrease
        new_rate=$(echo "$current_rate * $DECREASE_FACTOR" | bc | cut -d. -f1)
        [ "$new_rate" -lt "$MIN_RATE_KBPS" ] && new_rate="$MIN_RATE_KBPS"
        new_stable=0
    else
        # Stable: count consecutive stable cycles
        new_stable=$((stable + 1))
        if [ "$new_stable" -ge "$STABLE_THRESHOLD" ]; then
            new_rate=$((current_rate + INCREASE_STEP_KBPS))
            [ "$new_rate" -gt "$MAX_RATE_KBPS" ] && new_rate="$MAX_RATE_KBPS"
            new_stable=0
        fi
    fi

    echo "$new_rate $new_stable"
}

# Apply CAKE bandwidth via tc.
apply_cake() {
    local rate_kbps="$1"
    if [ "$DRY_RUN" = true ]; then
        echo "DRY_RUN: tc qdisc change dev $INTERFACE root cake bandwidth ${rate_kbps}kbit"
        return 0
    fi
    tc qdisc change dev "$INTERFACE" root cake bandwidth "${rate_kbps}kbit" 2>/dev/null || \
    tc qdisc replace dev "$INTERFACE" root cake bandwidth "${rate_kbps}kbit"
}

# Apply CAKE bandwidth on WireGuard interface (static, no autorate).
apply_wg_cake() {
    local rate_kbps="$1"
    if [ "$DRY_RUN" = true ]; then
        echo "DRY_RUN: tc qdisc change dev $WG_INTERFACE root cake bandwidth ${rate_kbps}kbit"
        return 0
    fi
    tc qdisc change dev "$WG_INTERFACE" root cake bandwidth "${rate_kbps}kbit" 2>/dev/null || \
    tc qdisc replace dev "$WG_INTERFACE" root cake bandwidth "${rate_kbps}kbit"
}

# --- Main loop ---

main() {
    echo "[autorate] Starting SkyGate CAKE autorate on $INTERFACE"
    echo "[autorate] Rate range: ${MIN_RATE_KBPS}-${MAX_RATE_KBPS} kbps, baseline RTT: ${BASELINE_RTT_MS}ms"

    # Initialize CAKE
    apply_cake "$CURRENT_RATE"
    echo "[autorate] Initial CAKE bandwidth: ${CURRENT_RATE}kbit"

    if [ "$WG_ENABLED" = "true" ]; then
        apply_wg_cake "$WG_CAKE_RATE_KBPS"
        echo "[autorate] WireGuard CAKE bandwidth: ${WG_CAKE_RATE_KBPS}kbit (static)"
    fi

    while true; do
        local rtt
        rtt=$(measure_rtt)

        local result
        result=$(calculate_rate "$CURRENT_RATE" "$rtt" "$STABLE_COUNT")
        local new_rate new_stable
        new_rate=$(echo "$result" | awk '{print $1}')
        new_stable=$(echo "$result" | awk '{print $2}')

        if [ "$new_rate" -ne "$CURRENT_RATE" ]; then
            apply_cake "$new_rate"
            echo "[autorate] RTT=${rtt}ms excess=$((rtt - BASELINE_RTT_MS))ms rate: ${CURRENT_RATE} -> ${new_rate} kbps"
            CURRENT_RATE="$new_rate"
        fi
        STABLE_COUNT="$new_stable"

        sleep "$INTERVAL_S"
    done
}

# Only run main if not being sourced (for testing)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main
fi
