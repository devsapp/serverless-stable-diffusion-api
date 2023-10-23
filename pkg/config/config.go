package config

import (
	"errors"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

var ConfigGlobal *Config

type ConfigYaml struct {
	// ots
	OtsEndpoint     string `yaml:"otsEndpoint"`
	OtsTimeToAlive  int    `yaml:"otsTimeToAlive"`
	OtsInstanceName string `yaml:"otsInstanceName"`
	OtsMaxVersion   int    `yaml:"otsMaxVersion"`
	// oss
	OssEndpoint string `yaml:"ossEndpoint"`
	Bucket      string `yaml:"bucket"`
	OssPath     string `yaml:"ossPath""`
	OssMode     string `yaml:"ossMode"`

	// db
	DbSqlite string `yaml:"dbSqlite"`

	// listen
	ListenInterval int32 `yaml:"listenInterval"`

	// function
	Image               string  `yaml:"image"`
	CAPort              int32   `yaml:"caPort"`
	Timeout             int32   `yaml:"timeout"`
	CPU                 float32 `yaml:"CPU"`
	GpuMemorySize       int32   `yaml:"gpuMemorySize"`
	ExtraArgs           string  `yaml:"extraArgs"`
	DiskSize            int32   `yaml:"diskSize"`
	MemorySize          int32   `yaml:"memorySize"`
	InstanceConcurrency int32   `yaml:"instanceConcurrency"`
	InstanceType        string  `yaml:"instanceType"`

	// user
	SessionExpire             int64  `yaml:"sessionExpire"`
	LoginSwitch               string `yaml:"loginSwitch"`
	ProgressImageOutputSwitch string `yaml:"progressImageOutputSwitch"`

	// sd
	SdUrlPrefix string `yaml:"sdUrlPrefix"`
	SdPath      string `yaml:"sdPath"`
	SdShell     string `yaml:"sdShell"`

	// model
	UseLocalModels string `yaml:"useLocalModel"`

	// flex mode
	FlexMode string `yaml:"flexMode"`

	// agent expose to user or not
	Expose string `yaml:"exposeToUser"`

	// proxy or control or agent
	ServerName string `yaml:"serverName"`
	Downstream string `yaml:"downstream"`
}

type ConfigEnv struct {
	// account
	AccountId            string
	AccessKeyId          string
	AccessKeySecret      string
	AccessKeyToken       string
	Region               string
	ServiceName          string
	ColdStartConcurrency int32
	ModelColdStartSerial bool
}

type Config struct {
	ConfigYaml
	ConfigEnv
}

// IsServerTypeMatch Identify whether serverName == name
func (c *Config) IsServerTypeMatch(name string) bool {
	if c.GetFlexMode() == SingleFunc {
		return true
	}
	return c.ServerName == name
}

func (c *Config) ExposeToUser() bool {
	return c.Expose == "yes"
}

// GetFlexMode flex mode
func (c *Config) GetFlexMode() FlexMode {
	if c.FlexMode == "singleFunc" {
		return SingleFunc
	}
	return MultiFunc
}

func (c *Config) UseLocalModel() bool {
	return c.UseLocalModels == "yes"
}
func (c *Config) EnableLogin() bool {
	return c.LoginSwitch == "on"
}

func (c *Config) GetSDPort() string {
	items := strings.Split(c.SdUrlPrefix, ":")
	if len(items) == 3 {
		return items[2]
	}
	return ""
}

func (c *Config) EnableProgressImg() bool {
	return c.ProgressImageOutputSwitch == "on"
}

func (c *Config) updateFromEnv() {
	// ots
	otsEndpoint := os.Getenv(OTS_ENDPOINT)
	otsInstanceName := os.Getenv(OTS_INSTANCE)
	if otsEndpoint != "" && otsInstanceName != "" {
		c.OtsEndpoint = otsEndpoint
		c.OtsInstanceName = otsInstanceName
	}
	// oss
	ossEndpoint := os.Getenv(OSS_ENDPOINT)
	bucket := os.Getenv(OSS_BUCKET)
	if path := os.Getenv(OSS_PATH); path != "" {
		c.OssPath = path
	}
	if mode := os.Getenv(OSS_MODE); mode != "" {
		c.OssMode = mode
	}
	if ossEndpoint != "" && bucket != "" {
		c.OssEndpoint = ossEndpoint
		c.Bucket = bucket
	}

	// login
	loginSwitch := os.Getenv(LOGINSWITCH)
	if loginSwitch != "" {
		c.LoginSwitch = loginSwitch
	}

	// use local model or not
	useLocalModel := os.Getenv(USER_LOCAL_MODEL)
	if useLocalModel != "" {
		c.UseLocalModels = useLocalModel
	}

	// sd image cover
	sdImage := os.Getenv(SD_IMAGE)
	if sdImage != "" {
		c.Image = sdImage
	}
	gpuMemorySize := os.Getenv(GPU_MEMORY_SIZE)
	if gpuMemorySize != "" {
		if size, err := strconv.Atoi(gpuMemorySize); err == nil {
			c.GpuMemorySize = int32(size)
		}
	}

	// flex mode
	flexMode := os.Getenv(FLEX_MODE)
	if flexMode != "" {
		c.FlexMode = flexMode
	}

	// agent expose to user
	exposeToUser := os.Getenv(EXPOSE_TO_USER)
	if exposeToUser != "" {
		c.Expose = exposeToUser
	}

	// proxy
	serverName := os.Getenv(SERVER_NAME)
	downstream := os.Getenv(DOWNSTREAM)
	if serverName != "" {
		c.ServerName = serverName
	}
	if downstream != "" {
		c.Downstream = downstream
	}

	coldStartConcurrency := os.Getenv(COLD_START_CONCURRENCY)
	c.ColdStartConcurrency = ColdStartConcurrency
	if coldStartConcurrency != "" {
		if concurrency, err := strconv.Atoi(coldStartConcurrency); err == nil {
			c.ColdStartConcurrency = int32(concurrency)
		}
	}
	modelColdStartSerial := os.Getenv(MODEL_COLD_START_SERIAL)
	c.ModelColdStartSerial = ModelColdStartSerial
	if modelColdStartSerial != "" {
		if serial, err := strconv.ParseBool(modelColdStartSerial); err == nil {
			c.ModelColdStartSerial = serial
		}
	}
}

func InitConfig(fn string) error {
	yamlFile, err := ioutil.ReadFile(fn)
	if err != nil {
		return err
	}
	configYaml := new(ConfigYaml)
	err = yaml.Unmarshal(yamlFile, &configYaml)
	if err != nil {
		return err
	}
	configEnv := new(ConfigEnv)
	configEnv.AccountId = os.Getenv(ACCOUNT_ID)
	configEnv.AccessKeyId = os.Getenv(ACCESS_KEY_ID)
	configEnv.AccessKeySecret = os.Getenv(ACCESS_KEY_SECRET)
	configEnv.AccessKeyToken = os.Getenv(ACCESS_KET_TOKEN)
	configEnv.Region = os.Getenv(REGION)
	configEnv.ServiceName = os.Getenv(SERVICE_NAME)
	// check valid
	for _, val := range []string{configEnv.AccountId, configEnv.AccessKeyId,
		configEnv.AccessKeySecret, configEnv.Region, configEnv.ServiceName} {
		if val == "" {
			return errors.New("env not set ACCOUNT_ID || ACCESS_KEY_Id || " +
				"ACCESS_KEY_SECRET || REGION || SERVICE_NAME, please check")
		}
	}
	ConfigGlobal = &Config{
		*configYaml,
		*configEnv,
	}
	// env cover yaml
	ConfigGlobal.updateFromEnv()
	if ConfigGlobal.GetFlexMode() == MultiFunc && ConfigGlobal.ServerName == PROXY && ConfigGlobal.Downstream == "" {
		return errors.New("proxy need set downstream")
	}
	return nil
}
