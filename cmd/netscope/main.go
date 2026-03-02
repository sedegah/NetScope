package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"netscope/internal/config"
	"netscope/internal/monitor"
	"netscope/internal/store"
	"netscope/internal/web"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	sub := os.Args[1]
	switch sub {
	case "monitor":
		runMonitor(os.Args[2:])
	case "web":
		runWeb(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func runMonitor(args []string) {
	fs := flag.NewFlagSet("monitor", flag.ExitOnError)
	configPath := fs.String("config", "devices.json", "path to JSON device config")
	interval := fs.Duration("interval", 10*time.Second, "probe interval")
	probes := fs.Int("probes", 4, "icmp probes per cycle")
	cycles := fs.Int("cycles", 0, "number of cycles (0=continuous)")
	autoSubnet := fs.String("auto-subnet", "", "CIDR subnet for automatic host discovery (e.g. 192.168.1.0/24)")
	autoMethod := fs.String("auto-method", "auto", "discovery method: auto|ping")
	autoRefresh := fs.Duration("auto-refresh", 30*time.Second, "auto-discovery refresh interval")
	_ = fs.Parse(args)

	s := store.NewMemoryStore()
	service := monitor.NewService(s)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	provider := newDeviceProvider(nil)
	if *autoSubnet != "" {
		if err := startAutoDiscovery(ctx, provider, *autoSubnet, *autoMethod, *autoRefresh); err != nil {
			log.Fatal(err)
		}
	} else {
		cfg, err := config.Load(*configPath)
		if err != nil {
			log.Fatal(err)
		}
		provider.Set(cfg.Devices)
	}

	cycle := 0
	for {
		cycle++
		fmt.Printf("\nCycle %d (%s)\n", cycle, time.Now().Format(time.RFC3339))
		fmt.Println("NAME\tADDRESS\tSTATUS\tLAT(ms)\tLOSS(%)")
		devices := provider.List()
		for _, d := range devices {
			snap := service.ProbeDevice(ctx, d, *probes, 1200*time.Millisecond)
			status := "DOWN"
			if snap.Online {
				status = "UP"
			}
			fmt.Printf("%s\t%s\t%s\t%.2f\t%.1f\n", snap.Name, snap.Address, status, snap.LatencyMS, snap.PacketLoss)
		}
		if len(devices) == 0 {
			fmt.Println("(no discovered devices yet)")
		}

		if *cycles > 0 && cycle >= *cycles {
			return
		}

		select {
		case <-ctx.Done():
			fmt.Println("stopped")
			return
		case <-time.After(*interval):
		}
	}
}

func runWeb(args []string) {
	fs := flag.NewFlagSet("web", flag.ExitOnError)
	configPath := fs.String("config", "devices.json", "path to JSON device config")
	interval := fs.Duration("interval", 10*time.Second, "probe interval")
	probes := fs.Int("probes", 4, "icmp probes per cycle")
	listen := fs.String("listen", ":8080", "http listen address")
	autoSubnet := fs.String("auto-subnet", "", "CIDR subnet for automatic host discovery (e.g. 192.168.1.0/24)")
	autoMethod := fs.String("auto-method", "auto", "discovery method: auto|ping")
	autoRefresh := fs.Duration("auto-refresh", 30*time.Second, "auto-discovery refresh interval")
	_ = fs.Parse(args)

	s := store.NewMemoryStore()
	service := monitor.NewService(s)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	provider := newDeviceProvider(nil)
	if *autoSubnet != "" {
		if err := startAutoDiscovery(ctx, provider, *autoSubnet, *autoMethod, *autoRefresh); err != nil {
			log.Fatal(err)
		}
	} else {
		cfg, err := config.Load(*configPath)
		if err != nil {
			log.Fatal(err)
		}
		provider.Set(cfg.Devices)
	}

	go func() {
		ticker := time.NewTicker(*interval)
		defer ticker.Stop()
		for {
			for _, d := range provider.List() {
				service.ProbeDevice(ctx, d, *probes, 1200*time.Millisecond)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()

	h := web.NewHandler(s)
	server := &http.Server{Addr: *listen, Handler: h}
	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()

	log.Printf("NetScope dashboard at http://localhost%s\n", *listen)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func usage() {
	fmt.Println("NetScope - Network Visibility & Health Monitoring")
	fmt.Println("Usage:")
	fmt.Println("  netscope monitor -config devices.json")
	fmt.Println("  netscope monitor -auto-subnet 192.168.1.0/24 -auto-method auto -auto-refresh 30s")
	fmt.Println("  netscope web -config devices.json -listen :8080")
	fmt.Println("  netscope web -auto-subnet 192.168.1.0/24 -auto-refresh 30s -listen :8080")
}
