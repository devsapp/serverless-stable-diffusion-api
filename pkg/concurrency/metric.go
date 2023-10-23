package concurrency

import (
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/utils"
	"sync"
	"sync/atomic"
	"time"
)

const (
	windowExpired = 3 * 60 // 5min
	windowLength  = 10240
	period        = 2 // 1s
	Retry         = 150
)

type Point struct {
	time int64
	val  int32
}

type Metric struct {
	lock        sync.Mutex
	window      []*Point
	coldFlag    atomic.Bool
	concurrency *int32
}

func NewMetric() *Metric {
	var initConcurrency int32 = 0
	coldFlag := atomic.Bool{}
	coldFlag.Store(false)
	return &Metric{
		window:      make([]*Point, 0, windowLength),
		concurrency: &initConcurrency,
		coldFlag:    coldFlag,
	}
}

// DoneTask
// update window and concurrency
func (m *Metric) doneTask() {
	m.lock.Lock()
	defer m.lock.Unlock()
	len := len(m.window)
	curPoint := &Point{utils.TimestampS(), *m.concurrency}
	if len == 0 || m.window[len-1].val > *m.concurrency {
		m.window = append(m.window, curPoint)
	} else {
		idx := m.findLeftNearestConcurrency(*m.concurrency)
		m.window[idx] = curPoint
		m.window = m.window[:idx+1]
	}
	atomic.AddInt32(m.concurrency, -1)
}

// WaitToValid
// judge request valid, if invalid wait for valid
// update curColdNum and conCurrency
func (m *Metric) waitToValid(curColdNum *int32) bool {
	//logrus.Infof("start: %V", m.window)
	retry := 0
	for retry < Retry {
		retry--
		m.lock.Lock()
		isCold := false
		threshold := utils.TimestampS() - windowExpired
		if len(m.window) == 0 || m.window[len(m.window)-1].time < threshold {
			m.window = make([]*Point, 0, windowLength)
			isCold = true
		} else {
			idx := m.findLeftNearestTime(threshold)
			preMaxConcurrency := m.window[idx].val
			m.window = m.window[idx:]
			if *m.concurrency >= preMaxConcurrency {
				isCold = true
			}
		}
		m.lock.Unlock()

		if !isCold {
			atomic.AddInt32(m.concurrency, 1)
			return false
		} else {
			if atomic.AddInt32(curColdNum, 1) <= config.ConfigGlobal.ColdStartConcurrency &&
				(!config.ConfigGlobal.ModelColdStartSerial ||
					(config.ConfigGlobal.ModelColdStartSerial && !m.coldFlag.Swap(true))) {
				atomic.AddInt32(m.concurrency, 1)
				return true
			} else {
				atomic.AddInt32(curColdNum, -1)
			}
		}
		// sleep period
		time.Sleep(time.Duration(period) * time.Second)
	}
	return false
}

func (m *Metric) SetColdFlag(flag bool) {
	if config.ConfigGlobal.ModelColdStartSerial {
		m.coldFlag.Store(flag)
	}
}

func (m *Metric) findLeftNearestTime(val int64) int {
	low := 0
	high := len(m.window) - 1
	for low <= high {
		mid := (low + high) / 2
		if m.window[mid].time < val {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}
	return low
}

func (m *Metric) findLeftNearestConcurrency(val int32) int {
	low := 0
	high := len(m.window) - 1
	for low <= high {
		mid := (low + high) / 2
		if m.window[mid].val <= val {
			high = mid - 1
		} else {
			low = mid + 1
		}
	}
	return low
}
