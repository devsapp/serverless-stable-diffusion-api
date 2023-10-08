package config

import (
	"errors"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
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

	// model
	UseLocalModels string `yaml:"useLocalModel"`
}

type ConfigEnv struct {
	// account
	AccountId       string
	AccessKeyId     string
	AccessKeySecret string
	AccessKeyToken  string
	Region          string
	ServiceName     string
}

type Config struct {
	ConfigYaml
	ConfigEnv
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
	return nil
}
