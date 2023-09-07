package module

import (
	"fmt"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/datastore"
	"log"
	"net/http"
)

var client = &http.Client{}

// ModelChangeEvent  models change callback func
// only deal lore/controNet models
func ModelChangeEvent(v any) {
	modelInfo := v.(*modelChangeSignal)
	modelType := modelInfo.modelType
	path := ""
	switch modelType {
	case config.LORA_MODEL:
		path = config.REFRESH_LORAS
	case config.CONTORLNET_MODEL:
		path = config.REFRESH_CONTROLNET
	default:
		log.Printf("modelType=%s no need reload", modelType)
		return
	}
	url := fmt.Sprintf("%s%s", config.ConfigGlobal.SdUrlPrefix, path)
	req, _ := http.NewRequest("GET", url, nil)
	_, err := client.Do(req)
	if err != nil {
		log.Println("listen model refresh do fail")
		return
	}
	if !config.ConfigGlobal.UseLocalModel() {
		// update
		if err = modelInfo.modelStore.Update(modelInfo.modelName, map[string]interface{}{
			datastore.KModelStatus: config.MODEL_LOADED}); err != nil {
			log.Println("listen model update status fail err=", err.Error())
		}
	}
	log.Println("listen models signal, model=", modelInfo.modelName)
}

// CancelEvent tasks cancel signal callback
func CancelEvent(v any) {
	path := config.CANCEL
	url := fmt.Sprintf("%s%s", config.ConfigGlobal.SdUrlPrefix, path)
	req, _ := http.NewRequest("POST", url, nil)
	_, err := client.Do(req)
	if err != nil {
		log.Println("listen cancel do fail")
	}
	log.Println("listen cancel signal, url=", url)
}
