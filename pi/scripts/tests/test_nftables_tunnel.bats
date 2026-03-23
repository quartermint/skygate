#!/usr/bin/env bats
# BATS tests for SkyGate nftables tunnel rules (Phase 3)
# Validates nftables.conf.j2 template contains required tunnel marking and forwarding rules.
# Run: bats pi/scripts/tests/test_nftables_tunnel.bats

TEMPLATE="${BATS_TEST_DIRNAME}/../../ansible/roles/networking/templates/nftables.conf.j2"

setup() {
    [ -f "$TEMPLATE" ] || skip "nftables.conf.j2 not found at $TEMPLATE"
}

@test "nftables template exists" {
    [ -f "$TEMPLATE" ]
}

@test "nftables has bypass_v4 set (Phase 1)" {
    grep -q "set bypass_v4" "$TEMPLATE"
}

@test "nftables has fwmark 0x1 for bypass traffic (Phase 1)" {
    grep -q "meta mark set 0x1" "$TEMPLATE"
}

@test "nftables has fwmark 0x2 for tunnel traffic (Phase 3)" {
    grep -q "meta mark set 0x2" "$TEMPLATE"
}

@test "nftables tunnel marking is conditional on wg_enabled" {
    grep -q 'wg_enabled | default(false)' "$TEMPLATE"
}

@test "nftables has AP-to-wg0 forwarding rule" {
    grep -q 'oifname "wg0"' "$TEMPLATE"
}

@test "nftables has WireGuard input port rule" {
    grep -q 'udp dport {{ wg_listen_port' "$TEMPLATE"
}

@test "nftables has MSS clamping on wg0" {
    grep -q "maxseg size set 1380" "$TEMPLATE"
}

@test "nftables has generic ct mark restore (supports 0x1 and 0x2)" {
    grep -q 'ct mark != 0x0 meta mark set ct mark' "$TEMPLATE"
}

@test "nftables does NOT have old single-mark ct restore" {
    ! grep -q 'ct mark 0x1 meta mark set ct mark' "$TEMPLATE"
}

@test "nftables preserves Phase 1 uplink masquerade" {
    grep -q 'oifname "{{ uplink_interface }}" masquerade' "$TEMPLATE"
}

@test "nftables preserves Phase 1 AP-to-uplink forwarding" {
    grep -q 'iifname "{{ ap_interface }}" oifname "{{ uplink_interface }}" accept' "$TEMPLATE"
}

@test "nftables tunnel mark applied only to non-bypass AP traffic" {
    grep -q 'ip daddr != @bypass_v4 meta mark set 0x2' "$TEMPLATE"
}

@test "nftables has exactly 4 wg_enabled conditional blocks" {
    local count
    count=$(grep -c 'wg_enabled | default(false)' "$TEMPLATE")
    [ "$count" -eq 4 ]
}
