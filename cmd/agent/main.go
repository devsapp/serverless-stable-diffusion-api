package main

import (
	"flag"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/datastore"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/server"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	defaultPort       = "8010"
	defaultDBType     = datastore.SQLite
	shutdownTimeout   = 5 * time.Second // 5s
	defaultConfigPath = "config.json"
)

func handleSignal() {
	// Wait for interrupt signal to gracefully shutdown the server with
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")
}

func main() {
	port := flag.String("port", defaultPort, "server listen port, default 8010")
	dbType := flag.String("dbType", string(defaultDBType), "db type default sqlite")
	configFile := flag.String("config", defaultConfigPath, "default config path")
	flag.Parse()

	// init config
	if err := config.InitConfig(*configFile); err != nil {
		log.Fatal(err.Error())
	}

	// init server and start
	agent, err := server.NewAgentServer(*port, datastore.DatastoreType(*dbType))
	if err != nil {
		log.Fatal("agent server init fail")
	}
	go agent.Start()

	// wait shutdown signal
	handleSignal()

	if err := agent.Close(shutdownTimeout); err != nil {
		log.Fatal("Shutdown server fail")
	}

	log.Println("Server exiting")
}
