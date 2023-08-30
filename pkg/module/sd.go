package module

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"io/ioutil"
	"os"
)

const SD_CONFIG = "config.json"

// UpdateSdConfig modify sd config.json sd_model_checkpoint and sd_vae
func UpdateSdConfig() error {
	// sdModel/sdVae from env
	sdModel := os.Getenv(config.MODEL_SD)
	sdVae := os.Getenv(config.MODEL_SD_VAE)
	if sdModel == "" || sdVae == "" {
		return errors.New("sd model not set in env")
	}
	// get sd config
	configPath := fmt.Sprintf("%s/%s", config.ConfigGlobal.SdPath, SD_CONFIG)
	fd, err := os.Open(configPath)
	if err != nil {
		return err
	}
	data, _ := ioutil.ReadAll(fd)
	fd.Close()
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	m["sd_model_checkpoint"] = sdModel
	m["sd_vae"] = sdVae
	m["sd_checkpoint_hash"] = ""
	output, err := json.MarshalIndent(m, "", "    ")
	if err != nil {
		return err
	}
	fdOut, err := os.OpenFile(configPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0775)
	defer fdOut.Close()

	fdOut.WriteString(string(output))
	return nil
}
