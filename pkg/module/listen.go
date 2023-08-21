package module

import (
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/datastore"
	"log"
	"sync"
	"time"
)

type CallBack func(v any)

type ListenType int32

const (
	CancelListen ListenType = iota
	ModelListen
)

// models change signal
type modelChangeSignal struct {
	modelStore datastore.Datastore
	modelName  string
	modelType  string
}

type DbTaskItem struct {
	listenType ListenType
	callBack   CallBack
	curVal     any
}

// ListenDbTask listen db value change and call callback func
// for example: tasks cancel signal and models register/update
type ListenDbTask struct {
	taskStore      datastore.Datastore
	modelStore     datastore.Datastore
	intervalSecond int32
	tasks          *sync.Map
	stop           chan struct{}
}

func NewListenDbTask(intervalSecond int32, taskStore datastore.Datastore,
	modelStore datastore.Datastore) *ListenDbTask {
	listenTask := &ListenDbTask{
		taskStore:      taskStore,
		modelStore:     modelStore,
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
			taskItem := value.(*DbTaskItem)
			switch taskItem.listenType {
			case CancelListen:
				l.cancelTask(taskId, taskItem)
			case ModelListen:
				l.modelTask(taskItem)
			}
			return true
		})
		time.Sleep(time.Duration(l.intervalSecond) * time.Second)
	}
}

// listen model task
func (l *ListenDbTask) modelTask(item *DbTaskItem) {
	curVal := item.curVal.(*map[string]string)
	datas, err := l.modelStore.ListAll([]string{datastore.KModelName, datastore.KModelEtag, datastore.KModelType})
	if err != nil {
		log.Fatal("listen models change fail")
	}
	for _, data := range datas {
		modelName := data[datastore.KModelName].(string)
		modelEtag := data[datastore.KModelEtag].(string)
		modelType := data[datastore.KModelType].(string)
		if val, existed := (*curVal)[modelName]; !existed || val != modelEtag {
			(*curVal)[modelName] = modelEtag
			item.callBack(&modelChangeSignal{l.modelStore, modelName, modelType})
		}
	}
}

// listen task cancel
func (l *ListenDbTask) cancelTask(taskId string, item *DbTaskItem) {
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
	if cancelVal == int64(1) {
		item.callBack(nil)
		l.tasks.Delete(taskId)
		return
	}

}

// AddTask add listen task
func (l *ListenDbTask) AddTask(key string, listenType ListenType, callBack CallBack) {
	curVal := make(map[string]string)
	if listenType == ModelListen {
		// model task need init curVal
		datas, err := l.modelStore.ListAll([]string{datastore.KModelName, datastore.KModelEtag})
		if err != nil {
			log.Fatal("listen models change fail")
		}
		for _, data := range datas {
			modelName := data[datastore.KModelName].(string)
			modelEtag := data[datastore.KModelEtag].(string)
			curVal[modelName] = modelEtag
		}
	}
	l.tasks.Store(key, &DbTaskItem{
		listenType: listenType,
		callBack:   callBack,
		curVal:     &curVal,
	})
}

// Close close listen
func (l *ListenDbTask) Close() {
	l.stop <- struct{}{}
}
