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

type ProxyServer struct {
	srv            *http.Server
	taskDataStore  datastore.Datastore
	modelDataStore datastore.Datastore
	userDataStore  datastore.Datastore
}

func NewProxyServer(port string, dbType datastore.DatastoreType) (*ProxyServer, error) {
	tableFactory := datastore.DatastoreFactory{}
	// init task table
	taskDataStore := tableFactory.NewTable(dbType, datastore.KTaskTableName)
	// init model table
	modelDataStore := tableFactory.NewTable(dbType, datastore.KModelTableName)
	// init user table
	userDataStore := tableFactory.NewTable(dbType, datastore.KUserTableName)

	// init handler
	proxyHandler := handler.NewProxyHandler(taskDataStore, modelDataStore, userDataStore)

	// init router
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	handler.RegisterHandlers(router, proxyHandler)

	return &ProxyServer{
		srv: &http.Server{
			Addr:    net.JoinHostPort("0.0.0.0", port),
			Handler: router,
		},
		taskDataStore:  taskDataStore,
		userDataStore:  userDataStore,
		modelDataStore: modelDataStore,
	}, nil
}

// Start proxy server
func (p *ProxyServer) Start() error {
	if err := p.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %s\n", err)
		return err
	}
	return nil
}

// Close shutdown proxy server, timeout=shutdownTimeout
func (p *ProxyServer) Close(shutdownTimeout time.Duration) error {
	p.userDataStore.Close()
	p.taskDataStore.Close()
	p.modelDataStore.Close()
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := p.srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
		return err
	}
	return nil
}
