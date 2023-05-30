package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/safwentrabelsi/tx-json-rpc-server/config"
	"github.com/safwentrabelsi/tx-json-rpc-server/ethclient"
	"github.com/safwentrabelsi/tx-json-rpc-server/rpc"
	log "github.com/sirupsen/logrus"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file: ",err)
	}
	err  = config.LoadConfig()
	if err != nil {
		log.Fatal("Error loading the config: ",err)
	}

}

func main() {
	cfg := config.GetConfig()
	log.SetFormatter(&log.JSONFormatter{})
    log.SetOutput(os.Stdout)

	logLevel, err := log.ParseLevel(cfg.LogLevel())
	if err != nil {
		log.Fatal("Invalid log level in the config: ",err)
	}
	log.SetLevel(logLevel)

	ethclient.Init()

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	go ethclient.Client.MonitorGas(ctx)

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		// Cleanup and exit
		cancel()
		os.Exit(0)
	}()

	// Start server
	err = rpc.StartServer(ethclient.Client)
	if err != nil {
		log.Fatal("Failed to start the JSON RPC server: ",err)
	}
}


