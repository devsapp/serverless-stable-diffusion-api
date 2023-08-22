package config

import (
	"errors"
	"os"
)

var ConfigGlobal = DefaultConfig()

type Config struct {
	// account
	AccountId       string
	AccessKeyId     string
	AccessKeySecret string
	Region          string

	// ots
	OtsEndpoint     string
	OtsInstanceName string
	OtsTimeToAlive  int // data expired time/second
	OtsMaxVersion   int // data column max version nums

	// oss
	OssEndpoint string
	Bucket      string

	// function
	ServiceName string
	Image       string

	// db
	DbSqlite string

	// proxy
	// Session
	SessionExpire int64
	ModelsNasPath string // sd models dir nas path

	// agent
	CAPort              int32
	CPU                 float32
	Timeout             int32
	InstanceType        string
	InstanceConcurrency int32
	MemorySize          int32
	DiskSize            int32
	GpuMemorySize       int32
	ExtraArgs           string
	// sd
	SdUrlPrefix string
	// output
	ImageOutputDir string
	ListenInterval int32
}

func DefaultConfig() *Config {
	return &Config{
		DbSqlite:        "./sqlite3",
		SdUrlPrefix:     "http://localhost:7860",
		ImageOutputDir:  "./images/",
		ListenInterval:  1,
		OssEndpoint:     "oss-cn-beijing.aliyuncs.com",
		Bucket:          "sd-api-t",
		AccountId:       os.Getenv(ACCOUNT_ID),
		AccessKeyId:     os.Getenv(ACCESS_KEY_ID),
		AccessKeySecret: os.Getenv(ACCESS_KEY_SECRET),
		ModelsNasPath:   "./nas/models",
		OtsEndpoint:     "https://sd-api-test.cn-beijing.ots.aliyuncs.com",
		OtsInstanceName: "sd-api-t",
		OtsMaxVersion:   1,
		OtsTimeToAlive:  -1,
		Region:          "cn-beijing",
		ServiceName:     "fc-stable-diffusion-api",
		Image:           "registry.cn-beijing.aliyuncs.com/aliyun-fc/fc-stable-diffusion:anime-v2",
	}
}

func InitConfig(fn string) error {
	ConfigGlobal = &Config{}
	ConfigGlobal.AccountId = os.Getenv(ACCOUNT_ID)
	ConfigGlobal.AccessKeyId = os.Getenv(ACCESS_KEY_ID)
	ConfigGlobal.AccessKeySecret = os.Getenv(ACCESS_KEY_SECRET)
	if ConfigGlobal.AccountId == "" || ConfigGlobal.AccessKeyId == "" || ConfigGlobal.AccessKeySecret == "" {
		return errors.New("not set ACCOUNT_ID || ACCESS_KEY_Id || ACCESS_KEY_SECRET, please check")
	}
	return nil
}
