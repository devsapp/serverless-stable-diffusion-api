package function

import (
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/datastore"
	"sync"
)

var FuncManagerGlobal *FuncManager

type FuncManager struct {
	endPoints *sync.Map
	funcStore datastore.FuncInterface
	ak        string
	sk        string
}

func NewFuncManager(dbType datastore.DatastoreType) {
	// init funcDataStore
	funcStore := datastore.NewFuncDataStore(dbType)

	FuncManagerGlobal = &FuncManager{
		endPoints: new(sync.Map),
		funcStore: funcStore,
	}
}
