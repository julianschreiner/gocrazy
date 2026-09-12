// traffic runs a local gateway and generates traffic against the Compose backends.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"gateway/internal/config"
	"gateway/internal/gateway"
)

type request struct {
	id, service, path, backend, failure string
	status                              int
	started                             time.Time
	clientMS, gatewayMS, backendMS      float64
}

type serviceStats struct {
	active, completed, failed, gatewayCount int
	clientMS, gatewayMS                     float64
}

type dashboard struct {
	sync.Mutex
	started time.Time
	stats   map[string]*serviceStats
	active  map[string]*request
	recent  []*request
	skipped int
}

func newDashboard() *dashboard {
	return &dashboard{
		started: time.Now(),
		stats:   map[string]*serviceStats{"users": {}, "orders": {}},
		active:  make(map[string]*request),
	}
}

// observe measures the actual gateway handler, including its wait for upstream.
// It passes the original ResponseWriter through, preserving proxy streaming.
func (d *dashboard) observe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d.Lock()
		req := d.active[r.Header.Get("X-Request-ID")]
		d.Unlock()
		started := time.Now()
		defer func() {
			if req == nil {
				return
			}
			d.Lock()
			defer d.Unlock()
			req.gatewayMS = milliseconds(time.Since(started))
			stats := d.stats[req.service]
			stats.gatewayCount++
			stats.gatewayMS += req.gatewayMS
		}()
		next.ServeHTTP(w, r)
	})
}

func milliseconds(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

func (d *dashboard) send(ctx context.Context, client *http.Client, baseURL string, id int) {
	service := "users"
	if id%3 == 0 {
		service = "orders"
	}
	status := http.StatusOK
	if id%10 == 0 {
		status = http.StatusServiceUnavailable
	}
	delay := 80 + rand.IntN(620)
	path := fmt.Sprintf("/%s?delay_ms=%d&status=%d", service, delay, status)
	req := &request{id: strconv.Itoa(id), service: service, path: path, backend: "-", started: time.Now()}
	d.Lock()
	d.active[req.id] = req
	d.stats[service].active++
	d.Unlock()

	outbound, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	var code int
	var backendMS float64
	backend := "-"
	if err == nil {
		outbound.Header.Set("X-Request-ID", req.id)
		var response *http.Response
		response, err = client.Do(outbound)
		if err == nil {
			code = response.StatusCode
			if name := response.Header.Get("X-Backend"); name != "" {
				backend = name
			}
			backendMS, _ = strconv.ParseFloat(response.Header.Get("X-Backend-Duration-Ms"), 64)
			_, err = io.Copy(io.Discard, response.Body)
			response.Body.Close()
		}
	}

	d.Lock()
	defer d.Unlock()
	req.status, req.backend, req.backendMS = code, backend, backendMS
	req.clientMS = milliseconds(time.Since(req.started))
	if err != nil {
		req.failure = err.Error()
	}
	stats := d.stats[service]
	stats.active--
	stats.completed++
	stats.clientMS += req.clientMS
	if err != nil || code >= 400 {
		stats.failed++
	}
	delete(d.active, req.id)
	d.recent = append(d.recent, req)
	if len(d.recent) > 10 {
		d.recent = d.recent[len(d.recent)-10:]
	}
}

func (d *dashboard) render(out io.Writer, baseURL string, rate, concurrency int, clear bool) {
	d.Lock()
	defer d.Unlock()
	var b strings.Builder
	if clear {
		b.WriteString("\033[H\033[2J")
	}
	elapsed := time.Since(d.started).Seconds()
	total := d.stats["users"].completed + d.stats["orders"].completed
	fmt.Fprintf(&b, "GATEWAY TRAFFIC   %s   elapsed %.0fs\n", baseURL, elapsed)
	fmt.Fprintf(&b, "Target %d req/s | concurrency limit %d | completed %d (%.1f/s) | skipped %d\n\n", rate, concurrency, total, float64(total)/elapsed, d.skipped)
	b.WriteString("Traffic -> gateway -> users  :8081\n                   -> orders :8082\n\n")
	b.WriteString("SERVICE  TRAFFIC SHARE           IN FLIGHT  DONE  ERRORS   AVG CLIENT  AVG GATEWAY\n")
	for _, name := range []string{"users", "orders"} {
		s := d.stats[name]
		bars := 0
		if total > 0 {
			bars = s.completed * 20 / total
		}
		clientAvg, gatewayAvg := 0.0, 0.0
		if s.completed > 0 {
			clientAvg = s.clientMS / float64(s.completed)
		}
		if s.gatewayCount > 0 {
			gatewayAvg = s.gatewayMS / float64(s.gatewayCount)
		}
		fmt.Fprintf(&b, "%-7s  [%-20s] %9d %5d %7d %10.1fms %10.1fms\n", name, strings.Repeat("#", bars), s.active, s.completed, s.failed, clientAvg, gatewayAvg)
	}
	b.WriteString("\nRECENT REQUESTS (backend name confirmed by its response)\n")
	b.WriteString("  ID  BACKEND  STATUS    CLIENT   GATEWAY   BACKEND  PATH\n")
	for i := len(d.recent) - 1; i >= 0; i-- {
		r := d.recent[i]
		status := strconv.Itoa(r.status)
		if r.failure != "" {
			status = "ERR"
		}
		fmt.Fprintf(&b, "%4s  %-7s  %6s %8.1f %9.1f %9.1f  %s\n", r.id, r.backend, status, r.clientMS, r.gatewayMS, r.backendMS, r.path)
	}
	b.WriteString("\nTimes are milliseconds. Gateway = handler time INCLUDING upstream wait.\n")
	b.WriteString("Client = full round trip. Backend = reported processing time (includes demo delay).\n")
	b.WriteString("In flight = requests sent toward each route; backend confirmed on completion.\n")
	b.WriteString("Every 10th request asks for a 503. Skipped = concurrency limit reached.\n")
	b.WriteString("Ctrl+C stops traffic and this demo gateway; Compose backends stay running.\n")
	fmt.Fprint(out, b.String())
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "config/cfg.yaml", "gateway config")
	rate := flag.Int("rate", 8, "requests per second (1-1000)")
	concurrency := flag.Int("concurrency", 8, "maximum simultaneous requests (1-1000)")
	duration := flag.Duration("duration", 0, "stop after this duration; 0 runs until Ctrl+C")
	plain := flag.Bool("plain", false, "print periodic snapshots without terminal escape codes")
	flag.Parse()
	if *rate < 1 || *rate > 1000 || *concurrency < 1 || *concurrency > 1000 || *duration < 0 {
		return fmt.Errorf("rate and concurrency must be between 1 and 1000; duration must be nonnegative")
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	handler, err := gateway.New(cfg)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	defer client.CloseIdleConnections()
	for _, pool := range cfg.UpstreamPools {
		response, err := client.Get(strings.TrimRight(pool.Targets[0], "/") + "/health")
		if err != nil {
			return fmt.Errorf("backend %s unavailable: %w; run sh scripts/traffic-demo.sh", pool.Name, err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("backend %s health check returned %d", pool.Name, response.StatusCode)
		}
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	d := newDashboard()
	server := &http.Server{Handler: d.observe(handler), ReadHeaderTimeout: 5 * time.Second, ErrorLog: log.New(io.Discard, "", 0)}
	defer server.Close()
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- server.Serve(listener) }()
	baseURL := "http://" + listener.Addr().String()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if *duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *duration)
		defer cancel()
	}
	terminal := false
	if info, err := os.Stdout.Stat(); err == nil {
		terminal = info.Mode()&os.ModeCharDevice != 0 && os.Getenv("TERM") != "dumb" && !*plain
	}
	if terminal {
		fmt.Print("\033[?25l")
		defer fmt.Print("\033[?25h\n")
	}
	refreshEvery := 2 * time.Second
	if terminal {
		refreshEvery = 150 * time.Millisecond
	}
	refresh := time.NewTicker(refreshEvery)
	defer refresh.Stop()
	ticks := time.NewTicker(time.Second / time.Duration(*rate))
	defer ticks.Stop()
	slots := make(chan struct{}, *concurrency)
	var workers sync.WaitGroup
	d.render(os.Stdout, baseURL, *rate, *concurrency, terminal)
	id := 0
	var serveErr error
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case serveErr = <-serverErrors:
			break loop
		case <-refresh.C:
			d.render(os.Stdout, baseURL, *rate, *concurrency, terminal)
		case <-ticks.C:
			select {
			case slots <- struct{}{}:
				id++
				workers.Add(1)
				go func(id int) {
					defer workers.Done()
					defer func() { <-slots }()
					// Stop scheduling on Ctrl+C, but allow in-flight requests to finish.
					d.send(context.Background(), client, baseURL, id)
				}(id)
			default:
				d.Lock()
				d.skipped++
				d.Unlock()
			}
		}
	}
	workers.Wait()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}
	d.render(os.Stdout, baseURL, *rate, *concurrency, terminal)
	return serveErr
}
