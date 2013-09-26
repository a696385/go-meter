package main

import (
	"log"
	"time"
	"strings"
	"fmt"
	"strconv"
)

type HttpOperation struct {
	OperationStart time.Time
	OperationEnd time.Time
	Size int64
}

var (
	successCount int = 0
	warningCount int = 0
	errorCount int = 0
	totalCount int = 0
	durations []int64 = make([]int64, 0)
	operations []HttpOperation = make([]HttpOperation, 0)
	absoluteCount int = 0
)

var statusPrinted int = -1
var lastPrint time.Time = time.Now()
var lastClear time.Time = time.Now()
var tableHeader []string = []string{"Total count", "Success %", "Warning %", "Error %", "Arg receive", "Arg reciave time", "Transfer"}


func printTable(args... interface {}){
	if len(args) == 0{
		fmt.Println(strings.Join(tableHeader,"    "))
		return
	}
	var toPrint []string = []string{}
	for i,int := range args {
		header := tableHeader[i]
		s := fmt.Sprintf("%v", int)
		for ;len(s)<len(header); {
			s = " " + s
		}
		toPrint = append(toPrint, s)
	}
	fmt.Println(strings.Join(toPrint,"    "))
}

func PrintStatus(){
	if statusPrinted > 10 || statusPrinted == -1 {
		statusPrinted = 0
		printTable()
	}
	var sendInSecondSize  =  map[time.Time] int64 {}
	for _,op := range operations {
		key := op.OperationStart.Round(time.Second)
		sendInSecondSize[key] += op.Size
	}
	var receiveInSecond  =  map[time.Time] int {}
	for _,op := range operations {
		key := op.OperationEnd.Round(time.Second)
		receiveInSecond[key] ++
	}
	len := 0
	count := 0
	for _,v := range receiveInSecond {
		count += v
		len ++
	}
	receivePerSecond := 0
	if len > 0 {
		receivePerSecond = count / len
	}
	len = 0
	var totalSendSize int64 = 0
	for _,v := range sendInSecondSize {
		totalSendSize += v
		len ++
	}

	var SendSizePerSecond int64 = 0
	if len > 0 {
		SendSizePerSecond = totalSendSize / int64(len)
	}

	SendSizePerSecondText := ""


	if SendSizePerSecond > 1024*1024*1024 {
		SendSizePerSecondText = strconv.FormatInt(SendSizePerSecond / 1024 / 1024 / 1024, 10) + " gb"
	} else if SendSizePerSecond > 1024*1024 {
		SendSizePerSecondText = strconv.FormatInt(SendSizePerSecond / 1024 / 1024, 10) + " mb"
	} else if SendSizePerSecond > 1024 {
		SendSizePerSecondText = strconv.FormatInt(SendSizePerSecond / 1024, 10) + " kb"
	} else {
		SendSizePerSecondText = strconv.FormatInt(SendSizePerSecond, 10) + " b"
	}

	var durationSum int64 = 0
	count = 0
	for _, el := range durations {
		durationSum += el
		count++
	}
	if count > 0 {
		durationSum /= int64(count)
	}
	second := time.Duration(durationSum)

	if totalCount == 0 {
		totalCount = 1
	}

	printTable(absoluteCount, successCount * 100 / totalCount, warningCount * 100 / totalCount, errorCount * 100 / totalCount, strconv.Itoa(receivePerSecond) + "/sec", strconv.FormatInt(second.Nanoseconds() / int64(1000000), 10) + "ms", SendSizePerSecondText + "/sec")
	statusPrinted++

	diff := time.Now().Sub(lastClear)
	if diff.Seconds() > float64(*ClearCountersSeconds) && *ClearCountersSeconds > 0 {
		lastClear = time.Now()
		successCount = 0
		warningCount = 0
		errorCount = 0
		totalCount = 0
		durations = make([]int64, 0)
		operations  = make([]HttpOperation, 0)
	}
}


func counter(status *Status){

	if (status.IsSuccess) {
		successCount++
	}
	if (status.IsWarning){
		warningCount++
	}
	if (status.IsError){
		errorCount++
	}
	totalCount++
	absoluteCount++
	if status.Duration != nil {
		durations = append(durations, status.Duration.Nanoseconds())
	}

	if status.StartedAt != nil && status.FinishedAt != nil {
		operations = append(operations, HttpOperation{OperationStart: *status.StartedAt, OperationEnd: *status.FinishedAt, Size: status.Size})
	}

	if status.Error != nil {
		log.Println(*status.Error)
	}
	diff := time.Now().Sub(lastPrint)
	if diff.Seconds() > 1 {
		lastPrint = time.Now()
		PrintStatus()
	}

}

func counterWatcher(c chan *Status){
	for status := range c {
		counter(status)
	}
}
