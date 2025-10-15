package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	fmt.Println("=== Mobula Pulse V2 Monitor ===")
	fmt.Println("Monitoring NEW POOL CREATION across multiple chains")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	config, err := loadEnv()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Metrics will be exposed on :2112/metrics for Prometheus")
	fmt.Println()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup
	stopChan := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println(" Starting Prometheus metrics server on :2112")
		if err := StartMetricsServer(":2112"); err != nil {
			fmt.Printf("⚠ Metrics server error: %v\n", err)
		}
	}()

	// Mobula Pulse V2 monitor only
	wg.Add(1)
	go func() {
		defer wg.Done()
		runMobulaPulseMonitor(config, stopChan)
	}()

	<-sigChan
	fmt.Println("\n\n🛑 Shutting down Mobula Pulse monitor...")
	close(stopChan)

	wg.Wait()
	fmt.Println("✓ Monitor stopped")
}
