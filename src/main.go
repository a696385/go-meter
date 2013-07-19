package main

import (
	"log"
	"os"
	"fmt"
	"flag"
	"runtime/pprof"
)

import (
	"name/away/settings"
)

var ClearCountersSeconds = flag.Int("clr", -1, "Clear counter seconds (do not clear: -1)")

var cpuprofile = flag.String("pprof", "", "write cpu profile to file")
var threadsCount = flag.Int("t", -1, "Threads count")
var configFileName = flag.String("cfg", "./config.json", "Config.json file name")


func main() {

	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	var e error
	setts := settings.Settings{}
	setts.Load(*configFileName)

	if *threadsCount > 0 {
		setts.Threads.Count = *threadsCount
	}

	var source *Source = &Source{}
	if setts.Request.Source.FileName != "" {
		source, e = LoadSource(setts.Request.Source.FileName, setts.Request.Source.Delimiter); if e != nil{
			panic(e)
		}
		fmt.Println("Source data lines: ", len(*source))
	}

	fmt.Println("Thread count: ", setts.Threads.Count)
	iteration := setts.Threads.Iteration
	if iteration < 0 {
		iteration = len(*source)
	}
	fmt.Println("Iteration count by thread: ", iteration)
	iteration++

	c := make(chan *Status, iteration * setts.Threads.Count)
	for i := 0; i < setts.Threads.Count; i++{
		go StartThread(&setts, source, c)
	}
	for i := iteration * setts.Threads.Count; i>0 ; i-- {
		counter(<-c)
	}
	PrintStatus()
	fmt.Println("Completed")
}
