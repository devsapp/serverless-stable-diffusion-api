package log

import (
	"bytes"
	"fmt"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"net/http"
	"time"
)

var monitor = NewMonitor()

const (
	timeout = 2 * time.Second
)

type Monitor struct {
	client *http.Client
}

func NewMonitor() *Monitor {
	return &Monitor{
		client: &http.Client{Timeout: timeout},
	}
}

func (m *Monitor) Post(body []byte, path string) error {
	url := fmt.Sprintf("%s/%s", config.ConfigGlobal.LogRemoteService, path)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	_, err = m.client.Do(req)
	return err
}
