package module

import (
	"crypto/tls"
	"fmt"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	huggingFaceUrl     = "http://huggingface.co/"
	huggingFaceTimeout = 2 * time.Second
)

var (
	httpClient  = http.DefaultClient
	ProxyGlobal = NewProxy()
)

type Proxy struct {
	server          *http.Server
	isHuggingFaceOk bool
	stop            chan struct{}
}

func NewProxy() *Proxy {
	proxy := &Proxy{
		isHuggingFaceOk: true,
		stop:            make(chan struct{}, 1),
	}
	go proxy.init()
	proxy.server = &http.Server{
		Addr:    ":1080",
		Handler: http.HandlerFunc(proxy.handleHTTP),
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	go proxy.server.ListenAndServe()
	return proxy
}

func (p *Proxy) init() {
	// check Hugging Face connectedness
	if os.Getenv(config.DISABLE_HF_CHECK) == "" {
		c := &http.Client{Timeout: huggingFaceTimeout}
		resp, err := c.Get(huggingFaceUrl)

		p.isHuggingFaceOk = err == nil && resp != nil
	}
}

func (p *Proxy) handleHTTP(w http.ResponseWriter, req *http.Request) {
	logrus.Infof("proxy:%V, %s", req, p.isHuggingFaceOk)
	if !p.isHuggingFaceOk {
		switch strings.ToLower(req.URL.Hostname()) {
		case "huggingface.co":
			logrus.Infof("connect to %s (%s), reject it.", req.URL.Hostname(), req.URL.String())
			w.WriteHeader(http.StatusRequestTimeout)
			w.Write([]byte(fmt.Sprintf("can not connect to %s", req.URL.Hostname())))
			return
		}
	}

	if req.Method == http.MethodConnect {
		// https
		dstConn, err := net.DialTimeout("tcp", req.Host, 10*time.Second)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		defer dstConn.Close()

		w.WriteHeader(http.StatusOK)
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
			return
		}

		cliConn, _, err := hijacker.Hijack()
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		}
		defer cliConn.Close()

		wg := new(sync.WaitGroup)
		wg.Add(2)

		go func() {
			defer wg.Done()
			io.Copy(dstConn, cliConn)
		}()

		go func() {
			defer wg.Done()
			io.Copy(cliConn, dstConn)
		}()

		wg.Wait()
	} else {
		// http
		resp, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		defer resp.Body.Close()

		// Copy header
		for k, v := range resp.Header {
			for _, vv := range v {
				w.Header().Add(k, vv)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}

func (p *Proxy) Close() {
	if p.server != nil {
		p.server.Close()
	}
	p.stop <- struct{}{}
}
