package log

import (
	"encoding/json"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
	"sync"
)

// send log && trace
const (
	defaultCacheCount = 64
	defaultCacheSize  = 16 * 1024 // 16KB
	logPath           = "collect/log"
	tracePath         = "collect/tracker"
)

var SDLogInstance = NewSDLog()

// Log ...
type Log struct {
	AccountID string `json:"accountID"`
	Level     string `json:"level"`
	Ts        int64  `json:"ts"`
	Msg       string `json:"msg"`
	RequestID string `json:"requestID"`
	Source    string `json:"source"`
}

func (l *Log) Size() int {
	return len(l.Msg)
}

// Tracker ...
type Tracker struct {
	Key       string      `json:"key"`
	AccountID string      `json:"accountID"`
	Ts        int64       `json:"ts"`
	Payload   interface{} `json:"payload"`
	Source    string      `json:"source"`
}

// SDLog sd log instance
type SDLog struct {
	taskId       string
	requestId    sync.Map
	cacheLog     []*Log
	cacheTrace   []*Tracker
	LogFlow      chan string
	TraceFlow    chan []string
	closeLog     chan struct{}
	closeTrace   chan struct{}
	accountId    string
	functionName string
}

func NewSDLog() *SDLog {
	sdLogInstance := &SDLog{
		LogFlow:      make(chan string, 8192),
		TraceFlow:    make(chan []string, 8192),
		cacheLog:     make([]*Log, 0, defaultCacheCount),
		cacheTrace:   make([]*Tracker, 0, defaultCacheCount),
		closeLog:     make(chan struct{}),
		closeTrace:   make(chan struct{}),
		accountId:    os.Getenv(config.FC_ACCOUNT_ID),
		functionName: os.Getenv(config.FC_FUNCTION_NAME),
		requestId:    sync.Map{},
	}
	go sdLogInstance.consumeLog()
	go sdLogInstance.consumeTrace()
	return sdLogInstance
}

func (s *SDLog) getRequestId() string {
	requestId := strings.Builder{}
	s.requestId.Range(func(key, value any) bool {
		req := key.(string)
		if requestId.Len() != 0 {
			requestId.WriteString(",")
		}
		requestId.WriteString(req)
		return true
	})
	return requestId.String()
}

func (s *SDLog) consumeLog() {
	cacheSize := 0
	for {
		select {
		case logStr := <-s.LogFlow:
			if s.taskId != "" {
				logrus.WithFields(logrus.Fields{
					"taskId": s.taskId,
				}).Info(logStr)
			} else if requestId := s.getRequestId(); requestId != "" {
				logrus.WithFields(logrus.Fields{
					"requestId": requestId,
				}).Info(logStr)
				if config.ConfigGlobal.SendLogToRemote() {
					logObj := &Log{
						AccountID: config.ConfigGlobal.AccountId,
						Msg:       logStr,
						RequestID: requestId,
						Source:    config.ConfigGlobal.ServerName,
						Level:     "info",
					}
					if cacheSize >= defaultCacheSize || len(s.cacheLog) >= defaultCacheCount {
						if body, err := json.Marshal(s.cacheLog); err == nil {
							go monitor.Post(body, logPath)
						}
						s.cacheLog = s.cacheLog[:0]
						cacheSize = 0
					}
					s.cacheLog = append(s.cacheLog, logObj)
					cacheSize += logObj.Size()
				}
			} else {
				logrus.Info(logStr)
			}
		case <-s.closeLog:
			return
		}
	}
}

func (s *SDLog) consumeTrace() {
	for {
		select {
		case traceSlice := <-s.TraceFlow:
			if config.ConfigGlobal.SendLogToRemote() {
				traceObj := &Tracker{
					AccountID: config.ConfigGlobal.AccountId,
					Key:       traceSlice[0],
					Ts:        0,
					Payload:   traceSlice[1],
					Source:    config.ConfigGlobal.ServerName,
				}
				if len(s.cacheTrace) >= defaultCacheCount {
					if body, err := json.Marshal(s.cacheTrace); err == nil {
						go monitor.Post(body, tracePath)
					}
					s.cacheTrace = s.cacheTrace[:0]
				}
				s.cacheTrace = append(s.cacheTrace, traceObj)

			}
		case <-s.closeTrace:
			return
		}
	}
}

func (s *SDLog) SetTaskId(taskId string) {
	s.taskId = taskId
}

func (s *SDLog) AddRequestId(requestId string) {
	s.requestId.Store(requestId, struct{}{})
}

func (s *SDLog) DelRequestId(requestId string) {
	s.requestId.Delete(requestId)
}

func (s *SDLog) Close() {
	s.closeLog <- struct{}{}
	s.closeTrace <- struct{}{}
	// send trace
	if len(s.cacheTrace) > 0 {
		if body, err := json.Marshal(s.cacheTrace); err == nil {
			monitor.Post(body, tracePath)
		}
	}
	// send log
	if len(s.cacheLog) > 0 {
		if body, err := json.Marshal(s.cacheLog); err == nil {
			monitor.Post(body, logPath)
		}
	}
}
