package server

import (
	"context"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/datastore"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/handler"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/module"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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

func NewProxyServer(port string, dbType datastore.DatastoreType, mode string) (*ProxyServer, error) {
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
	// init function table
	funcDataStore := tableFactory.NewTable(dbType, datastore.KModelServiceTableName)
	// init func manager
	if err := module.InitFuncManager(funcDataStore); err != nil {
		return nil, err
	}
	if config.ConfigGlobal.IsServerTypeMatch(config.CONTROL) {
		// init listen event
		listenTask := module.NewListenDbTask(config.ConfigGlobal.ListenInterval, taskDataStore, modelDataStore,
			configDataStore)
		// add config listen task
		listenTask.AddTask("configTask", module.ConfigListen, module.ConfigEvent)
	}
	// init handler
	proxyHandler := handler.NewProxyHandler(taskDataStore, modelDataStore, userDataStore,
		configDataStore, funcDataStore)

	// init router
	if mode == gin.DebugMode {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(cors.Default())
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(handler.Stat())

	// auth permission check
	if config.ConfigGlobal.EnableLogin() {
		router.Use(handler.ApiAuth())
	}
	handler.RegisterHandlers(router, proxyHandler)
	router.NoRoute(proxyHandler.NoRouterHandler)

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
		logrus.Fatalf("listen: %s\n", err)
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
		logrus.Fatal("Server forced to shutdown: ", err)
		return err
	}
	return nil
}
