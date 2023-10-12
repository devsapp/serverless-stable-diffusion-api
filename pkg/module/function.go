package module

import (
	"encoding/json"
	"errors"
	"fmt"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	fc "github.com/alibabacloud-go/fc-open-20210406/v2/client"
	fcService "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/datastore"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/utils"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

const (
	RETRY_INTERVALMS = time.Duration(10) * time.Millisecond
)

type SdModels struct {
	sdModel  string
	sdVae    string
	endpoint string
}

var FuncManagerGlobal *FuncManager

// FuncManager manager fc function
// create function and http trigger
// update instance env
type FuncManager struct {
	endpoints map[string][]string
	//modelToInfo map[string][]*SdModels
	funcStore          datastore.Datastore
	fcClient           *fc.Client
	lock               sync.RWMutex
	lastInvokeEndpoint string
}

func InitFuncManager(funcStore datastore.Datastore) error {
	// init fc client
	fcEndpoint := fmt.Sprintf("%s.%s.fc.aliyuncs.com", config.ConfigGlobal.AccountId,
		config.ConfigGlobal.Region)
	fcClient, err := fc.NewClient(new(openapi.Config).SetAccessKeyId(config.ConfigGlobal.AccessKeyId).
		SetAccessKeySecret(config.ConfigGlobal.AccessKeySecret).SetSecurityToken(config.ConfigGlobal.AccessKeyToken).
		SetProtocol("HTTP").SetEndpoint(fcEndpoint))
	if err != nil {
		return err
	}
	FuncManagerGlobal = &FuncManager{
		endpoints: make(map[string][]string),
		funcStore: funcStore,
		fcClient:  fcClient,
		//modelToInfo: make(map[string][]*SdModels),
	}
	// load func endpoint to cache
	FuncManagerGlobal.loadFunc()
	return nil
}

// GetLastInvokeEndpoint get last invoke endpoint
func (f *FuncManager) GetLastInvokeEndpoint() string {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.lastInvokeEndpoint
}

// GetEndpoint get endpoint, key=sdModel
// retry and read from db if create function fail
// first get from cache
// second get from db
// third create function and return endpoint
func (f *FuncManager) GetEndpoint(sdModel string) (string, error) {
	key := "default"
	if config.ConfigGlobal.GetFlexMode() == config.MultiFunc {
		key = sdModel
	}
	// retry
	reTry := 2
	for reTry > 0 {
		// first get cache
		if endpoint := f.getEndpointFromCache(key); endpoint != "" {
			f.lastInvokeEndpoint = endpoint
			return endpoint, nil
		}

		f.lock.Lock()
		// second get from db
		if endpoint := f.getEndpointFromDb(key); endpoint != "" {
			f.lastInvokeEndpoint = endpoint
			f.lock.Unlock()
			return endpoint, nil
		}
		// third create function
		if endpoint := f.createFunc(key, sdModel, getEnv(sdModel)); endpoint != "" {
			f.lastInvokeEndpoint = endpoint
			f.lock.Unlock()
			return endpoint, nil
		}
		f.lock.Unlock()
		reTry--
		time.Sleep(RETRY_INTERVALMS)
	}
	return "", errors.New("not get sd endpoint")
}

// UpdateAllFunctionEnv update instance env, restart agent function
func (f *FuncManager) UpdateAllFunctionEnv() error {
	// reload from db
	f.lock.Lock()
	f.loadFunc()
	f.lock.Unlock()
	// update all function env
	for key, item := range f.endpoints {
		if err := f.UpdateFunctionEnv(key, item[1]); err != nil {
			return err
		}
	}
	return nil
}

// UpdateFunctionEnv update instance env
// input modelName and env
func (f *FuncManager) UpdateFunctionEnv(key, modelName string) error {
	// check model->function exist
	if !f.funcExist(key) {
		return nil
	}
	env := getEnv(modelName)
	functionName := getFunctionName(key)
	if _, err := f.fcClient.UpdateFunction(&config.ConfigGlobal.ServiceName, &functionName,
		new(fc.UpdateFunctionRequest).SetGpuMemorySize(config.ConfigGlobal.GpuMemorySize).
			SetEnvironmentVariables(env)); err != nil {
		logrus.Info(err.Error())
		return err
	}
	return nil
}

// UpdateFunctionImage update instance Image
func (f *FuncManager) UpdateFunctionImage(key string) error {
	functionName := getFunctionName(key)
	if _, err := f.fcClient.UpdateFunction(&config.ConfigGlobal.ServiceName, &functionName,
		new(fc.UpdateFunctionRequest).SetGpuMemorySize(config.ConfigGlobal.GpuMemorySize).
			SetCustomContainerConfig(new(fc.CustomContainerConfig).
				SetImage(config.ConfigGlobal.Image))); err != nil {
		return err
	}

	return nil
}

// get endpoint from cache
func (f *FuncManager) getEndpointFromCache(key string) string {
	f.lock.RLock()
	defer f.lock.RUnlock()
	if val, ok := f.endpoints[key]; ok {
		return val[0]
	}
	return ""
}

// get endpoint from db
func (f *FuncManager) getEndpointFromDb(key string) string {
	if data, err := f.funcStore.Get(key, []string{datastore.KModelServiceSdModel,
		datastore.KModelServiceEndPoint}); err == nil && len(data) > 0 {
		// update cache
		f.endpoints[key] = []string{data[datastore.KModelServiceEndPoint].(string),
			data[datastore.KModelServiceSdModel].(string)}
		return data[datastore.KModelServiceEndPoint].(string)
	}
	return ""
}

func (f *FuncManager) createFunc(key, sdModel string, env map[string]*string) string {
	functionName := getFunctionName(key)
	serviceName := config.ConfigGlobal.ServiceName
	if endpoint, err := f.createFCFunction(serviceName, functionName, env); err == nil && endpoint != "" {
		// update cache
		f.endpoints[key] = []string{endpoint, sdModel}
		// put func to db
		f.putFunc(key, functionName, sdModel, endpoint)
		return endpoint
	} else {
		logrus.Info(err.Error())
	}
	return ""
}

// load endpoint from db
func (f *FuncManager) loadFunc() {
	// load func from db
	funcAll, _ := f.funcStore.ListAll([]string{datastore.KModelServiceKey, datastore.KModelServiceEndPoint,
		datastore.KModelServiceSdModel, datastore.KModelServerImage})
	for _, data := range funcAll {
		key := data[datastore.KModelServiceKey].(string)
		image := data[datastore.KModelServerImage].(string)
		if image != config.ConfigGlobal.Image {
			// update function image
			f.UpdateFunctionImage(key)
			// update db
			f.funcStore.Update(key, map[string]interface{}{
				datastore.KModelServerImage: config.ConfigGlobal.Image,
				datastore.KModelModifyTime:  fmt.Sprintf("%d", utils.TimestampS()),
			})
		}
		endpoint := data[datastore.KModelServiceEndPoint].(string)
		// init lastInvokeEndpoint
		if f.lastInvokeEndpoint == "" {
			f.lastInvokeEndpoint = endpoint
		}
		sdModel := data[datastore.KModelServiceSdModel].(string)
		f.endpoints[key] = []string{endpoint, sdModel}
	}
}

// check model->func exist
func (f *FuncManager) funcExist(key string) bool {
	if data, err := f.funcStore.Get(key, []string{datastore.KModelServiceEndPoint}); err != nil ||
		data == nil || len(data) == 0 {
		return false
	} else {
		return true
	}
	return false
}

// write func into db
func (f *FuncManager) putFunc(key, functionName, sdModel, endpoint string) {
	f.funcStore.Put(key, map[string]interface{}{
		datastore.KModelServiceKey:            key,
		datastore.KModelServiceSdModel:        sdModel,
		datastore.KModelServiceFunctionName:   functionName,
		datastore.KModelServiceEndPoint:       endpoint,
		datastore.KModelServerImage:           config.ConfigGlobal.Image,
		datastore.KModelServiceCreateTime:     fmt.Sprintf("%d", utils.TimestampS()),
		datastore.KModelServiceLastModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
	})
}

// create fc function
func (f *FuncManager) createFCFunction(serviceName, functionName string,
	env map[string]*string) (endpoint string, err error) {
	createRequest := getCreateFuncRequest(functionName, env)
	header := &fc.CreateFunctionHeaders{
		XFcAccountId: utils.String(config.ConfigGlobal.AccountId),
	}
	// create function
	if _, err := f.fcClient.CreateFunctionWithOptions(&serviceName, createRequest,
		header, &fcService.RuntimeOptions{}); err != nil {
		return "", err
	}
	// create http triggers
	httpTriggerRequest := getHttpTrigger()
	resp, err := f.fcClient.CreateTrigger(&serviceName, &functionName, httpTriggerRequest)
	if err != nil {
		return "", err
	}
	return *(resp.Body.UrlInternet), nil
}

// get create function request
func getCreateFuncRequest(functionName string, env map[string]*string) *fc.CreateFunctionRequest {
	return &fc.CreateFunctionRequest{
		FunctionName:         utils.String(functionName),
		CaPort:               utils.Int32(config.ConfigGlobal.CAPort),
		Cpu:                  utils.Float32(config.ConfigGlobal.CPU),
		Timeout:              utils.Int32(config.ConfigGlobal.Timeout),
		InstanceType:         utils.String(config.ConfigGlobal.InstanceType),
		Runtime:              utils.String("custom-container"),
		InstanceConcurrency:  utils.Int32(config.ConfigGlobal.InstanceConcurrency),
		MemorySize:           utils.Int32(config.ConfigGlobal.MemorySize),
		DiskSize:             utils.Int32(config.ConfigGlobal.DiskSize),
		Handler:              utils.String("index.handler"),
		GpuMemorySize:        utils.Int32(config.ConfigGlobal.GpuMemorySize),
		EnvironmentVariables: env,
		CustomContainerConfig: &fc.CustomContainerConfig{
			AccelerationType: utils.String("Default"),
			Image:            utils.String(config.ConfigGlobal.Image),
			WebServerMode:    utils.Bool(true),
		},
	}
}

// get trigger request
func getHttpTrigger() *fc.CreateTriggerRequest {
	triggerConfig := make(map[string]interface{})
	triggerConfig["authType"] = config.AUTH_TYPE
	triggerConfig["methods"] = []string{config.HTTP_GET, config.HTTP_POST, config.HTTP_PUT}
	byteConfig, _ := json.Marshal(triggerConfig)
	return &fc.CreateTriggerRequest{
		TriggerName:   utils.String(config.TRIGGER_NAME),
		TriggerType:   utils.String(config.TRIGGER_TYPE),
		TriggerConfig: utils.String(string(byteConfig)),
	}
}

// hash key, avoid generating invalid characters
func getFunctionName(key string) string {
	return fmt.Sprintf("sd_%s", utils.Hash(key))
}

func getEnv(sdModel string) map[string]*string {
	env := map[string]*string{
		config.SD_START_PARAMS:      utils.String(config.ConfigGlobal.ExtraArgs),
		config.MODEL_SD:             utils.String(sdModel),
		config.MODEL_REFRESH_SIGNAL: utils.String(fmt.Sprintf("%d", utils.TimestampS())), // value = now timestamp
		config.OTS_INSTANCE:         utils.String(config.ConfigGlobal.OtsInstanceName),
		config.OTS_ENDPOINT:         utils.String(config.ConfigGlobal.OtsEndpoint),
	}
	if config.ConfigGlobal.OssMode == config.REMOTE {
		env[config.OSS_ENDPOINT] = utils.String(config.ConfigGlobal.OssEndpoint)
		env[config.OSS_BUCKET] = utils.String(config.ConfigGlobal.Bucket)
	}
	return env
}
