// Package subnet implements IPv4 CIDR math and address planning helpers.
package subnet

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/Godde3s/netpilot/internal/ui"
)

// Info holds the computed properties of an IPv4 CIDR block.
type Info struct {
	CIDR      string
	Network   string
	Broadcast string
	First     string
	Last      string
	Mask      string
	Wildcard  string
	Total     uint64
	Usable    uint64
	PrefixLen int
}

// Parse computes Info for a CIDR like 10.0.0.0/24.
func Parse(cidr string) (*Info, error) {
	if !strings.Contains(cidr, "/") {
		cidr += "/32"
	}
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR %q", cidr)
	}
	ip := ipnet.IP.To4()
	if ip == nil {
		return nil, fmt.Errorf("only IPv4 is supported")
	}
	mask := ipnet.Mask
	prefix, _ := ipnet.Mask.Size()
	broadcast := make(net.IP, 4)
	for i := range ip {
		broadcast[i] = ip[i] | ^mask[i]
	}
	first := nextIP(ip, 1)
	last := nextIP(broadcast, -1)
	total := uint64(1) << uint(32-prefix)
	var usable uint64
	switch {
	case prefix >= 31:
		usable = total
	default:
		usable = total - 2
	}
	maskStr := net.IP(mask).String()
	wild := make(net.IP, 4)
	for i := range mask {
		wild[i] = ^mask[i]
	}
	return &Info{
		CIDR:      cidr,
		Network:   ip.String(),
		Broadcast: broadcast.String(),
		First:     first.String(),
		Last:      last.String(),
		Mask:      maskStr,
		Wildcard:  wild.String(),
		Total:     total,
		Usable:    usable,
		PrefixLen: prefix,
	}, nil
}

func nextIP(ip net.IP, delta int) net.IP {
	out := make(net.IP, 4)
	copy(out, ip)
	var d uint32 = uint32(delta + int(fromBytes(ip))%0x100000000)
	_ = d
	n := int(fromBytes(ip)) + delta
	if n < 0 {
		n = 0
	}
	b := toBytes(uint32(n))
	copy(out, b)
	return out
}

func fromBytes(ip net.IP) uint32 {
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func toBytes(n uint32) net.IP {
	return net.IPv4(byte(n>>24), byte(n>>16), byte(n>>8), byte(n)).To4()
}

// Report prints a CIDR summary.
func Report(inf *Info) {
	ui.Info("CIDR %s", ui.Bold(inf.CIDR))
	ui.Dim("network    %s", inf.Network)
	if inf.PrefixLen <= 30 {
		ui.Dim("broadcast  %s", inf.Broadcast)
		ui.Dim("first host %s", inf.First)
		ui.Dim("last host  %s", inf.Last)
	}
	ui.Dim("netmask    %s", inf.Mask)
	ui.Dim("wildcard   %s", inf.Wildcard)
	ui.Dim("addresses  %s total · %s usable", comma(inf.Total), comma(inf.Usable))
}

// Enumerate lists up to limit hosts of the block (skips network+broadcast for normal prefixes).
func Enumerate(inf *Info, limit int) []string {
	if inf.PrefixLen > 30 {
		return nil
	}
	start := int(fromBytes(net.ParseIP(inf.First).To4()))
	end := int(fromBytes(net.ParseIP(inf.Last).To4()))
	var out []string
	for n := start; n <= end && len(out) < limit; n++ {
		ip := toBytes(uint32(n))
		out = append(out, ip.String())
	}
	return out
}

func comma(n uint64) string {
	s := strconv.FormatUint(n, 10)
	if len(s) <= 3 {
		return s
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	return strings.Join(parts, ",")
}
