package module

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/datastore"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/log"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/utils"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	SD_CONFIG         = "config.json"
	SD_START_TIMEOUT  = 2 * 60 * 1000 // 2min
	SD_DETECT_TIMEOUT = 1000          // 1s
	SD_DETECT_RETEY   = 4             // detect 4 fail
)

type SDManager struct {
	pid     int
	port    string
	flag    bool
	stdout  io.ReadCloser
	endChan chan struct{}
}

func NewSDManager(port string) *SDManager {
	sd := new(SDManager)
	sd.port = port
	sd.endChan = make(chan struct{}, 1)
	if err := sd.init(); err != nil {
		logrus.Error(err.Error())
	}
	return sd
}

func (s *SDManager) getEnv() []string {
	env := make([]string, 0)
	fileMgrToken := ""
	fileMgrEndpoint := ""
	fileMgrName := "admin"
	if adminEnv := FuncManagerGlobal.GetFcFuncEnv(fileMgrName); adminEnv != nil {
		if token := (*adminEnv)["TOKEN"]; token != nil {
			fileMgrToken = *token
		}
		fileMgrEndpoint = fmt.Sprintf("http://%s.%s.%s.%s.fc.devsapp.net", fileMgrName,
			config.ConfigGlobal.ServiceName, config.ConfigGlobal.AccountId, config.ConfigGlobal.Region)
	}
	env = append(env,
		fmt.Sprintf("SERVERLESS_SD_FILEMGR_TOKEN=%s", fileMgrToken),
		fmt.Sprintf("SERVERLESS_SD_FILEMGR_DOMAIN=%s", fileMgrEndpoint))

	// not set DISABLE_HF_CHECK, default proxy enable
	if !config.ConfigGlobal.GetDisableHealthCheck() {
		env = append(env,
			"HTTP_PROXY=http://127.0.0.1:1080",
			"HTTPS_PROXY=http://127.0.0.1:1080",
		)
	}
	return env
}

func (s *SDManager) init() error {
	sdStartTs := utils.TimestampMS()
	// start sd
	execItem, err := utils.DoExecAsync(config.ConfigGlobal.SdShell, config.ConfigGlobal.SdPath, s.getEnv())
	if err != nil {
		return err
	}
	// init read sd log
	go func() {
		stdout := bufio.NewScanner(execItem.Stdout)
		defer execItem.Stdout.Close()
		for stdout.Scan() {
			select {
			case <-s.endChan:
				break
			default:
				log.SDLogInstance.LogFlow <- stdout.Text()
			}
		}

	}()
	s.pid = execItem.Pid
	s.stdout = execItem.Stdout
	// make sure sd started
	if !utils.PortCheck(s.port, SD_START_TIMEOUT) {
		return errors.New("sd not start after 2min")
	}
	s.flag = true
	// start detect
	go s.detectAlive()
	sdEndTs := utils.TimestampMS()
	log.SDLogInstance.TraceFlow <- []string{config.TrackerKeyStableDiffusionStartup,
		fmt.Sprintf("sd start cost=%d", sdEndTs-sdStartTs)}
	return nil
}

func (s *SDManager) detectAlive() {
	retry := SD_DETECT_RETEY
	for s.flag {
		time.Sleep(time.Duration(SD_DETECT_TIMEOUT) * time.Millisecond)
		if !utils.PortCheck(s.port, SD_DETECT_TIMEOUT) && !checkSdExist(strconv.Itoa(s.pid)) {
			retry--
		} else {
			retry = SD_DETECT_RETEY
		}
		if retry <= 0 {
			s.endChan <- struct{}{}
			logrus.Info("restart sd ......")
			s.init()
			return
		}
	}
}

func (s *SDManager) Close() {
	s.flag = false
	syscall.Kill(-s.pid, syscall.SIGKILL)
	s.endChan <- struct{}{}
}

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

func checkSdExist(pid string) bool {
	execItem := utils.DoExec("ps -ef|grep webui|grep -v agent|grep -v grep|awk '{print $2}'", "", nil)
	if strings.Trim(execItem.Output, "\n") == pid {
		return true
	}
	return false
}
