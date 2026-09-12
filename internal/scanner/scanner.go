// Package scanner implements a fast, concurrent TCP port scanner.
package scanner

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/Godde3s/netpilot/internal/ui"
)

// Result describes the outcome of probing one port.
type Result struct {
	Port int
	Open bool
	Err  error
	Took time.Duration
}

// Options tunes the scan behaviour.
type Options struct {
	Workers    int
	Timeout    time.Duration
	ShowClosed bool
}

// ParsePorts expands expressions like "80", "1-1024" or "22,80,443" into a sorted port list.
func ParsePorts(expr string) ([]int, error) {
	var ports []int
	seen := map[int]bool{}
	add := func(lo, hi int) error {
		if lo < 1 || hi > 65535 || lo > hi {
			return fmt.Errorf("invalid port range %d-%d", lo, hi)
		}
		for p := lo; p <= hi; p++ {
			if !seen[p] {
				seen[p] = true
				ports = append(ports, p)
			}
		}
		return nil
	}
	part := ""
	flush := func() error {
		if part == "" {
			return nil
		}
		if lo, hi, ok := splitRange(part); ok {
			return add(lo, hi)
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return fmt.Errorf("invalid port %q", part)
		}
		return add(n, n)
	}
	for i := 0; i <= len(expr); i++ {
		if i == len(expr) || expr[i] == ',' {
			if err := flush(); err != nil {
				return nil, err
			}
			part = ""
			continue
		}
		part += string(expr[i])
	}
	sort.Ints(ports)
	if len(ports) == 0 {
		return nil, fmt.Errorf("no ports to scan")
	}
	return ports, nil
}

func splitRange(s string) (int, int, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == '-' {
			lo, err1 := strconv.Atoi(s[:i])
			hi, err2 := strconv.Atoi(s[i+1:])
			if err1 != nil || err2 != nil {
				return 0, 0, false
			}
			return lo, hi, true
		}
	}
	return 0, 0, false
}

// ScanTCP probes the given ports concurrently and prints results as they arrive.
func ScanTCP(host string, ports []int, opt Options) (open []int) {
	if opt.Workers <= 0 {
		opt.Workers = 256
	}
	if opt.Timeout <= 0 {
		opt.Timeout = 1500 * time.Millisecond
	}
	jobs := make(chan int, len(ports))
	var mu sync.Mutex
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for p := range jobs {
			start := time.Now()
			conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(p)), opt.Timeout)
			took := time.Since(start)
			if err == nil {
				conn.Close()
				mu.Lock()
				open = append(open, p)
				mu.Unlock()
				ui.Success("port %-6d open   %s", p, ui.Gray(took.Round(time.Millisecond).String()))
			} else if opt.ShowClosed {
				ui.Dim("port %-6d closed %s", p, ui.Gray(took.Round(time.Millisecond).String()))
			}
		}
	}
	for i := 0; i < opt.Workers && i < len(ports); i++ {
		wg.Add(1)
		go worker()
	}
	for _, p := range ports {
		jobs <- p
	}
	close(jobs)
	wg.Wait()
	sort.Ints(open)
	return open
}
