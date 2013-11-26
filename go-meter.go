package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"
)

var (
	_method         = flag.String("m", "GET", "HTTP Metod")
	_url            = flag.String("u", "http://localhost", "URL")
	_connection     = flag.Int("c", 64, "Connections count")
	_threads        = flag.Int("t", 4, "Threads count")
	_mrq            = flag.Int("mrq", -1, "Max request per second")
	_source         = flag.String("s", "", "POST/PUT Body source file with \"\\n\" delimeter or URLs on GET/DELETE")
	_duration       = flag.Duration("d", time.Duration(30)*time.Second, "Test duration")
	_verbose        = flag.Bool("v", false, "Live stats view")
	_excludeSeconds = flag.Duration("es", time.Duration(0)*time.Second, "Exclude first seconds from stats")
	_help           = flag.Bool("h", false, "Help")
	_cpuprofile     = flag.String("cpuprofile", "", "write cpu profile to file")
)

type RequestStats struct {
	ResponseCode int
	Duration     time.Duration
	NetIn        int64
	NetOut       int64
}

type Config struct {
	Method            string
	Host              string
	Url               *url.URL
	Connections       int
	Threads           int
	MRQ               int
	Verbose           bool
	ExcludeSeconds    time.Duration
	Source            *Source
	Duration          time.Duration
	ConnectionManager *ConnectionManager
	WorkerQuit        chan bool
	WorkerQuited      chan bool
	StatsQuit         chan bool
	StatsQuited       chan bool
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

	if *_cpuprofile != "" {
		f, err := os.Create(*_cpuprofile)
		if err != nil {
			fmt.Printf("Can not start cpu proffile %v\n", err)
			return
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	config := &Config{
		Method:         *_method,
		Host:           URL.Host,
		Url:            URL,
		Connections:    *_connection,
		Threads:        *_threads,
		MRQ:            *_mrq,
		Verbose:        *_verbose,
		ExcludeSeconds: *_excludeSeconds,
		Source:         sourceData,
		Duration:       *_duration,
		WorkerQuit:     make(chan bool, *_threads),
		WorkerQuited:   make(chan bool, *_threads),
		StatsQuit:      make(chan bool, 1),
		StatsQuited:    make(chan bool, 1),
		RequestStats:   make(chan *RequestStats, *_connection*512),
	}

	if strings.Index(config.Host, ":") > -1 {
		h := strings.SplitN(config.Host, ":", 2)
		config.Host = h[0]
	}

	runtime.GOMAXPROCS(*_threads)

	logUrl := config.Url.String()
	if *_method != "POST" && *_method == "PUT" {
		logUrl = config.Url.Host
	}

	if config.MRQ == -1 {
		fmt.Printf("Running test threads: %d, connections: %d in %v %s %s\n", *_threads, config.Connections, config.Duration, config.Method, logUrl)
	} else {
		fmt.Printf("Running test threads: %d, connections: %d, max req/sec: %d, in %v %s %s\n", *_threads, config.Connections, config.MRQ, config.Duration, config.Method, logUrl)
	}

	config.ConnectionManager = NewConnectionManager(config)

	//check any connect
	anyConnected := false
	for i := 0; i < config.Connections && !anyConnected; i++ {
		connection := config.ConnectionManager.conns[i]
		if connection.IsConnected() {
			anyConnected = true
		}
	}
	if !anyConnected {
		fmt.Printf("Can not connect to %s\n", config.Url.Host)
		return
	}

	go StartStatsAggregator(config)

	for i := 0; i < config.Threads; i++ {
		go NewThread(config)
	}

	//Start SIGTERM listen
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	//Wait timers or SIGTERM
	select {
	case <-time.After(config.Duration):
	case <-signalChan:
	}
	for i := 0; i < config.Threads; i++ {
		config.WorkerQuit <- true
	}
	//Wait for threads complete
	for i := 0; i < config.Threads; i++ {
		<-config.WorkerQuited
	}

	//Stop stats aggregator
	config.StatsQuit <- true
	//Close connections
	for i := 0; i < config.Connections; i++ {
		connection := config.ConnectionManager.conns[i]
		if !connection.IsConnected() {
			continue
		}
		connection.conn.Close()
	}
	//Wait stats aggregator complete
	<-config.StatsQuited
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
