package main

import (
	"setbull_trader/cmd/trading/app"
	"setbull_trader/pkg/log"
)

func main() {
	// Initialize enhanced logging system
	logConfig := log.DefaultLogConfig()
	logConfig.LogDir = "logs"
	logConfig.Level = "info"

	// Initialize logger with configuration
	log.InitLoggerWithConfig(logConfig)

	// Log application startup
	log.Info("Setbull Trader application starting", map[string]interface{}{
		"version":   "1.0.0",
		"log_dir":   logConfig.LogDir,
		"log_level": logConfig.Level,
	})

	// Start the application
	app := app.NewApp()
	if err := app.Run(); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	log.Info("Setbull Trader application started successfully")
}
