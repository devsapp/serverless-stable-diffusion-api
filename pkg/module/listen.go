package module

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/datastore"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/utils"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type CallBack func(v any)

type ListenType int32

const (
	CancelListen ListenType = iota
	ModelListen
	ConfigListen
)

const (
	ConfigDefaultKey = "default"
)

// config change signal
type configSignal struct {
	configStore datastore.Datastore
	md5         string
	config      string
}

type TaskItem struct {
	listenType ListenType
	callBack   CallBack
	curVal     any
}

// ListenDbTask listen db value change and call callback func
// for example: tasks cancel signal and models register/update
type ListenDbTask struct {
	taskStore      datastore.Datastore
	modelStore     datastore.Datastore
	configStore    datastore.Datastore
	intervalSecond int32
	tasks          *sync.Map
	stop           chan struct{}
}

func NewListenDbTask(intervalSecond int32, taskStore datastore.Datastore,
	modelStore datastore.Datastore, configStore datastore.Datastore) *ListenDbTask {
	listenTask := &ListenDbTask{
		taskStore:      taskStore,
		modelStore:     modelStore,
		configStore:    configStore,
		intervalSecond: intervalSecond,
		tasks:          new(sync.Map),
		stop:           make(chan struct{}),
	}
	go listenTask.init()
	return listenTask
}

// init listen
func (l *ListenDbTask) init() {
	for {
		select {
		case <-l.stop:
			break
		default:
			// go on next
		}
		l.tasks.Range(func(key, value any) bool {
			taskId := key.(string)
			taskItem := value.(*TaskItem)
			switch taskItem.listenType {
			case CancelListen:
				l.cancelTask(taskId, taskItem)
			case ModelListen:
				l.modelTask(taskItem)
			case ConfigListen:
				l.configTask(taskItem)
			}
			return true
		})
		time.Sleep(time.Duration(l.intervalSecond) * time.Second)
	}
}

// listen config
func (l *ListenDbTask) configTask(item *TaskItem) {
	confSignal := item.curVal.(*configSignal)
	md5Old := confSignal.md5
	configOld := confSignal.config
	// read config from disk
	configPath := fmt.Sprintf("%s/%s", config.ConfigGlobal.SdPath, SD_CONFIG)
	md5New := ""
	if !utils.FileExists(configPath) {
		log.Printf("[configTask] file %s not exist", configPath)
	} else {
		md5New, _ = utils.FileMD5(configPath)
	}
	// diff
	if md5New != "" && md5New != md5Old {
		configNew, _ := ioutil.ReadFile(configPath)
		// diff config content
		if configOld == "" || !isSameConfig([]byte(configOld), configNew) {
			log.Println("[configTask] update config")
			// update success
			if err := updateConfig(configNew, md5New, l.configStore); err == nil {
				confSignal.md5 = md5New
				confSignal.config = string(configNew)
				// change, call callback and update db
				item.callBack(nil)
			}
		}
	}
}

// listen model task
func (l *ListenDbTask) modelTask(item *TaskItem) {
	// controlNet
	oldVal := *(item.curVal.(*map[string]struct{}))
	path := fmt.Sprintf("%s/models/%s", config.ConfigGlobal.SdPath, "ControlNet")
	curVal := listModelFile(path)
	add, del := utils.DiffSet(oldVal, curVal)
	if len(add) != 0 || len(del) != 0 {
		log.Printf("[modelTask] controlnet model change add: %s, del: %s",
			strings.Join(add, ","), strings.Join(del, ","))
		item.curVal = &curVal
		item.callBack(config.CONTORLNET_MODEL)
	}
	// vae
	if oldVal, err := getVaeFromSD(); err == nil {
		path := fmt.Sprintf("%s/models/%s", config.ConfigGlobal.SdPath, "VAE")
		curVal := listModelFile(path)
		add, del := utils.DiffSet(oldVal, curVal)
		if len(add) != 0 || len(del) != 0 {
			log.Printf("[modelTask] vae model change add: %s, del: %s",
				strings.Join(add, ","), strings.Join(del, ","))
			item.callBack(config.SD_VAE)
		}
	}
	// checkpoint
	if oldVal, err := getCheckPointFromSD(); err == nil {
		path := fmt.Sprintf("%s/models/%s", config.ConfigGlobal.SdPath, "Stable-diffusion")
		curVal := listModelFile(path)
		add, del := utils.DiffSet(oldVal, curVal)
		if len(add) != 0 || len(del) != 0 {
			log.Printf("[modelTask] Stable-diffusion model change add: %s, del: %s",
				strings.Join(add, ","), strings.Join(del, ","))
			item.callBack(config.SD_MODEL)
		}
	}
}

// listen task cancel
func (l *ListenDbTask) cancelTask(taskId string, item *TaskItem) {
	ret, err := l.taskStore.Get(taskId, []string{datastore.KTaskCancel, datastore.KTaskStatus})
	if err != nil {
		l.tasks.Delete(taskId)
		return
	}
	// check task finish delete db listen task
	status := ret[datastore.KTaskStatus].(string)
	if status == config.TASK_FINISH {
		l.tasks.Delete(taskId)
		return
	}
	// cancel val == 1
	cancelVal := ret[datastore.KTaskCancel].(int64)
	if cancelVal == int64(config.CANCEL_VALID) {
		item.callBack(nil)
		l.tasks.Delete(taskId)
		return
	}

}

// AddTask add listen task
func (l *ListenDbTask) AddTask(key string, listenType ListenType, callBack CallBack) {
	var curVal interface{}
	if listenType == ModelListen {
		// controlNet load and other type model not need
		path := fmt.Sprintf("%s/models/%s", config.ConfigGlobal.SdPath, "ControlNet")
		datas := listModelFile(path)
		val := make(map[string]struct{})
		for data := range datas {
			modelName := data
			val[modelName] = struct{}{}
		}
		curVal = &val
	} else if listenType == ConfigListen {
		// read db
		data, err := l.configStore.Get(ConfigDefaultKey, []string{datastore.KConfigMd5, datastore.KConfigVal})
		if err != nil {
			log.Fatal("[AddTask] read config db error:", err.Error())
		}
		md5Old := ""
		configOld := ""
		if data != nil && len(data) > 0 {
			md5Old = data[datastore.KConfigMd5].(string)
			configOld = data[datastore.KConfigVal].(string)
		}
		md5New := ""
		configPath := fmt.Sprintf("%s/%s", config.ConfigGlobal.SdPath, SD_CONFIG)
		if !utils.FileExists(configPath) {
			log.Printf("file %s not exist", configPath)
		} else {
			md5New, _ = utils.FileMD5(configPath)
		}
		// diff
		if md5New != "" && md5New != md5Old {
			configNew, _ := ioutil.ReadFile(configPath)
			if configOld == "" {
				if err := putConfig(configNew, md5New, l.configStore); err == nil {
					log.Println("[AddTask] put config")
					md5Old = md5New
					configOld = string(configNew)
				}
			} else if !isSameConfig([]byte(configOld), configNew) {
				// diff config content
				log.Println("[AddTask] update config")
				// update success
				if err := updateConfig(configNew, md5New, l.configStore); err == nil {
					md5Old = md5New
					configOld = string(configNew)
				}
			}
		}

		curVal = &configSignal{
			configStore: l.configStore,
			md5:         md5Old,
			config:      configOld,
		}

	}
	l.tasks.Store(key, &TaskItem{
		listenType: listenType,
		callBack:   callBack,
		curVal:     curVal,
	})
}

// Close listen
func (l *ListenDbTask) Close() {
	l.stop <- struct{}{}
}

func listModelFile(path string) map[string]struct{} {
	files := utils.ListFile(path)
	ret := make(map[string]struct{})
	for _, name := range files {
		if strings.HasSuffix(name, ".pt") || strings.HasSuffix(name, ".ckpt") ||
			strings.HasSuffix(name, ".safetensors") {
			ret[name] = struct{}{}
		}
	}
	return ret
}

func updateConfig(data []byte, md5 string, configStore datastore.Datastore) error {
	// check data valid
	if !json.Valid(data) {
		log.Printf("[updateConfig] config json not valid, please check")
		return errors.New("config not valid json")
	}
	if err := configStore.Update(ConfigDefaultKey, map[string]interface{}{
		datastore.KConfigMd5:        md5,
		datastore.KConfigVal:        string(data),
		datastore.KConfigModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
	}); err != nil {
		log.Printf("[updateConfig] update db error")
		return err
	}
	return nil
}

func putConfig(data []byte, md5 string, configStore datastore.Datastore) error {
	// check data valid
	if !json.Valid(data) {
		log.Printf("[putConfig] config json not valid, please check")
		return errors.New("config not valid json")
	}
	if err := configStore.Put(ConfigDefaultKey, map[string]interface{}{
		datastore.KConfigMd5:        md5,
		datastore.KConfigVal:        string(data),
		datastore.KConfigModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
		datastore.KConfigCreateTime: fmt.Sprintf("%d", utils.TimestampS()),
	}); err != nil {
		log.Printf("[putConfig] put db error")
		return err
	}
	return nil
}

func getVaeFromSD() (map[string]struct{}, error) {
	url := fmt.Sprintf("%s%s", config.ConfigGlobal.SdUrlPrefix, config.GET_SD_VAE)
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	var result []map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	ret := make(map[string]struct{})
	for _, one := range result {
		k := one["model_name"]
		ret[k] = struct{}{}
	}
	return ret, nil
}

func getCheckPointFromSD() (map[string]struct{}, error) {
	url := fmt.Sprintf("%s%s", config.ConfigGlobal.SdUrlPrefix, config.GET_SD_MODEL)
	req, err := http.NewRequest("GET", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	var result []map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	ret := make(map[string]struct{})
	for _, one := range result {
		k := one["title"]
		modelSlice := strings.Split(k, " ")
		ret[modelSlice[0]] = struct{}{}
	}
	return ret, nil
}

// check config old and new same
func isSameConfig(old, new []byte) bool {
	notCheckField := map[string]struct{}{
		"sd_checkpoint_hash":   {},
		"sd_model_checkpoint":  {},
		"sd_vae":               {},
		"upscaler_for_img2img": {},
	}
	var oldMap map[string]interface{}
	if err := json.Unmarshal(old, &oldMap); err != nil {
		log.Println(err.Error())
		return true
	}
	var newMap map[string]interface{}
	if err := json.Unmarshal(new, &newMap); err != nil {
		log.Println(err.Error())
		return true
	}
	for key, val := range newMap {
		if _, ok := notCheckField[key]; ok {
			continue
		}
		if oldVal, ok := oldMap[key]; !ok {
			return false
		} else if !utils.IsSame(key, val, oldVal) {
			return false
		}
	}
	return true
}
