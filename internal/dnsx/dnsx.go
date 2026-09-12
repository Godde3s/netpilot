// Package dnsx offers quick DNS record inspection.
package dnsx

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/Godde3s/netpilot/internal/ui"
)

// Lookup runs a plain resolver query for the given record type.
func Lookup(ctx context.Context, host, rtype string) ([]string, error) {
	res := &net.Resolver{PreferGo: true}
	var err error
	var vals []string
	switch strings.ToUpper(rtype) {
	case "A":
		vals, err = res.LookupHost(ctx, host)
		if err == nil {
			vals = filterIPv4(vals)
		}
	case "AAAA":
		vals, err = res.LookupHost(ctx, host)
		if err == nil {
			vals = filterIPv6(vals)
		}
	case "CNAME":
		var cname string
		cname, err = res.LookupCNAME(ctx, host)
		vals = []string{cname}
	case "MX":
		var mxs []*net.MX
		mxs, err = res.LookupMX(ctx, host)
		for _, m := range mxs {
			vals = append(vals, fmt.Sprintf("%d %s", m.Pref, strings.TrimSuffix(m.Host, ".")))
		}
	case "NS":
		var nss []*net.NS
		nss, err = res.LookupNS(ctx, host)
		for _, n := range nss {
			vals = append(vals, strings.TrimSuffix(n.Host, "."))
		}
	case "TXT":
		var txts []string
		txts, err = res.LookupTXT(ctx, host)
		vals = txts
	case "PTR":
		var names []string
		names, err = res.LookupAddr(ctx, host)
		vals = names
	default:
		return nil, fmt.Errorf("unsupported record type %q (use A, AAAA, CNAME, MX, NS, TXT, PTR)", rtype)
	}
	if err != nil {
		return nil, err
	}
	sort.Strings(vals)
	return vals, nil
}

// Report prints the results of a lookup.
func Report(host, rtype string, vals []string) {
	if len(vals) == 0 {
		ui.Warn("%s %s — no records", host, strings.ToUpper(rtype))
		return
	}
	ui.Info("%s %s — %d record(s)", host, ui.Bold(strings.ToUpper(rtype)), len(vals))
	for _, v := range vals {
		ui.Dim("%s", v)
	}
}

func filterIPv4(in []string) []string {
	var out []string
	for _, s := range in {
		if net.ParseIP(s).To4() != nil {
			out = append(out, s)
		}
	}
	return out
}

func filterIPv6(in []string) []string {
	var out []string
	for _, s := range in {
		if ip := net.ParseIP(s); ip != nil && ip.To4() == nil {
			out = append(out, s)
		}
	}
	return out
}

// EnsureTimeout returns a context with the given deadline.
func EnsureTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
