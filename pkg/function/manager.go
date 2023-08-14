package function

import (
	"fmt"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/datastore"
	"sync"
)

var FuncManagerGlobal *FuncManager

type EndPoint struct {
	url  string
	once sync.Once
}

func NewEndPoint(url string) *EndPoint {
	return &EndPoint{
		url: url,
	}
}

func (e *EndPoint) getUrl() string {
	if e.url != "" {
		return e.url
	}
	e.once.Do(e.create)
	return e.url
}

func (e *EndPoint) create() {

}

type FuncManager struct {
	endPoints *sync.Map
	funcStore datastore.FuncInterface
	ak        string
	sk        string
}

func NewFuncManager(dbType datastore.DatastoreType) {
	// init funcDataStore
	funcStore := datastore.NewFuncDataStore(dbType)

	FuncManagerGlobal := &FuncManager{
		endPoints: new(sync.Map),
		funcStore: funcStore,
	}
	// load func from db
	funcAll, _ := funcStore.ListAll([]string{datastore.KFuncKey, datastore.KFuncEndPoint})
	for i, _ := range funcAll {
		FuncManagerGlobal.endPoints.Store(funcAll[i][datastore.KFuncKey], NewEndPoint(funcAll[i][datastore.KFuncEndPoint].(string)))
	}
}

func (f *FuncManager) GetEndPoint(sdModel, sdVae string) (string, error) {
	key := fmt.Sprintf("%s:%s", sdModel, sdVae)
	val, _ := f.endPoints.LoadOrStore(key, new(EndPoint))
	return val.(*EndPoint).getUrl(), nil
}
