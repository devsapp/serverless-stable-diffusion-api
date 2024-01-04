package concurrency

import (
	"github.com/sirupsen/logrus"
	"sync"
	"sync/atomic"
)

var ConCurrencyGlobal = NewConcurrency()

type Concurrency struct {
	metrics    *sync.Map
	curColdNum *int32
}

func NewConcurrency() *Concurrency {
	var curColdNum int32 = 0
	return &Concurrency{
		metrics:    new(sync.Map),
		curColdNum: &curColdNum,
	}
}

// WaitToValid Avoid excessive cold start concurrency
func (c *Concurrency) WaitToValid(metric string) bool {
	metricItem, _ := c.metrics.LoadOrStore(metric, NewMetric())
	return metricItem.(*Metric).waitToValid(c.curColdNum)
}

func (c *Concurrency) DoneTask(metric, taskId string) {
	if metricItem, ok := c.metrics.Load(metric); ok {
		metricItem.(*Metric).doneTask()
		//logrus.Info(fmt.Sprintf("finish: %V, coldNum:%d", metricItem.(*Metric).window), *c.curColdNum)
		return
	}
	logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorf("done task err: metric %s not exist", metric)
}

func (c *Concurrency) DecColdNum(metric, taskId string) {
	if metricItem, ok := c.metrics.Load(metric); ok {
		metricItem.(*Metric).SetColdFlag(false)
	} else {
		logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorf("decColdNum task err: metric %s not exist", metric)
	}
	atomic.AddInt32(c.curColdNum, -1)
}
