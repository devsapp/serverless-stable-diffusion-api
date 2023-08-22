package module

import (
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFunction(t *testing.T) {
	config.InitConfig("")
	err := InitFuncManager(nil)
	assert.Nil(t, err)
	functionName := "sd-test"
	env := map[string]*string{
		"EXTRA_ARGS": utils.String("--api"),
	}
	endpoint, err := FuncManagerGlobal.createFCFunction(config.ConfigGlobal.ServiceName, functionName, env)
	assert.Nil(t, err)
	assert.NotEqual(t, endpoint, "")
}
