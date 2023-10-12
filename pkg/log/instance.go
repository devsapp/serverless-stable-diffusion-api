package log

import "github.com/sirupsen/logrus"

var SDLogInstance = NewSDLog()

// SDLog sd log instance
type SDLog struct {
	taskId    string
	requestId string
	LogFlow   chan string
	close     chan struct{}
}

func NewSDLog() *SDLog {
	sdLogInstance := &SDLog{
		LogFlow: make(chan string, 10000),
		close:   make(chan struct{}, 1),
	}
	go sdLogInstance.consume()
	return sdLogInstance
}

func (s *SDLog) consume() {
	for {
		select {
		case logStr := <-s.LogFlow:
			if s.taskId != "" {
				logrus.WithFields(logrus.Fields{
					"taskId": s.taskId,
				}).Info(logStr)
			} else if s.requestId != "" {
				logrus.WithFields(logrus.Fields{
					"requestId": s.requestId,
				}).Info(logStr)
			} else {
				logrus.Info(logStr)
			}
		case <-s.close:
			break
		}
	}
}

func (s *SDLog) SetTaskId(taskId string) {
	s.taskId = taskId
}

func (s *SDLog) Close() {
	s.close <- struct{}{}
}
