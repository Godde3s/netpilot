// Package cli wires the netpilot command tree: ports, ping, health, dns, subnet.
package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Godde3s/netpilot/internal/dnsx"
	"github.com/Godde3s/netpilot/internal/health"
	"github.com/Godde3s/netpilot/internal/ping"
	"github.com/Godde3s/netpilot/internal/scanner"
	"github.com/Godde3s/netpilot/internal/subnet"
	"github.com/Godde3s/netpilot/internal/ui"
)

const banner = `
  _   _                _ _ _
 | \ | | ___ _   _  __| | (_)_ __   ___  _ __
 |  \| |/ _ \ | | |/ _  | | | '_ \ / _ \| '_ \
 | |\  |  __/ |_| | (_| | | | | | | (_) | | | |
 |_| \_|\___|\__, |\__,_|_|_|_| |_|\___/|_| |_|
             |___/  network toolkit · v1.0.0
`

const usage = `netpilot — a single-binary network toolkit

Usage:
  netpilot <command> [options]

Commands:
  ports    <host> -p <spec>     concurrent TCP port scan      e.g. -p 22,80,443 or -p 1-1000
  ping     <host>               TCP-based latency probe       e.g. -P 443 -c 10
  health   <url...>             HTTP(S) uptime checks         e.g. https://example.com
  dns      <host>               DNS record lookup             e.g. -t MX  (-T timeout)
  subnet   <cidr>               IPv4 CIDR calculator          e.g. 10.0.4.0/22

Global options:
  -t <seconds>    timeout per probe (default 1.5)
  -h              show help
`

// Run parses argv and dispatches the requested command.
func Run(args []string) error {
	if len(args) == 0 {
		fmt.Print(banner)
		fmt.Print(usage)
		return nil
	}
	switch args[0] {
	case "help", "-h", "--help":
		fmt.Print(banner)
		fmt.Print(usage)
		return nil
	case "ports":
		return cmdPorts(args[1:])
	case "ping":
		return cmdPing(args[1:])
	case "health":
		return cmdHealth(args[1:])
	case "dns":
		return cmdDNS(args[1:])
	case "subnet":
		return cmdSubnet(args[1:])
	default:
		return fmt.Errorf("unknown command %q — run `netpilot help`", args[0])
	}
}

// reorderArgs moves flags before positional arguments so that
// `netpilot subnet 10.0.4.0/22 -n 5` parses the same as flags-first form.
// valueFlags lists long/short flag names (without dash) that consume a value.
func reorderArgs(args []string, valueFlags ...string) []string {
	vals := map[string]bool{}
	for _, v := range valueFlags {
		vals[v] = true
	}
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") {
			positional = append(positional, a)
			continue
		}
		flags = append(flags, a)
		name := strings.TrimLeft(a, "-")
		if eq := strings.IndexByte(a, '='); eq < 0 && vals[name] && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return append(flags, positional...)
}

func timeoutFlag(fs *flag.FlagSet) *float64 {
	return fs.Float64("t", 1.5, "timeout seconds per probe")
}

func cmdPorts(args []string) error {
	fs := flag.NewFlagSet("ports", flag.ExitOnError)
	portsSpec := fs.String("p", "1-1000", "port spec: N, N-M, comma list")
	workers := fs.Int("w", 256, "concurrent workers")
	showClosed := fs.Bool("v", false, "also show closed ports")
	to := timeoutFlag(fs)
	if err := fs.Parse(reorderArgs(args, "p", "w", "v", "t")); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: netpilot ports <host> -p <spec>")
	}
	host := fs.Arg(0)
	ports, err := scanner.ParsePorts(*portsSpec)
	if err != nil {
		return err
	}
	timeout := time.Duration(*to * float64(time.Second))
	ui.Info("scanning %s — %d ports · %d workers", ui.Bold(host), len(ports), *workers)
	start := time.Now()
	open := scanner.ScanTCP(host, ports, scanner.Options{Workers: *workers, Timeout: timeout, ShowClosed: *showClosed})
	ui.Info("done in %s — %d open", time.Since(start).Round(time.Millisecond), len(open))
	for _, p := range open {
		ui.Dim("%s:%d", host, p)
	}
	return nil
}

func cmdPing(args []string) error {
	fs := flag.NewFlagSet("ping", flag.ExitOnError)
	port := fs.Int("P", 443, "TCP port to probe")
	count := fs.Int("c", 4, "number of probes")
	interval := fs.Float64("i", 0.4, "seconds between probes")
	to := timeoutFlag(fs)
	if err := fs.Parse(reorderArgs(args, "P", "c", "i", "t")); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: netpilot ping <host> -P <port> -c <count>")
	}
	timeout := time.Duration(*to * float64(time.Second))
	st, err := ping.Run(fs.Arg(0), *port, *count, time.Duration(*interval*float64(time.Second)), timeout)
	if err != nil {
		return err
	}
	ping.Print(st)
	return nil
}

func cmdHealth(args []string) error {
	fs := flag.NewFlagSet("health", flag.ExitOnError)
	head := fs.Bool("head", false, "use HEAD requests")
	followTls := fs.Bool("k", false, "(kept for familiarity) skip TLS verify is NOT supported — checks always verify")
	to := timeoutFlag(fs)
	if err := fs.Parse(reorderArgs(args, "head", "k", "t")); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: netpilot health <url...>")
	}
	timeout := time.Duration(*to * float64(time.Second))
	c := health.Client(timeout)
	client := c
	if *followTls {
		ui.Warn("note: TLS is always verified in netpilot")
	}
	var failures int
	for _, u := range fs.Args() {
		if !strings.Contains(u, "://") {
			u = "https://" + u
		}
		o := health.Check(client, u, *head)
		health.Report(o)
		if !o.Up {
			failures++
		}
	}
	if failures > 0 {
		os.Exit(2)
	}
	return nil
}

func cmdDNS(args []string) error {
	fs := flag.NewFlagSet("dns", flag.ExitOnError)
	rtype := fs.String("t", "A", "record type: A, AAAA, CNAME, MX, NS, TXT, PTR")
	to := fs.Float64("T", 6, "timeout seconds for resolution")
	if err := fs.Parse(reorderArgs(args, "t")); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: netpilot dns <host> -t <type>")
	}
	ctx, cancel := dnsx.EnsureTimeout(time.Duration(*to * 4 * float64(time.Second)))
	defer cancel()
	vals, err := dnsx.Lookup(ctx, fs.Arg(0), *rtype)
	if err != nil {
		return err
	}
	dnsx.Report(fs.Arg(0), *rtype, vals)
	return nil
}

func cmdSubnet(args []string) error {
	fs := flag.NewFlagSet("subnet", flag.ExitOnError)
	list := fs.Int("n", 0, "enumerate first N hosts")
	if err := fs.Parse(reorderArgs(args, "n")); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: netpilot subnet <cidr> [-n N]")
	}
	inf, err := subnet.Parse(fs.Arg(0))
	if err != nil {
		return err
	}
	subnet.Report(inf)
	if *list > 0 {
		hosts := subnet.Enumerate(inf, *list)
		ui.Info("first %d host(s)", len(hosts))
		for _, h := range hosts {
			ui.Dim("%s", h)
		}
	}
	return nil
}
