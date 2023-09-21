package module

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/datastore"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/utils"
	"io/ioutil"
	"os"
)

const SD_CONFIG = "config.json"

// UpdateSdConfig modify sd config.json sd_model_checkpoint and sd_vae
func UpdateSdConfig(configStore datastore.Datastore) error {
	// sdModel/sdVae from env
	sdModel := os.Getenv(config.MODEL_SD)
	if sdModel == "" {
		return errors.New("sd model not set in env")
	}
	var data []byte
	configPath := fmt.Sprintf("%s/%s", config.ConfigGlobal.SdPath, SD_CONFIG)
	// get sd config from remote
	configData, err := configStore.Get(ConfigDefaultKey, []string{datastore.KConfigVal})
	if err == nil && configData != nil && len(configData) > 0 {
		data = []byte(configData[datastore.KConfigVal].(string))
	} else {
		// get sd config from local
		fd, err := os.Open(configPath)
		if err != nil {
			return err
		}
		data, _ = ioutil.ReadAll(fd)
		fd.Close()
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	m["sd_model_checkpoint"] = sdModel
	m["sd_vae"] = "None"
	m["sd_checkpoint_hash"] = ""
	output, err := json.MarshalIndent(m, "", "    ")
	if err != nil {
		return err
	}
	// delete first
	utils.DeleteLocalFile(configPath)
	fdOut, err := os.OpenFile(configPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0775)
	defer fdOut.Close()

	fdOut.WriteString(string(output))
	return nil
}
