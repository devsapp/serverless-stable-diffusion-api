package config

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
	SessionExpire int
	ModelsNasPath string // sd models dir nas path

	// agent
	CAPort              int32
	CPU                 float32
	Timeout             int32
	InstanceType        string
	InstanceConcurrency int32
	MemorySize          int32
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
		Bucket:          "sd-api-test",
		ModelsNasPath:   "./nas/models",
		OtsEndpoint:     "https://sd-api-test.cn-beijing.ots.aliyuncs.com",
		OtsInstanceName: "sd-api-test",
		OtsMaxVersion:   1,
		OtsTimeToAlive:  -1,
	}
}

func InitConfig(fn string) {
	ConfigGlobal = &Config{}
}
