package main

import (
	"setbull_trader/cmd/trading/app"
	"setbull_trader/pkg/log"
)

func main() {
	log.InitLogger()
	app := app.NewApp()
	if err := app.Run(); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}
}
