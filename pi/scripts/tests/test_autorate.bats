#!/usr/bin/env bats
# BATS tests for SkyGate autorate.sh
# Tests the calculate_rate function logic without requiring tc or fping.
# Run: bats pi/scripts/tests/test_autorate.bats

setup() {
    # Source autorate.sh to get access to functions
    export INTERFACE="eth0"
    export REFLECTOR="1.1.1.1"
    export MIN_RATE_KBPS=5000
    export BASE_RATE_KBPS=20000
    export MAX_RATE_KBPS=100000
    export BASELINE_RTT_MS=40
    export THRESHOLD_MS=15
    export INCREASE_STEP_KBPS=5000
    export DECREASE_FACTOR="0.7"
    export STABLE_THRESHOLD=5
    export INTERVAL_S=2
    export DRY_RUN=true

    source "${BATS_TEST_DIRNAME}/../autorate.sh"
    export -f calculate_rate apply_cake measure_rtt
}

@test "calculate_rate decreases on high RTT (bufferbloat)" {
    # RTT 70ms with baseline 40ms = excess 30ms > threshold 15ms
    result=$(calculate_rate 20000 70 0)
    new_rate=$(echo "$result" | awk '{print $1}')
    new_stable=$(echo "$result" | awk '{print $2}')

    # 20000 * 0.7 = 14000
    [ "$new_rate" -eq 14000 ]
    [ "$new_stable" -eq 0 ]
}

@test "calculate_rate does not go below minimum" {
    # Already at minimum, high RTT
    result=$(calculate_rate 5000 70 0)
    new_rate=$(echo "$result" | awk '{print $1}')

    [ "$new_rate" -eq 5000 ]
}

@test "calculate_rate increases after stable period" {
    # RTT 45ms with baseline 40ms = excess 5ms < threshold 15ms
    # stable_count = 4 (one below threshold of 5)
    result=$(calculate_rate 20000 45 4)
    new_rate=$(echo "$result" | awk '{print $1}')
    new_stable=$(echo "$result" | awk '{print $2}')

    # stable_count becomes 5 >= STABLE_THRESHOLD, so increase by 5000
    [ "$new_rate" -eq 25000 ]
    [ "$new_stable" -eq 0 ]
}

@test "calculate_rate does not exceed maximum" {
    # At 98000, stable for long enough, should cap at 100000
    result=$(calculate_rate 98000 45 4)
    new_rate=$(echo "$result" | awk '{print $1}')

    [ "$new_rate" -eq 100000 ]
}

@test "calculate_rate holds steady during stable period below threshold" {
    # RTT stable but haven't reached STABLE_THRESHOLD yet
    result=$(calculate_rate 20000 45 2)
    new_rate=$(echo "$result" | awk '{print $1}')
    new_stable=$(echo "$result" | awk '{print $2}')

    [ "$new_rate" -eq 20000 ]
    [ "$new_stable" -eq 3 ]
}

@test "calculate_rate resets stable count on bufferbloat" {
    # Was stable for 4 cycles, then bufferbloat hits
    result=$(calculate_rate 30000 70 4)
    new_stable=$(echo "$result" | awk '{print $2}')

    [ "$new_stable" -eq 0 ]
}

@test "calculate_rate handles exact threshold boundary" {
    # RTT exactly at baseline + threshold (40 + 15 = 55)
    # excess = 15, which is NOT > threshold (equal, not greater)
    result=$(calculate_rate 20000 55 0)
    new_rate=$(echo "$result" | awk '{print $1}')

    # Should NOT decrease (threshold check is >, not >=)
    [ "$new_rate" -eq 20000 ]
}

@test "apply_cake dry run outputs tc command" {
    output=$(apply_cake 20000)
    [[ "$output" == *"tc qdisc change"* ]]
    [[ "$output" == *"20000kbit"* ]]
}

@test "successive decreases converge toward minimum" {
    rate=50000
    for i in 1 2 3 4 5; do
        result=$(calculate_rate "$rate" 70 0)
        rate=$(echo "$result" | awk '{print $1}')
    done
    # After 5 consecutive decreases from 50000:
    # 50000 -> 35000 -> 24500 -> 17150 -> 12005 -> 8403
    [ "$rate" -gt "$MIN_RATE_KBPS" ]
    [ "$rate" -lt 10000 ]
}
