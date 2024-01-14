package server

import (
	"context"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/datastore"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/handler"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/log"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/module"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net"
	"net/http"
	"time"
)

type AgentServer struct {
	srv             *http.Server
	listenTask      *module.ListenDbTask
	taskDataStore   datastore.Datastore
	modelDataStore  datastore.Datastore
	configDataStore datastore.Datastore
	sdManager       *module.SDManager
}

func NewAgentServer(port string, dbType datastore.DatastoreType, mode string) (*AgentServer, error) {
	agentServer := new(AgentServer)
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
	tableFactory := datastore.DatastoreFactory{}
	if config.ConfigGlobal.ExposeToUser() {
		// init func manager
		if err := module.InitFuncManager(nil); err != nil {
			return nil, err
		}
		agentServer.sdManager = module.NewSDManager(config.ConfigGlobal.GetSDPort())
		// enable ReverserProxy
		router.Any("/*path", handler.ReverseProxy)
	} else {
		// only api
		// init function table
		funcDataStore := tableFactory.NewTable(dbType, datastore.KModelServiceTableName)
		// init func manager
		if err := module.InitFuncManager(funcDataStore); err != nil {
			return nil, err
		}
		// init oss manager
		if err := module.NewOssManager(); err != nil {
			logrus.Fatal("oss init fail")
		}
		// init task table
		taskDataStore := tableFactory.NewTable(dbType, datastore.KTaskTableName)
		// init model table
		modelDataStore := tableFactory.NewTable(dbType, datastore.KModelTableName)
		// init config table
		configDataStore := tableFactory.NewTable(dbType, datastore.KConfigTableName)
		// init listen event
		listenTask := module.NewListenDbTask(config.ConfigGlobal.ListenInterval, taskDataStore, modelDataStore,
			configDataStore)
		// add listen model change
		listenTask.AddTask("modelTask", module.ModelListen, module.ModelChangeEvent)
		// init handler
		agentHandler := handler.NewAgentHandler(taskDataStore, modelDataStore, configDataStore, listenTask)

		// update sd config.json
		if !config.ConfigGlobal.ExposeToUser() {
			if err := module.UpdateSdConfig(configDataStore); err != nil {
				logrus.Fatal("sd config update fail")
			}
		}
		agentServer.sdManager = module.NewSDManager(config.ConfigGlobal.GetSDPort())

		handler.RegisterHandlers(router, agentHandler)
		router.NoRoute(agentHandler.NoRouterAgentHandler)
		agentServer.listenTask = listenTask
		agentServer.taskDataStore = taskDataStore
		agentServer.modelDataStore = modelDataStore
		agentServer.configDataStore = configDataStore
	}

	agentServer.srv = &http.Server{
		Addr:    net.JoinHostPort("0.0.0.0", port),
		Handler: router,
	}
	return agentServer, nil

}

// Start proxy server
func (p *AgentServer) Start() error {
	if err := p.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logrus.Fatalf("listen: %s\n", err)
		return err
	}
	return nil
}

// Close shutdown proxy server, timeout=shutdownTimeout
func (p *AgentServer) Close(shutdownTimeout time.Duration) error {
	// close listen task
	if p.listenTask != nil {
		p.listenTask.Close()
	}
	if p.taskDataStore != nil {
		p.taskDataStore.Close()
	}
	if p.modelDataStore != nil {
		p.modelDataStore.Close()
	}
	if p.configDataStore != nil {
		p.configDataStore.Close()
	}
	if p.sdManager != nil {
		p.sdManager.Close()
	}

	if log.SDLogInstance != nil {
		log.SDLogInstance.Close()
	}

	if module.ProxyGlobal != nil {
		module.ProxyGlobal.Close()
	}

	// shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := p.srv.Shutdown(ctx); err != nil {
		logrus.Fatal("Server forced to shutdown: ", err)
		return err
	}
	return nil
}
