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
	"log"
	"sync"
	"time"
)

const (
	RETRY_INTERVALMS = time.Duration(10) * time.Millisecond
)

type SdModels struct {
	sdModel string
	sdVae   string
}

var FuncManagerGlobal *FuncManager

// FuncManager manager fc function
// create function and http trigger
// update instance env
type FuncManager struct {
	endpoints   map[string]string
	modelToInfo map[string][]*SdModels
	funcStore   datastore.Datastore
	fcClient    *fc.Client
	lock        sync.RWMutex
}

func InitFuncManager(funcStore datastore.Datastore) error {
	// init fc client
	fcEndpoint := fmt.Sprintf("%s.%s.fc.aliyuncs.com", config.ConfigGlobal.AccountId,
		config.ConfigGlobal.Region)
	fcClient, err := fc.NewClient(new(openapi.Config).SetAccessKeyId(config.ConfigGlobal.AccessKeyId).
		SetAccessKeySecret(config.ConfigGlobal.AccessKeySecret).SetProtocol("HTTP").SetEndpoint(fcEndpoint))
	if err != nil {
		return err
	}
	FuncManagerGlobal = &FuncManager{
		endpoints:   make(map[string]string),
		funcStore:   funcStore,
		fcClient:    fcClient,
		modelToInfo: make(map[string][]*SdModels),
	}
	// load func endpoint to cache
	FuncManagerGlobal.loadFunc()
	return nil
}

// GetEndpoint get endpoint, key={sdModel_sdVae}
// retry and read from db if create function fail
// first get from cache
// second get from db
// third create function and return endpoint
func (f *FuncManager) GetEndpoint(sdModel, sdVae string) (string, error) {
	key := getKey(sdModel, sdVae)
	// retry
	reTry := 2
	for reTry > 0 {
		// first get cache
		if endpoint := f.getEndpointFromCache(key); endpoint != "" {
			return endpoint, nil
		}

		f.lock.Lock()
		// second get from db
		if endpoint := f.getEndpointFromDb(key); endpoint != "" {
			f.lock.Unlock()
			return endpoint, nil
		}
		// third create function
		if endpoint := f.createFunc(sdModel, sdVae, getEnv(sdModel, sdVae)); endpoint != "" {
			f.lock.Unlock()
			return endpoint, nil
		}
		f.lock.Unlock()
		reTry--
		time.Sleep(RETRY_INTERVALMS)
	}
	return "", errors.New("not get sd endpoint")
}

// UpdateFunctionEnv update instance env
// input modelName and env
func (f *FuncManager) UpdateFunctionEnv(modelName string) error {
	infos := f.getModelInfo(modelName)
	if infos == nil {
		return nil
	}
	for _, info := range infos {
		env := getEnv(info.sdModel, info.sdVae)
		key := getKey(info.sdModel, info.sdVae)
		functionName := getFunctionName(key)
		if _, err := f.fcClient.UpdateFunction(&config.ConfigGlobal.ServiceName, &functionName,
			new(fc.UpdateFunctionRequest).SetGpuMemorySize(config.ConfigGlobal.GpuMemorySize).
				SetEnvironmentVariables(env)); err != nil {
			return err
		}
	}
	return nil
}

// get endpoint from cache
func (f *FuncManager) getEndpointFromCache(key string) string {
	f.lock.RLock()
	defer f.lock.RUnlock()
	if val, ok := f.endpoints[key]; ok {
		return val
	}
	return ""
}

// get endpoint from db
func (f *FuncManager) getEndpointFromDb(key string) string {
	if data, err := f.funcStore.Get(key, []string{datastore.KModelServiceEndPoint}); err == nil && len(data) == 1 {
		// update cache
		f.endpoints[key] = data[datastore.KModelServiceEndPoint].(string)
		return data[datastore.KModelServiceEndPoint].(string)
	}
	return ""
}

func (f *FuncManager) createFunc(sdModel, sdVae string, env map[string]*string) string {
	key := getKey(sdModel, sdVae)
	functionName := getFunctionName(key)
	serviceName := config.ConfigGlobal.ServiceName
	if endpoint, err := f.createFCFunction(serviceName, functionName, env); err == nil && endpoint != "" {
		// update cache
		f.endpoints[functionName] = endpoint
		// put func to db
		f.putFunc(sdModel, sdVae, endpoint)
		return endpoint
	} else {
		log.Println(err.Error())
	}
	return ""
}

// load endpoint from db
func (f *FuncManager) loadFunc() {
	// clear modelToInfo
	f.modelToInfo = make(map[string][]*SdModels)
	// load func from db
	funcAll, _ := f.funcStore.ListAll([]string{datastore.KModelServiceKey, datastore.KModelServiceEndPoint,
		datastore.KModelServiceSdModel, datastore.KModelServiceSdVae})
	for _, data := range funcAll {
		key := data[datastore.KModelServiceKey].(string)
		endpoint := data[datastore.KModelServiceEndPoint].(string)
		f.endpoints[key] = endpoint
		sdModel := data[datastore.KModelServiceSdModel].(string)
		sdVae := data[datastore.KModelServiceSdVae].(string)
		info := &SdModels{
			sdModel: sdModel,
			sdVae:   sdVae,
		}
		if _, ok := f.modelToInfo[sdModel]; !ok {
			f.modelToInfo[sdModel] = []*SdModels{info}
		} else {
			f.modelToInfo[sdModel] = append(f.modelToInfo[sdModel], info)
		}
		if _, ok := f.modelToInfo[sdVae]; !ok {
			f.modelToInfo[sdVae] = []*SdModels{info}
		} else {
			f.modelToInfo[sdVae] = append(f.modelToInfo[sdVae], info)
		}
	}

}

// input sdModel or sdVae return (sdModel && sdVae)
func (f *FuncManager) getModelInfo(modelName string) []*SdModels {
	// first get from cache
	f.lock.RLock()
	info, ok := f.modelToInfo[modelName]
	f.lock.RUnlock()
	if ok {
		return info
	}

	// second load db and update cache
	f.lock.Lock()
	f.loadFunc()
	f.lock.Unlock()

	// third get from cache
	f.lock.RLock()
	info, ok = f.modelToInfo[modelName]
	f.lock.RUnlock()
	if ok {
		return info
	}
	return nil
}

// write func into db
func (f *FuncManager) putFunc(sdModel, sdVae, endpoint string) {
	key := getKey(sdModel, sdVae)
	f.funcStore.Put(key, map[string]interface{}{
		datastore.KModelServiceKey:            key,
		datastore.KModelServiceSdModel:        sdModel,
		datastore.KModelServiceSdVae:          sdVae,
		datastore.KModelServiceEndPoint:       endpoint,
		datastore.KModelServiceCreateTime:     fmt.Sprintf("%d", utils.TimestampS()),
		datastore.KModelServiceLastModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
	})
}

// create fc fucntion
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

func getKey(sdModel, sdVae string) string {
	return fmt.Sprintf("%s:%s", sdModel, sdVae)
}

// hash key, avoid generating invalid characters
func getFunctionName(key string) string {
	return fmt.Sprintf("sd_%s", utils.Hash(key))
}

func getEnv(sdModel, sdVae string) map[string]*string {
	return map[string]*string{
		config.SD_START_PARAMS:      utils.String(config.ConfigGlobal.ExtraArgs),
		config.MODEL_SD:             utils.String(sdModel),
		config.MODEL_SD_VAE:         utils.String(sdVae),
		config.MODEL_REFRESH_SIGNAL: utils.String(fmt.Sprintf("%d", utils.TimestampS())), // value = now timestamp
		config.ACCESS_KEY_ID:        utils.String(config.ConfigGlobal.AccessKeyId),
		config.ACCESS_KEY_SECRET:    utils.String(config.ConfigGlobal.AccessKeySecret),
		config.OSS_BUCKET:           utils.String(config.ConfigGlobal.Bucket),
		config.OSS_ENDPOINT:         utils.String(config.ConfigGlobal.OssEndpoint),
		config.OTS_INSTANCE:         utils.String(config.ConfigGlobal.OtsInstanceName),
		config.OTS_ENDPOINT:         utils.String(config.ConfigGlobal.OtsEndpoint),
	}
}
