package main

import (
	"fmt"
	"github.com/a696385/go-meter/http"
	"net/url"
	"os"
	"time"
)

func NewThread(config *Config) {
	timerAllow := time.NewTicker(time.Duration(250) * time.Millisecond)
	allow := int32(config.MRQ / 4 / config.Threads)
	if config.MRQ == -1 {
		allow = -1
	} else if allow <= 0 {
		allow = 1
		timerAllow.Stop()
		timerAllow.C = nil
	}
	currentAllow := allow
	for {
		select {
		//Set allow requests per time timer
		case <-timerAllow.C:
			currentAllow = allow
		//Get free tcp connection
		case connection := <-config.ConnectionManager.C:
			if config.MRQ != -1 {
				currentAllow--
			}
			//Return connection to pool if allowed request is 0
			if currentAllow > 0 || config.MRQ == -1 {
				connection.Take()
				//Create request object
				req := getRequest(config.Method, config.Url, config.Host, config.Source.GetNext())
				//Send request if we connected
				go connection.Exec(req, config.RequestStats)
			} else {
				connection.Return()
			}
		//Wait exit event
		case <-config.WorkerQuit:
			//Complete exit
			config.WorkerQuited <- true
			return
		}
	}
}

func getRequest(method string, URL *url.URL, host string, body *[]byte) *http.Request {
	header := map[string][]string{}

	if method == "POST" || method == "PUT" {
		return &http.Request{
			Method:        method,
			URL:           URL,
			Header:        header,
			Body:          *body,
			ContentLength: int64(len(*body)),
			Host:          host,
			Created:       time.Now(),
		}
	} else {
		//Use source data as URL request or original URL
		var (
			r   *url.URL
			err error
		)
		if body != nil {
			r, err = url.Parse(string(*body))
			if err != nil {
				fmt.Printf("ERROR: URL is broken %s\n", string(*body))
				os.Exit(1)
			}
		} else {
			r = URL
		}
		return &http.Request{
			Method:  method,
			URL:     r,
			Header:  header,
			Host:    host,
			Created: time.Now(),
		}
	}
}
