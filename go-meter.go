package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	_method         = flag.String("m", "GET", "HTTP Metod")
	_url            = flag.String("u", "http://localhost:3001/", "URL")
	_connection     = flag.Int("c", 100, "Connections count")
	_threads        = flag.Int("t", 12, "Threads count")
	_mrq            = flag.Int("mrq", -1, "Max request per second")
	_source         = flag.String("s", "", "POST/PUT Body source file with \"\\n\" delimeter or URLs on GET/DELETE")
	_duration       = flag.Duration("d", time.Duration(30)*time.Second, "Test duration")
	_reconnect      = flag.Bool("reconnect", false, "Reconnect for every request")
	_verbose        = flag.Bool("v", false, "Live stats view")
	_excludeSeconds = flag.Duration("es", time.Duration(0)*time.Second, "Exclude first seconds from stats")
	_help           = flag.Bool("h", false, "Help")
)

type RequestStats struct {
	ResponseCode int
	Duration     time.Duration
	ReadError    error
	WriteError   error
	NetIn        int64
	NetOut       int64
}

type Config struct {
	Method            string
	Url               *url.URL
	Connections       int
	Threads           int
	MRQ               int
	Reconnect         bool
	Verbose           bool
	ExcludeSeconds    time.Duration
	Source            *Source
	Duration          time.Duration
	ConnectionManager *ConnectionManager
	WorkerQuit        chan bool
	WorkerQuited      chan bool
	StatsQuit         chan bool
	RequestStats      chan *RequestStats
}

func main() {
	flag.Parse()

	if *_help {
		flag.Usage()
		return
	}

	var (
		sourceData *Source
		err        error
	)

	*_method = strings.ToUpper(*_method)

	if *_method == "POST" || *_method == "PUT" || (len(*_source) > 0 && FileExists(*_source)) {
		sourceData, err = LoadSource(*_source, "\n")
		if err != nil {
			fmt.Printf("ERROR: Can not load source file %s\n", *_source)
			return
		}
	} else {
		sourceData = &Source{}
	}

	URL, err := url.Parse(*_url)
	if err != nil {
		fmt.Printf("ERROR: URL is broken %s\n", *_url)
		return
	}

	config := &Config{
		Method:         *_method,
		Url:            URL,
		Connections:    *_connection,
		Threads:        *_threads,
		MRQ:            *_mrq,
		Reconnect:      *_reconnect,
		Verbose:        *_verbose,
		ExcludeSeconds: *_excludeSeconds,
		Source:         sourceData,
		Duration:       *_duration,
		WorkerQuit:     make(chan bool, *_threads),
		WorkerQuited:   make(chan bool, *_threads),
		StatsQuit:      make(chan bool, 2),
		RequestStats:   make(chan *RequestStats),
	}

	logUrl := config.Url.String()
	if *_method != "POST" && *_method == "PUT" {
		logUrl = config.Url.Host
	}

	if config.MRQ == -1 {
		fmt.Printf("Running test threads: %d, connections: %d in %v %s %s", config.Threads, config.Connections, config.Duration, config.Method, logUrl)
	} else {
		fmt.Printf("Running test threads: %d, connections: %d, max req/sec: %d, in %v %s %s", config.Threads, config.Connections, config.MRQ, config.Duration, config.Method, logUrl)
	}
	if config.Reconnect {
		fmt.Printf(" with reconnect")
	}
	fmt.Print("\n")

	config.ConnectionManager = NewConnectionManager(config)

	go StartStatsAggregator(config)

	for i := 0; i < config.Threads; i++ {
		go NewThread(config)
	}

	//Start Ctr+C listen
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	//Wait timers or SIGTERM
	endTime := time.After(config.Duration)
	select {
	case <-endTime:
		for i := 0; i < config.Threads; i++ {
			config.WorkerQuit <- true
		}
	case <-signalChan:
		for i := 0; i < config.Threads; i++ {
			config.WorkerQuit <- true
		}
	}
	//Wait for threads complete
	for i := 0; i < config.Threads; i++ {
		<-config.WorkerQuited
	}

	//Stop stats aggregator
	config.StatsQuit <- true
	//Close connections
	for i := 0; i < config.Connections; i++ {
		connection := config.ConnectionManager.Get()
		if !connection.IsConnected() {
			continue
		}
		connection.conn.Close()
	}
	//Wait stats aggregator complete
	<-config.StatsQuit
	//Print result
	PrintStats(os.Stdout, config)
}

func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
