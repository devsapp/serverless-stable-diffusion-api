package server

import (
	"context"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/datastore"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/handler"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/module"
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
	funcDataStore  datastore.Datastore
	configStore    datastore.Datastore
}

func NewProxyServer(port string, dbType datastore.DatastoreType) (*ProxyServer, error) {
	// init oss manager
	if err := module.NewOssManager(); err != nil {
		return nil, err
	}
	tableFactory := datastore.DatastoreFactory{}
	// init task table
	taskDataStore := tableFactory.NewTable(dbType, datastore.KTaskTableName)
	// init model table
	modelDataStore := tableFactory.NewTable(dbType, datastore.KModelTableName)
	// init user table
	userDataStore := tableFactory.NewTable(dbType, datastore.KUserTableName)
	if err := module.InitUserManager(userDataStore); err != nil {
		return nil, err
	}
	// init config table
	configDataStore := tableFactory.NewTable(dbType, datastore.KConfigTableName)

	var funcDataStore datastore.Datastore
	if config.ConfigGlobal.IsServerTypeMatch(config.CONTROL) {
		// init function table
		funcDataStore = tableFactory.NewTable(dbType, datastore.KModelServiceTableName)
		// init func manager
		if err := module.InitFuncManager(funcDataStore); err != nil {
			return nil, err
		}
		// init listen event
		listenTask := module.NewListenDbTask(config.ConfigGlobal.ListenInterval, taskDataStore, modelDataStore,
			configDataStore)
		// add config listen task
		listenTask.AddTask("configTask", module.ConfigListen, module.ConfigEvent)
	}
	// init handler
	proxyHandler := handler.NewProxyHandler(taskDataStore, modelDataStore, userDataStore, configDataStore)

	// init router
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// auth permission check
	if config.ConfigGlobal.EnableLogin() {
		router.Use(handler.ApiAuth())
	}
	handler.RegisterHandlers(router, proxyHandler)

	return &ProxyServer{
		srv: &http.Server{
			Addr:    net.JoinHostPort("0.0.0.0", port),
			Handler: router,
		},
		taskDataStore:  taskDataStore,
		userDataStore:  userDataStore,
		modelDataStore: modelDataStore,
		funcDataStore:  funcDataStore,
		configStore:    configDataStore,
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
	if p.userDataStore != nil {
		p.userDataStore.Close()
	}
	if p.taskDataStore != nil {
		p.taskDataStore.Close()
	}
	if p.modelDataStore != nil {
		p.modelDataStore.Close()
	}
	if p.funcDataStore != nil {
		p.funcDataStore.Close()
	}
	if p.configStore != nil {
		p.configStore.Close()
	}
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := p.srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
		return err
	}
	return nil
}
