package server

import (
	"context"
	"errors"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/datastore"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/handler"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/module"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/utils"
	"github.com/gin-gonic/gin"
	"log"
	"net"
	"net/http"
	"time"
)

const SD_START_TIMEOUT = 10 * 60 * 1000 // 10min

type AgentServer struct {
	srv             *http.Server
	listenTask      *module.ListenDbTask
	taskDataStore   datastore.Datastore
	modelDataStore  datastore.Datastore
	configDataStore datastore.Datastore
}

func NewAgentServer(port string, dbType datastore.DatastoreType) (*AgentServer, error) {
	// init router
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	if config.ConfigGlobal.ExposeToUser() {
		// enable ReverserProxy
		router.Any("/*path", handler.ReverseProxy)
		return &AgentServer{
			srv: &http.Server{
				Addr:    net.JoinHostPort("0.0.0.0", port),
				Handler: router,
			},
		}, nil
	} else {
		// only api
		// init oss manager
		if err := module.NewOssManager(); err != nil {
			log.Fatal("oss init fail")
		}
		tableFactory := datastore.DatastoreFactory{}
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
				log.Fatal("sd config update fail")
			}
		}

		handler.RegisterHandlers(router, agentHandler)

		// make sure sd started
		if !utils.PortCheck(config.ConfigGlobal.GetSDPort(), SD_START_TIMEOUT) {
			log.Fatal("sd not start after 10min")
			return nil, errors.New("sd not start after 10min")
		}

		return &AgentServer{
			srv: &http.Server{
				Addr:    net.JoinHostPort("0.0.0.0", port),
				Handler: router,
			},
			listenTask:      listenTask,
			taskDataStore:   taskDataStore,
			modelDataStore:  modelDataStore,
			configDataStore: configDataStore,
		}, nil
	}
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
	// shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := p.srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
		return err
	}
	return nil
}
