package module

import (
	"fmt"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/sirupsen/logrus"
	"net/http"
)

var client = &http.Client{}

// ModelChangeEvent  models change callback func
func ModelChangeEvent(v any) {
	modelType := v.(string)
	path := ""
	method := "GET"
	switch modelType {
	case config.SD_MODEL:
		path = config.REFRESH_SD_MODEL
		method = "POST"
	case config.CONTORLNET_MODEL:
		path = config.REFRESH_CONTROLNET
		method = "GET"
	case config.SD_VAE:
		path = config.REFRESH_VAE
		method = "POST"
	default:
		logrus.Infof("[ModelChangeEvent] modelType=%s no need reload", modelType)
		return
	}
	url := fmt.Sprintf("%s%s", config.ConfigGlobal.SdUrlPrefix, path)
	req, _ := http.NewRequest(method, url, nil)
	_, err := client.Do(req)
	if err != nil {
		logrus.Info("[ModelChangeEvent] listen model refresh do fail")
		return
	}
}

// CancelEvent tasks cancel signal callback
func CancelEvent(v any) {
	path := config.CANCEL
	url := fmt.Sprintf("%s%s", config.ConfigGlobal.SdUrlPrefix, path)
	req, _ := http.NewRequest("POST", url, nil)
	_, err := client.Do(req)
	if err != nil {
		logrus.Info("[CancelEvent] listen cancel do fail")
	}
	logrus.Info("[CancelEvent] listen cancel signal, url=", url)
}

// ConfigEvent config.json
func ConfigEvent(v any) {
	// update all function env
	retry := 2
	for retry > 0 {
		if err := FuncManagerGlobal.UpdateAllFunctionEnv(); err == nil {
			return
		}
		retry--
	}
}
