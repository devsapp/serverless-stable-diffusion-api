package server

import (
	"context"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/datastore"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/handler"
	"github.com/gin-gonic/gin"
	"log"
	"net"
	"net/http"
	"time"
)

type AgentServer struct {
	srv *http.Server
}

func NewAgentServer(port string, dbType datastore.DatastoreType) (*AgentServer, error) {

	// init task table
	taskDataStore, err := datastore.NewTaskDataStore(dbType)
	if err != nil {
		log.Fatal("taskDataStore init fail")
		return nil, err
	}
	// init model table
	modelDataStore, err := datastore.NewModelDataStore(dbType)
	if err != nil {
		log.Fatal("modelDataStore init fail")
		return nil, err
	}
	// init handler
	agentHandler := handler.NewAgentHandler(taskDataStore, modelDataStore)

	// init router
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	handler.RegisterHandlers(router, agentHandler)

	return &AgentServer{
		srv: &http.Server{
			Addr:    net.JoinHostPort("0.0.0.0", port),
			Handler: router,
		},
	}, nil
}

// Start proxy server
func (p *AgentServer) Start() error {
	if err := p.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %s\n", err)
		return err
	}
	return nil
}

// Close shutdown proxy server, timeout=shutdownTimeout
func (p *AgentServer) Close(shutdownTimeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := p.srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
		return err
	}
	return nil
}
