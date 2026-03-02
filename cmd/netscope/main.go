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
	_ = fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	s := store.NewMemoryStore()
	service := monitor.NewService(s)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cycle := 0
	for {
		cycle++
		fmt.Printf("\nCycle %d (%s)\n", cycle, time.Now().Format(time.RFC3339))
		fmt.Println("NAME\tADDRESS\tSTATUS\tLAT(ms)\tLOSS(%)")
		for _, d := range cfg.Devices {
			snap := service.ProbeDevice(ctx, d, *probes, 1200*time.Millisecond)
			status := "DOWN"
			if snap.Online {
				status = "UP"
			}
			fmt.Printf("%s\t%s\t%s\t%s\t%.2f\t%.1f\n", snap.Name, snap.Address, d.Type, status, snap.LatencyMS, snap.PacketLoss)
			fmt.Printf("%s\t%s\t%s\t%.2f\t%.1f\n", snap.Name, snap.Address, status, snap.LatencyMS, snap.PacketLoss)
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
	autoSubnet := fs.String("auto-subnet", "", "auto discovery subnet CIDR (enables autonomous mode)")
	autoMethod := fs.String("auto-method", string(discovery.MethodAuto), "auto discovery method: auto|nmap|ping|arp")
	autoRefresh := fs.Duration("auto-refresh", 30*time.Second, "device rediscovery interval in autonomous mode")
	autoTimeout := fs.Duration("auto-timeout", 900*time.Millisecond, "ping timeout for autonomous discovery")
	autoWorkers := fs.Int("auto-workers", 64, "workers for autonomous ping discovery")
	_ = fs.Parse(args)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	current := make([]config.Device, 0)
	discoverAndPersist := func() []config.Device {
		dctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		devices, err := discovery.Discover(dctx, discovery.Method(*autoMethod), *autoSubnet, *autoTimeout, *autoWorkers)
		if err != nil {
			log.Printf("discovery warning: %v", err)
			return nil
		}
		if err := config.Save(*configPath, config.Config{Devices: devices}); err != nil {
			log.Printf("save warning: %v", err)
		}
		return devices
	}

	if *autoSubnet != "" {
		current = discoverAndPersist()
	} else {
		cfg, err := config.Load(*configPath)
		if err != nil {
			log.Fatal(err)
		}
		current = cfg.Devices
	_ = fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	s := store.NewMemoryStore()
	service := monitor.NewService(s)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		ticker := time.NewTicker(*interval)
		defer ticker.Stop()
		lastDiscovery := time.Now()

		for {
			if *autoSubnet != "" && time.Since(lastDiscovery) >= *autoRefresh {
				if refreshed := discoverAndPersist(); len(refreshed) > 0 {
					current = refreshed
				}
				lastDiscovery = time.Now()
			}

			for _, d := range current {
		for {
			for _, d := range cfg.Devices {
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
	fmt.Println("  netscope discover -subnet 192.168.0.0/24 [-method auto|nmap|ping|arp] [-output devices.json]")
	fmt.Println("  netscope monitor -config devices.json [-auto-subnet 192.168.0.0/24 -auto-method auto -auto-refresh 30s]")
	fmt.Println("  netscope web -config devices.json -listen :8080 [-auto-subnet 192.168.0.0/24 -auto-method auto -auto-refresh 30s]")
	fmt.Println("  netscope monitor -config devices.json")
	fmt.Println("  netscope web -config devices.json -listen :8080")
}
