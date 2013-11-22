package main

import (
	"fmt"
	"io"
	"math"
	"sort"
	"time"
)

//Statistic data
var source StatsSource

//Total connection error
var ConnectionErrors int32 = 0

//Format with space prefix
type SpancesFormat struct {
	data   interface{}
	len    int
	right  bool
	format string
}

func (this SpancesFormat) String() string {
	if this.format == "" {
		this.format = "%v"
	}
	s := fmt.Sprintf(this.format, this.data)
	for {
		if len(s) >= this.len {
			break
		}
		if this.right {
			s += " "
		} else {
			s = " " + s
		}
	}
	return s
}

//Statistic data
type StatsSource struct {
	Readed          int64
	Writed          int64
	Requests        int
	Skiped          int
	Min             time.Duration
	Max             time.Duration
	Sum             int64
	Codes           map[int]int
	DurationPercent map[time.Duration]int
	ReadErrors      int
	WriteErrors     int
	Work            time.Duration
}

//Statistic data for verbose mode
type StatsSourcePerSecond struct {
	Readed   int64
	Writed   int64
	Requests int
	Skiped   int
	Sum      int64
}

//Stat aggregator
func StartStatsAggregator(config *Config) {
	allowStore := true
	allowStoreTime := time.After(config.ExcludeSeconds)
	if config.ExcludeSeconds.Seconds() > 0 {
		allowStore = false
	}

	verboseTimer := time.NewTicker(time.Duration(1) * time.Second)
	if config.Verbose {
		fmt.Printf("%s %s %s %s %s %s\n",
			newSpancesFormatRightf("Second", 10, "%s"),
			newSpancesFormatRightf("Total", 10, "%s"),
			newSpancesFormatRightf("Req/sec", 10, "%s"),
			newSpancesFormatRightf("Avg/sec", 10, "%s"),
			newSpancesFormatRightf("In/sec", 10, "%s"),
			newSpancesFormatRightf("Out/sec", 10, "%s"),
		)
	} else {
		verboseTimer.Stop()
	}

	source = StatsSource{
		Codes:           make(map[int]int),
		DurationPercent: make(map[time.Duration]int),
	}

	perSecond := StatsSourcePerSecond{}

	start := time.Now()
	for {
		select {
		//Verbose mode timer
		case <-verboseTimer.C:
			if perSecond.Requests-perSecond.Skiped > 0 && config.Verbose {
				//Get Avg response time
				avgMilliseconds := perSecond.Sum / int64(perSecond.Requests-perSecond.Skiped)
				avg := time.Duration(avgMilliseconds) * time.Millisecond
				//Print stats
				fmt.Printf("%s %s %s %s %s %s\n",
					newSpancesFormatRightf(roundToSecondDuration(time.Now().Sub(start)), 10, "%v"),
					newSpancesFormatRightf(source.Requests, 10, "%d"),
					newSpancesFormatRightf(perSecond.Requests, 10, "%d"),
					newSpancesFormatRightf(avg, 10, "%v"),
					newSpancesFormatRightf(Bites(perSecond.Readed), 10, "%s"),
					newSpancesFormatRightf(Bites(perSecond.Writed), 10, "%s"),
				)
			}
			//Clear data
			perSecond = StatsSourcePerSecond{}
		//Allow store avg data timer
		case <-allowStoreTime:
			allowStore = true
		//Request response
		case res := <-config.RequestStats:
			//Check errors
			if res.ReadError != nil {
				source.ReadErrors++
				continue
			} else if res.WriteError != nil {
				source.WriteErrors++
				continue
			}
			//Add counters
			source.Requests++
			perSecond.Requests++
			perSecond.Readed += res.NetIn
			perSecond.Writed += res.NetOut
			source.Readed += res.NetIn
			source.Writed += res.NetOut
			//Add HTTP code counter
			source.Codes[res.ResponseCode]++
			if !allowStore {
				perSecond.Skiped++
				source.Skiped++
				continue
			}
			//Add sum duration in milliseconds
			sum := int64(res.Duration.Seconds() * 1000)
			source.Sum += sum
			perSecond.Sum += sum

			//Check min/mix request duration
			if source.Min > res.Duration {
				source.Min = roundDuration(res.Duration)
			}
			if source.Max < res.Duration {
				source.Max = roundDuration(res.Duration)
			}
			//Round duration to 10 ms and add to stats
			duration := time.Duration(res.Duration.Nanoseconds()/10000000) * time.Millisecond * 10
			source.DurationPercent[duration]++
		//Exit event
		case <-config.StatsQuit:
			//Strore work time
			source.Work = time.Duration(time.Now().Sub(start).Seconds()*1000) * time.Millisecond
			if config.Verbose {
				s := ""
				for {
					if len(s) >= 61 {
						break
					}
					s += "-"
				}
				fmt.Println(s)
			}
			//Confirm exit
			config.StatsQuit <- true
			return
		}
	}
}

func newSpancesFormat(data interface{}, len int) SpancesFormat {
	return SpancesFormat{data, len, false, "%v"}
}
func newSpancesFormatf(data interface{}, len int, format string) SpancesFormat {
	return SpancesFormat{data, len, false, format}
}

func newSpancesFormatRightf(data interface{}, len int, format string) SpancesFormat {
	return SpancesFormat{data, len, true, format}
}

//Print all statistic
func PrintStats(w io.Writer, config *Config) {
	//Calulate avg request duration
	avg := time.Duration(0)
	if source.Requests-source.Skiped > 0 {
		avgMilliseconds := source.Sum / int64(source.Requests-source.Skiped)
		avg = time.Duration(avgMilliseconds) * time.Millisecond
	}

	//Print latency stats, traffic stats
	fmt.Printf("Stats:      %v %v %v\n", newSpancesFormat("Min", 9), newSpancesFormat("Avg", 9), newSpancesFormat("Max", 9))
	fmt.Printf("  Latency   %v %v %v\n", newSpancesFormat(source.Min, 9), newSpancesFormat(avg, 9), newSpancesFormat(source.Max, 9))
	fmt.Printf("  %d requests in %v", source.Requests, source.Work)
	//Errors
	if source.ReadErrors > 0 || source.WriteErrors > 0 && source.Requests > 0 {
		fmt.Printf(", errors: read %d - %.2f%%, write %d - %.2f%%", source.ReadErrors, getPercent(source.ReadErrors, source.Requests), source.WriteErrors, getPercent(source.WriteErrors, source.Requests))
	}
	//Traffic
	fmt.Printf(", net: in %s, out %s\n", Bytes(source.Readed), Bytes(source.Writed))
	//Connection errors
	if ConnectionErrors > 0 {
		fmt.Printf("  connection errors: %d\n", ConnectionErrors)
	}
	//Print details info
	if source.Requests > 0 {
		//Sort HTTP Code and print
		fmt.Println("HTTP Codes: ")
		keys := make([]int, len(source.Codes))
		i := 0
		for key, _ := range source.Codes {
			keys[i] = key
			i++
		}
		sort.Ints(keys)
		for _, key := range keys {
			value := source.Codes[key]
			fmt.Printf("     %d    %v%%\n", key, newSpancesFormatf(getPercent(value, source.Requests), 9, "%.2f"))
		}

		//Sort latency
		fmt.Println("Latency: ")
		durationKeys := make([]int, len(source.DurationPercent))
		i = 0
		for key, _ := range source.DurationPercent {
			durationKeys[i] = int(key)
			i++
		}
		sort.Ints(durationKeys)

		//Group duration perionds with < 1% of request to one
		var (
			duractionRangesKeys   []string
			duractionRangesValues []int
		)

		var (
			lastValue = -1
			lastKey   time.Duration
			firstKey  time.Duration
		)
		for _, key := range durationKeys {
			duration := time.Duration(key)
			value := source.DurationPercent[duration]
			if lastValue == -1 && (value*100/(source.Requests-source.Skiped)) < 1 {
				//First of group
				lastValue = value
				lastKey = duration
				firstKey = duration
				continue
			} else if lastValue > -1 && (value*100/(source.Requests-source.Skiped)) < 1 {
				//Continue group
				lastValue += value
				lastKey = duration
			} else if lastValue > -1 && (value*100/(source.Requests-source.Skiped)) >= 1 {
				//End of group
				str := firstKey.String()
				if lastKey != firstKey {
					str = fmt.Sprintf("%v - %v", firstKey, lastKey)
				}
				duractionRangesKeys = append(duractionRangesKeys, str)
				duractionRangesValues = append(duractionRangesValues, lastValue)
				lastValue = -1
			} else {
				//Not group
				duractionRangesKeys = append(duractionRangesKeys, duration.String())
				duractionRangesValues = append(duractionRangesValues, value)
			}
		}
		//Has not stored group
		if lastValue > -1 {
			str := firstKey.String()
			if lastKey != firstKey {
				str = fmt.Sprintf("%v - %v", firstKey, lastKey)
			}
			duractionRangesKeys = append(duractionRangesKeys, str)
			duractionRangesValues = append(duractionRangesValues, lastValue)
		}
		//Print latency stats
		maxLen := 0
		for _, key := range duractionRangesKeys {
			if len(key) > maxLen {
				maxLen = len(key)
			}
		}
		for index, key := range duractionRangesKeys {
			value := duractionRangesValues[index]
			fmt.Printf("     %v    %v%%\n", newSpancesFormatf(key, maxLen, "%s"), newSpancesFormatf(getPercent(value, (source.Requests-source.Skiped)), 9, "%.2f"))
		}
	}

	//Print speed stats
	if int(source.Work.Seconds()) > 0 {
		fmt.Printf("Requests: %.2f/sec\n", float64(source.Requests)/source.Work.Seconds())
		fmt.Printf("Net In: %s/sec\n", Bites(source.Readed/int64(source.Work.Seconds())))
		fmt.Printf("Net Out: %s/sec\n", Bites(source.Writed/int64(source.Work.Seconds())))
		fmt.Printf("Transfer: %s/sec\n", Bytes((source.Writed+source.Readed)/int64(source.Work.Seconds())))
	}
}

func logn(n, b float64) float64 { return math.Log(n) / math.Log(b) }

func humanateBytes(s int64, base float64, sizes []string) string {
	if s < 10 {
		return fmt.Sprintf("%dB", s)
	}
	e := math.Floor(logn(float64(s), base))
	suffix := sizes[int(e)]
	val := float64(s) / math.Pow(base, math.Floor(e))
	f := "%.0f"
	if val < 10 {
		f = "%.1f"
	}

	return fmt.Sprintf(f+"%s", val, suffix)
}

func Bytes(s int64) string {
	sizes := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
	return humanateBytes(int64(s), 1000, sizes)
}

func Bites(s int64) string {
	sizes := []string{"Bit", "KBit", "MBit", "GBit", "TBit", "PBit", "EBit"}
	return humanateBytes(int64(s*8), 1000, sizes)
}

func getPercent(c int, max int) float64 {
	return float64(c) * 100 / float64(max)
}

func roundDuration(d time.Duration) time.Duration {
	return time.Duration(RoundFloat(d.Seconds()*1000, 0)) * time.Millisecond
}

func roundToSecondDuration(d time.Duration) time.Duration {
	return time.Duration(RoundFloat(d.Seconds(), 0)) * time.Second
}

func RoundFloat(x float64, prec int) float64 {
	var rounder float64
	pow := math.Pow(10, float64(prec))
	intermed := x * pow

	if intermed < 0.0 {
		intermed -= 0.5
	} else {
		intermed += 0.5
	}
	rounder = float64(int64(intermed))

	return rounder / float64(pow)
}
