package module

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	fc3 "github.com/alibabacloud-go/fc-20230330/client"
	fc "github.com/alibabacloud-go/fc-open-20210406/v2/client"
	fcService "github.com/alibabacloud-go/tea-utils/v2/service"
	gr "github.com/awesome-fc/golang-runtime"
	"github.com/devsapp/goutils/aigc/project"
	fcUtils "github.com/devsapp/goutils/fc"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/datastore"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/utils"
	"github.com/sirupsen/logrus"
)

const (
	RETRY_INTERVALMS = time.Duration(10) * time.Millisecond
)

type SdModels struct {
	sdModel  string
	sdVae    string
	endpoint string
}

// FuncResource Fc resource
type FuncResource struct {
	Image          string                  `json:"image"`
	CPU            float32                 `json:"cpu"`
	GpuMemorySize  int32                   `json:"gpuMemorySize"`
	InstanceType   string                  `json:"InstanceType"`
	MemorySize     int32                   `json:"memorySize"`
	Timeout        int32                   `json:"timeout"`
	Env            map[string]*string      `json:"env"`
	VpcConfig      *map[string]interface{} `json:"vpcConfig"`
	NasConfig      *map[string]interface{} `json:"nasConfig"`
	OssMountConfig *map[string]interface{} `json:"ossMountConfig"`
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
	fc3Client          *fc3.Client
	lock               sync.RWMutex
	lastInvokeEndpoint string
	prefix             string
}

func isFc3() bool {
	return config.ConfigGlobal.ServiceName == ""
}

func InitFuncManager(funcStore datastore.Datastore) error {
	// init fc client
	fcEndpoint := fmt.Sprintf("%s.%s.fc.aliyuncs.com", config.ConfigGlobal.AccountId,
		config.ConfigGlobal.Region)
	FuncManagerGlobal = &FuncManager{
		endpoints: make(map[string][]string),
		funcStore: funcStore,
	}
	// extra prefix
	if parts := strings.Split(config.ConfigGlobal.FunctionName, project.PrefixDelimiter); len(parts) >= 2 {
		FuncManagerGlobal.prefix = fmt.Sprintf("%s%s", parts[0], project.PrefixDelimiter)
	}
	var err error
	if isFc3() {
		FuncManagerGlobal.fc3Client, err = fc3.NewClient(new(openapi.Config).SetAccessKeyId(config.ConfigGlobal.AccessKeyId).
			SetAccessKeySecret(config.ConfigGlobal.AccessKeySecret).SetSecurityToken(config.ConfigGlobal.AccessKeyToken).
			SetProtocol("HTTP").SetEndpoint(fcEndpoint))
	} else {
		FuncManagerGlobal.fcClient, err = fc.NewClient(new(openapi.Config).SetAccessKeyId(config.ConfigGlobal.AccessKeyId).
			SetAccessKeySecret(config.ConfigGlobal.AccessKeySecret).SetSecurityToken(config.ConfigGlobal.AccessKeyToken).
			SetProtocol("HTTP").SetEndpoint(fcEndpoint))
	}

	if err != nil {
		return err
	}
	if funcStore != nil {
		// load func endpoint to cache
		FuncManagerGlobal.loadFunc()
		//FuncManagerGlobal.checkDbAndFcMatch()
	}
	return nil
}

// check ots table function list match fc function or not
func (f *FuncManager) checkDbAndFcMatch() {
	for sdModel, _ := range f.endpoints {
		functionName := GetFunctionName(sdModel)
		if f.GetFcFunc(functionName) == nil {
			logrus.Errorf("sdModel:%s function in db, not in FC, auto delete ots table fucntion key=%s",
				sdModel, sdModel)
			// function in db not in FC
			f.funcStore.Delete(sdModel)
		}
	}
}

// GetLastInvokeEndpoint get last invoke endpoint
func (f *FuncManager) GetLastInvokeEndpoint(sdModel *string) string {
	f.lock.RLock()
	defer f.lock.RUnlock()
	if sdModel == nil || *sdModel == "" {
		return f.lastInvokeEndpoint
	} else if endpoint := f.getEndpointFromCache(*sdModel); endpoint != "" {
		f.lastInvokeEndpoint = endpoint
		return endpoint
	}
	return f.lastInvokeEndpoint
}

// GetEndpoint get endpoint, key=sdModel
// retry and read from db if create function fail
// first get from cache
// second get from db
// third create function and return endpoint
func (f *FuncManager) GetEndpoint(sdModel string) (string, error) {
	key := "default"
	if config.ConfigGlobal.GetFlexMode() == config.MultiFunc && sdModel != "" {
		key = sdModel
	}
	var err error
	endpoint := ""
	// retry
	reTry := 2
	for reTry > 0 {
		// first get cache
		if endpoint = f.getEndpointFromCache(key); endpoint != "" {
			f.lastInvokeEndpoint = endpoint
			return endpoint, nil
		}

		f.lock.Lock()
		// second get from db
		if endpoint, err = f.getEndpointFromDb(key); endpoint != "" {
			f.lastInvokeEndpoint = endpoint
			f.lock.Unlock()
			return endpoint, nil
		}
		// third create function
		if endpoint, err = f.createFunc(key, sdModel, getEnv(sdModel)); endpoint != "" {
			f.lastInvokeEndpoint = endpoint
			f.lock.Unlock()
			return endpoint, nil
		}
		// four create fail get function
		functionName := GetFunctionName(sdModel)
		if f.GetFcFunc(functionName) != nil {
			if endpoint = GetHttpTrigger(functionName); endpoint != "" {
				f.lastInvokeEndpoint = endpoint
				f.endpoints[key] = []string{endpoint, sdModel}
				logrus.Warnf("function %s sdModel %s in FC not in db, please check。Solution：del %s in FC",
					functionName, sdModel, functionName)
				f.lock.Unlock()
				return endpoint, nil
			}
		}
		f.lock.Unlock()
		reTry--
		time.Sleep(RETRY_INTERVALMS)
	}
	return "", err
}

// UpdateAllFunctionEnv update instance env, restart agent function
func (f *FuncManager) UpdateAllFunctionEnv() error {
	// reload from db
	f.lock.Lock()
	f.loadFunc()
	f.lock.Unlock()
	// update all function env
	for key, _ := range f.endpoints {
		if err := f.UpdateFunctionEnv(key); err != nil {
			return err
		}
	}
	return nil
}

// UpdateFunctionEnv update instance env
// input modelName and env
func (f *FuncManager) UpdateFunctionEnv(key string) error {
	functionName := GetFunctionName(key)
	res := f.GetFuncResource(functionName)
	if res == nil {
		return nil
	}
	res.Env[config.MODEL_REFRESH_SIGNAL] = utils.String(fmt.Sprintf("%d", utils.TimestampS())) // value = now timestamp
	//compatible fc3.0
	if isFc3() {
		if _, err := f.fc3Client.UpdateFunction(&functionName,
			new(fc3.UpdateFunctionRequest).SetRequest(new(fc3.UpdateFunctionInput).SetRuntime("custom-container").
				SetEnvironmentVariables(res.Env).SetGpuConfig(new(fc3.GPUConfig).
				SetGpuMemorySize(res.GpuMemorySize).SetGpuType(res.InstanceType)))); err != nil {
			logrus.Info(err.Error())
			return err
		}
	} else {
		if _, err := f.fcClient.UpdateFunction(&config.ConfigGlobal.ServiceName, &functionName,
			new(fc.UpdateFunctionRequest).SetRuntime("custom-container").SetGpuMemorySize(res.GpuMemorySize).
				SetEnvironmentVariables(res.Env)); err != nil {
			logrus.Info(err.Error())
			return err
		}
	}
	return nil
}

// UpdateFunctionResource update function resource
func (f *FuncManager) UpdateFunctionResource(resources map[string]*FuncResource) ([]string, []string, []string) {
	success := make([]string, 0, len(resources))
	fail := make([]string, 0, len(resources))
	errs := make([]string, 0, len(resources))
	for key, resource := range resources {
		functionName := GetFunctionName(key)
		if isFc3() {
			if _, err := f.fc3Client.UpdateFunction(&functionName, getFC3UpdateFunctionRequest(resource)); err != nil {
				fail = append(fail, functionName)
				errs = append(errs, err.Error())

			} else {
				success = append(success, key)
			}
		} else {
			if _, err := f.fcClient.UpdateFunction(&config.ConfigGlobal.ServiceName, &functionName,
				new(fc.UpdateFunctionRequest).SetRuntime("custom-container").SetGpuMemorySize(resource.GpuMemorySize).
					SetMemorySize(resource.MemorySize).SetCpu(resource.CPU).SetInstanceType(resource.InstanceType).
					SetTimeout(resource.Timeout).SetCustomContainerConfig(new(fc.CustomContainerConfig).
					SetImage(resource.Image)).SetEnvironmentVariables(resource.Env)); err != nil {
				fail = append(fail, functionName)
				errs = append(errs, err.Error())

			} else {
				success = append(success, key)
			}
		}
	}
	return success, fail, errs
}

// DeleteFunction delete function
func (f *FuncManager) DeleteFunction(functions []string) ([]string, []string) {
	if isFc3() {
		return f.delFunctionFC3(functions)
	} else {
		return f.delFunction(functions)
	}
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
func (f *FuncManager) getEndpointFromDb(key string) (string, error) {
	if data, err := f.funcStore.Get(key, []string{datastore.KModelServiceSdModel,
		datastore.KModelServiceEndPoint}); err == nil && len(data) > 0 {
		// update cache
		f.endpoints[key] = []string{data[datastore.KModelServiceEndPoint].(string),
			data[datastore.KModelServiceSdModel].(string)}
		return data[datastore.KModelServiceEndPoint].(string), nil
	} else {
		return "", err
	}
}

func (f *FuncManager) createFunc(key, sdModel string, env map[string]*string) (string, error) {
	functionName := GetFunctionName(key)
	var endpoint string
	var err error
	if isFc3() {
		endpoint, err = f.createFc3Function(functionName, env)
	} else {
		serviceName := config.ConfigGlobal.ServiceName
		endpoint, err = f.createFCFunction(serviceName, functionName, env)
	}
	if err == nil && endpoint != "" {
		// update cache
		f.endpoints[key] = []string{endpoint, sdModel}
		// put func to db
		f.putFunc(key, functionName, sdModel, endpoint)
		return endpoint, nil
	} else {
		logrus.Info(err.Error())
		return "", err
	}
}

// GetFcFuncEnv get fc function env info
func (f *FuncManager) GetFcFuncEnv(functionName string) *map[string]*string {
	if funcBody := f.GetFcFunc(functionName); funcBody != nil {
		switch funcBody.(type) {
		case *fc.GetFunctionResponse:
			return &funcBody.(*fc.GetFunctionResponse).Body.EnvironmentVariables
		case *fc3.GetFunctionResponse:
			return &funcBody.(*fc3.GetFunctionResponse).Body.EnvironmentVariables
		}
	}
	return nil
}

func (f *FuncManager) GetFuncResource(functionName string) *FuncResource {
	if funcBody := f.GetFcFunc(functionName); funcBody != nil {
		switch funcBody.(type) {
		case *fc.GetFunctionResponse:
			info := funcBody.(*fc.GetFunctionResponse)
			return &FuncResource{
				Image:         *info.Body.CustomContainerConfig.Image,
				CPU:           *info.Body.Cpu,
				MemorySize:    *info.Body.MemorySize,
				GpuMemorySize: *info.Body.GpuMemorySize,
				Timeout:       *info.Body.Timeout,
				InstanceType:  *info.Body.InstanceType,
				Env:           info.Body.EnvironmentVariables,
			}
		case *fc3.GetFunctionResponse:
			info := funcBody.(*fc3.GetFunctionResponse)
			return &FuncResource{
				Image:         *info.Body.CustomContainerConfig.Image,
				CPU:           *info.Body.Cpu,
				MemorySize:    *info.Body.MemorySize,
				GpuMemorySize: *info.Body.GpuConfig.GpuMemorySize,
				Timeout:       *info.Body.Timeout,
				InstanceType:  *info.Body.GpuConfig.GpuType,
				Env:           info.Body.EnvironmentVariables,
			}
		}
	}
	return nil
}

// GetFcFunc  get fc function info
func (f *FuncManager) GetFcFunc(functionName string) interface{} {
	if isFc3() {
		if resp, err := f.fc3Client.GetFunction(&functionName, &fc3.GetFunctionRequest{}); err == nil {
			return resp
		}
	} else {
		serviceName := config.ConfigGlobal.ServiceName
		if resp, err := f.fcClient.GetFunction(&serviceName, &functionName, &fc.GetFunctionRequest{}); err == nil {
			return resp
		}
	}
	return nil
}

// load endpoint from db
func (f *FuncManager) loadFunc() {
	// load func from db
	funcAll, _ := f.funcStore.ListAll([]string{datastore.KModelServiceKey, datastore.KModelServiceEndPoint,
		datastore.KModelServiceSdModel, datastore.KModelServerImage})
	for _, data := range funcAll {
		key := data[datastore.KModelServiceKey].(string)
		sdModel := data[datastore.KModelServiceSdModel].(string)
		// check fc && db match
		functionName := GetFunctionName(sdModel)
		if f.GetFcFunc(functionName) == nil {
			logrus.Errorf("functionName:%s, sdModel:%s function in db, not in FC, please delete ots table fucntion "+
				"key=%s", functionName, sdModel, sdModel)
			// function in db not in FC， del ots data
			//f.funcStore.Delete(sdModel)
			continue
		}
		//image := data[datastore.KModelServerImage].(string)
		//if image != "" && config.ConfigGlobal.Image != "" &&
		//	image != config.ConfigGlobal.Image {
		//	// update function image
		//	if err := f.UpdateFunctionImage(key); err != nil {
		//		logrus.Info("update function image err=", err.Error())
		//	}
		//	// update db
		//	f.funcStore.Update(key, map[string]interface{}{
		//		datastore.KModelServerImage: config.ConfigGlobal.Image,
		//		datastore.KModelModifyTime:  fmt.Sprintf("%d", utils.TimestampS()),
		//	})
		//}
		endpoint := data[datastore.KModelServiceEndPoint].(string)
		// init lastInvokeEndpoint
		if f.lastInvokeEndpoint == "" {
			f.lastInvokeEndpoint = endpoint
		}
		f.endpoints[key] = []string{endpoint, sdModel}
	}
}

// write func into db
func (f *FuncManager) putFunc(key, functionName, sdModel, endpoint string) {
	f.funcStore.Put(key, map[string]interface{}{
		datastore.KModelServiceKey:            key,
		datastore.KModelServiceSdModel:        sdModel,
		datastore.KModelServiceFunctionName:   functionName,
		datastore.KModelServiceEndPoint:       endpoint,
		datastore.KModelServiceCreateTime:     fmt.Sprintf("%d", utils.TimestampS()),
		datastore.KModelServiceLastModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
	})
}

func (f *FuncManager) GetSd() *fcUtils.Function {
	return f.ListFunction().StableDiffusion
}

func (f *FuncManager) GetFileMgr() *fcUtils.Function {
	return f.ListFunction().Filemgr
}

func (f *FuncManager) ListFunction() *project.T {
	ctx := &gr.FCContext{
		Credentials: gr.Credentials{
			AccessKeyID:     config.ConfigGlobal.AccessKeyId,
			AccessKeySecret: config.ConfigGlobal.AccessKeySecret,
			SecurityToken:   config.ConfigGlobal.AccessKeyToken,
		},
		Region:    config.ConfigGlobal.Region,
		AccountID: config.ConfigGlobal.AccountId,
		Service: gr.ServiceMeta{
			ServiceName: config.ConfigGlobal.ServiceName,
		},
		Function: gr.FunctionMeta{
			Name: config.ConfigGlobal.FunctionName,
		},
	}
	functions := project.Get(ctx)
	return &functions
}

func GetHttpTrigger(functionName string) string {
	if isFc3() {
		if result, err := FuncManagerGlobal.fc3Client.ListTriggers(&functionName, new(fc3.ListTriggersRequest)); err == nil {
			for _, trigger := range result.Body.Triggers {
				if trigger.HttpTrigger != nil {
					return *trigger.HttpTrigger.UrlIntranet
				}
			}
		}
	} else {
		if result, err := FuncManagerGlobal.fcClient.ListTriggers(&config.ConfigGlobal.ServiceName,
			&functionName, new(fc.ListTriggersRequest)); err == nil {
			for _, trigger := range result.Body.Triggers {
				if trigger.UrlInternet != nil {
					return *trigger.UrlIntranet
				}
			}
		}
	}
	return ""
}

// ---------fc2.0----------
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
	return *resp.Body.UrlIntranet, nil
}

// get create function request
func getCreateFuncRequest(functionName string, env map[string]*string) *fc.CreateFunctionRequest {
	defaultReq := &fc.CreateFunctionRequest{
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
	if sd := FuncManagerGlobal.GetSd(); sd != nil {
		if config.ConfigGlobal.Image == "" {
			defaultReq.CustomContainerConfig.Image = sd.CustomContainerConfig.Image
		}
		defaultReq.CustomContainerConfig.Command = func() *string {
			if sd.CustomContainerConfig.Entrypoint != nil && len(sd.CustomContainerConfig.Entrypoint) > 0 {
				return sd.CustomContainerConfig.Entrypoint[0]
			}
			return nil
		}()
		defaultReq.CustomContainerConfig.Args = func() *string {
			if sd.CustomContainerConfig.Command != nil && len(sd.CustomContainerConfig.Command) > 0 {
				return sd.CustomContainerConfig.Command[0]
			}
			return nil
		}()
		if sd.EnvironmentVariables != nil {
			allEnv := make(map[string]*string)
			for k, v := range sd.EnvironmentVariables {
				allEnv[k] = v
			}
			for k, v := range defaultReq.EnvironmentVariables {
				allEnv[k] = v
			}
			defaultReq.EnvironmentVariables = allEnv
		}
	}
	return defaultReq
}

func (f *FuncManager) delFunction(functionNames []string) (fails []string, errs []string) {
	for _, functionName := range functionNames {
		f.fcClient.DeleteTrigger(&config.ConfigGlobal.ServiceName, &functionName, utils.String(config.TRIGGER_NAME))
		if _, err := f.fcClient.DeleteFunction(&config.ConfigGlobal.ServiceName, &functionName); err != nil {
			logrus.Warnf("%s delete fail, err: %s", functionName, err.Error())
			fails = append(fails, functionName)
			errs = append(errs, err.Error())
		}
	}
	return
}

// get trigger request
func getHttpTrigger() *fc.CreateTriggerRequest {
	triggerConfig := make(map[string]interface{})
	triggerConfig["authType"] = config.AUTH_TYPE
	triggerConfig["methods"] = []string{config.HTTP_GET, config.HTTP_POST, config.HTTP_PUT}
	triggerConfig["disableURLInternet"] = true
	byteConfig, _ := json.Marshal(triggerConfig)
	return &fc.CreateTriggerRequest{
		TriggerName:   utils.String(config.TRIGGER_NAME),
		TriggerType:   utils.String(config.TRIGGER_TYPE),
		TriggerConfig: utils.String(string(byteConfig)),
	}
}

// ------------end fc2.0----------

// --------------fc3.0--------------
func (f *FuncManager) createFc3Function(functionName string,
	env map[string]*string) (endpoint string, err error) {
	createRequest := f.getCreateFuncRequestFc3(functionName, env)
	if createRequest == nil {
		return "", errors.New("get createFunctionRequest error")
	}
	// create function
	if _, err := f.fc3Client.CreateFunction(createRequest); err != nil {
		return "", err
	}
	// create http triggers
	httpTriggerRequest := getHttpTriggerFc3()
	resp, err := f.fc3Client.CreateTrigger(&functionName, httpTriggerRequest)
	if err != nil {
		return "", err
	}

	return *resp.Body.HttpTrigger.UrlIntranet, nil
}

// fc3.0 get create function request
func (f *FuncManager) getCreateFuncRequestFc3(functionName string, env map[string]*string) *fc3.CreateFunctionRequest {
	// get current function
	function := f.GetFcFunc(config.ConfigGlobal.FunctionName)
	if function == nil {
		return nil
	}
	curFunction := function.(*fc3.GetFunctionResponse)
	input := &fc3.CreateFunctionInput{
		FunctionName:         utils.String(functionName),
		Cpu:                  utils.Float32(config.ConfigGlobal.CPU),
		Timeout:              utils.Int32(config.ConfigGlobal.Timeout),
		Runtime:              utils.String("custom-container"),
		InstanceConcurrency:  utils.Int32(config.ConfigGlobal.InstanceConcurrency),
		MemorySize:           utils.Int32(config.ConfigGlobal.MemorySize),
		DiskSize:             utils.Int32(config.ConfigGlobal.DiskSize),
		EnvironmentVariables: env,
		Handler:              utils.String("index.handler"),
		CustomContainerConfig: &fc3.CustomContainerConfig{
			AccelerationType: utils.String("Default"),
			Image:            utils.String(config.ConfigGlobal.Image),
			Port:             utils.Int32(config.ConfigGlobal.CAPort),
		},
		GpuConfig: &fc3.GPUConfig{
			GpuMemorySize: utils.Int32(config.ConfigGlobal.GpuMemorySize),
			GpuType:       utils.String(config.ConfigGlobal.InstanceType),
		},
		Role:           curFunction.Body.Role,
		VpcConfig:      curFunction.Body.VpcConfig,
		NasConfig:      curFunction.Body.NasConfig,
		OssMountConfig: curFunction.Body.OssMountConfig,
	}
	if sd := FuncManagerGlobal.GetSd(); sd != nil {
		if config.ConfigGlobal.Image == "" {
			input.CustomContainerConfig.Image = sd.CustomContainerConfig.Image
		}
		if sd.CustomContainerConfig.Entrypoint != nil && len(sd.CustomContainerConfig.Entrypoint) > 0 {
			input.CustomContainerConfig.Entrypoint = sd.CustomContainerConfig.Entrypoint
		}
		if sd.CustomContainerConfig.Command != nil && len(sd.CustomContainerConfig.Command) > 0 {
			input.CustomContainerConfig.Command = sd.CustomContainerConfig.Command
		}
		if sd.EnvironmentVariables != nil {
			allEnv := make(map[string]*string)
			for k, v := range sd.EnvironmentVariables {
				allEnv[k] = v
			}
			for k, v := range input.EnvironmentVariables {
				allEnv[k] = v
			}
			input.EnvironmentVariables = allEnv
		}
	}
	return &fc3.CreateFunctionRequest{
		Request: input,
	}
}

// delete function
func (f *FuncManager) delFunctionFC3(functionNames []string) (fails []string, errs []string) {
	for _, functionName := range functionNames {
		if _, err := f.fc3Client.DeleteFunction(&functionName); err != nil {
			logrus.Warnf("%s delete fail, err: %s", functionName, err.Error())
			fails = append(fails, functionName)
			errs = append(errs, err.Error())
		}
	}
	return
}

// get trigger request
func getHttpTriggerFc3() *fc3.CreateTriggerRequest {
	triggerConfig := make(map[string]interface{})
	triggerConfig["authType"] = config.AUTH_TYPE
	triggerConfig["methods"] = []string{config.HTTP_GET, config.HTTP_POST, config.HTTP_PUT}
	triggerConfig["disableURLInternet"] = true
	byteConfig, _ := json.Marshal(triggerConfig)
	input := &fc3.CreateTriggerInput{
		TriggerName:   utils.String(config.TRIGGER_NAME),
		TriggerType:   utils.String(config.TRIGGER_TYPE),
		TriggerConfig: utils.String(string(byteConfig)),
	}
	return &fc3.CreateTriggerRequest{
		Request: input,
	}
}

// ----------end fc3-----------

// GetFunctionName hash key, avoid generating invalid characters
func GetFunctionName(key string) string {
	return fmt.Sprintf("%ssd_%s", FuncManagerGlobal.prefix, utils.Hash(key))
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

func getFC3UpdateFunctionRequest(resource *FuncResource) *fc3.UpdateFunctionRequest {
	req := new(fc3.UpdateFunctionInput).SetRuntime("custom-container").
		SetMemorySize(resource.MemorySize).SetCpu(resource.CPU).SetGpuConfig(new(fc3.GPUConfig).
		SetGpuType(resource.InstanceType).SetGpuMemorySize(resource.GpuMemorySize)).
		SetTimeout(resource.Timeout).SetCustomContainerConfig(new(fc3.CustomContainerConfig).
		SetImage(resource.Image)).SetEnvironmentVariables(resource.Env)
	if resource.VpcConfig != nil {
		vpcConfig := &fc3.VPCConfig{}
		if err := utils.MapToStruct(*resource.VpcConfig, vpcConfig); err == nil {
			req.SetVpcConfig(vpcConfig)
		}
	}
	if resource.NasConfig != nil {
		nasConfig := &fc3.NASConfig{}
		if err := utils.MapToStruct(*resource.NasConfig, nasConfig); err == nil {
			req.SetNasConfig(nasConfig)
		}
	}
	if resource.OssMountConfig != nil {
		ossConfig := &fc3.OSSMountConfig{}
		if err := utils.MapToStruct(*resource.OssMountConfig, ossConfig); err == nil {
			req.SetOssMountConfig(ossConfig)
		}
	}
	return new(fc3.UpdateFunctionRequest).SetRequest(req)
}
