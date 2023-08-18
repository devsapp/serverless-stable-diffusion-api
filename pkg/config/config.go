package config

var ConfigGlobal *Config

type Config struct {
	ServiceName     string
	Image           string
	AccessKeyId     string
	AccessKeySecret string
	OtsEndpoint     string

	//db
	DbSqlite string

	// Session
	SessionExpire int

	// sd
	SdUrlPrefix string

	// output
	ImageOutputDir string
}

func InitConfig(fn string) {
	ConfigGlobal = &Config{}
}
