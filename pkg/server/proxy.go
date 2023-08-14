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
	srv *http.Server
}

func NewProxyServer(port string, dbType datastore.DatastoreType) (*ProxyServer, error) {

	//// init task table
	//taskDataStore, err := datastore.NewTaskDataStore(dbType)
	//if err != nil {
	//	log.Println("taskDataStore init fail")
	//	return nil, err
	//}
	//// init model table
	//modelDataStore, err := datastore.NewModelDataStore(dbType)
	//if err != nil {
	//	log.Println("modelDataStore init fail")
	//	return nil, err
	//}
	//// init func manager
	//function.NewFuncManager(dbType)
	// init handler
	proxyHandler := handler.NewProxyHandler()

	// init router
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	handler.RegisterHandlers(router, proxyHandler)

	return &ProxyServer{
		srv: &http.Server{
			Addr:    net.JoinHostPort("0.0.0.0", port),
			Handler: router,
		},
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
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := p.srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
		return err
	}
	return nil
}
