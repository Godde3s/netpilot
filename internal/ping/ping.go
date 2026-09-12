// Package ping implements latency measurement over TCP — useful in
// environments where ICMP is blocked or requires privileges.
package ping

import (
	"fmt"
	"math"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/Godde3s/netpilot/internal/ui"
)

// Stats aggregates the results of a ping run.
type Stats struct {
	Sent   int
	Recv   int
	Lost   int
	Min    time.Duration
	Max    time.Duration
	Avg    time.Duration
	Jitter time.Duration
}

// Run performs count TCP dials against host:port and prints live results.
func Run(host string, port, count int, interval, timeout time.Duration) (*Stats, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	st := &Stats{Min: time.Duration(math.MaxInt64)}
	var samples []time.Duration
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 1; i <= count; i++ {
		seq := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			conn, err := net.DialTimeout("tcp", addr, timeout)
			rtt := time.Since(start)
			if conn != nil {
				conn.Close()
			}
			mu.Lock()
			defer mu.Unlock()
			st.Sent++
			if err != nil {
				st.Lost++
				ui.Fail("seq=%d %s", seq, err)
				return
			}
			st.Recv++
			samples = append(samples, rtt)
			if rtt < st.Min {
				st.Min = rtt
			}
			if rtt > st.Max {
				st.Max = rtt
			}
			ui.Success("seq=%d from %s time=%s", seq, addr, ui.Cyan(rtt.Round(time.Millisecond).String()))
		}()
		if i < count {
			time.Sleep(interval)
		}
	}
	wg.Wait()

	if len(samples) > 0 {
		var sum time.Duration
		for _, s := range samples {
			sum += s
		}
		st.Avg = sum / time.Duration(len(samples))
		var varSum float64
		for _, s := range samples {
			d := float64(s - st.Avg)
			varSum += d * d
		}
		st.Jitter = time.Duration(math.Sqrt(varSum / float64(len(samples))))
	}
	if st.Min == time.Duration(math.MaxInt64) {
		st.Min = 0
	}
	return st, nil
}

// Print renders the summary block.
func Print(st *Stats) {
	var lossPct float64
	if st.Sent > 0 {
		lossPct = float64(st.Lost) / float64(st.Sent) * 100
	}
	health := ui.Green("healthy")
	switch {
	case lossPct == 100:
		health = ui.Red("down")
	case lossPct > 0:
		health = ui.Yellow("degraded")
	}
	fmt.Println()
	ui.Info("target summary — %s", health)
	ui.Dim("sent %d · recv %d · loss %.1f%%", st.Sent, st.Recv, lossPct)
	if st.Recv > 0 {
		ui.Dim("min %s · avg %s · max %s · jitter %s",
			st.Min.Round(time.Microsecond).String(), st.Avg.Round(time.Microsecond).String(), st.Max.Round(time.Microsecond).String(), st.Jitter.Round(time.Microsecond).String())
	}
}
