package settings

import (
	"os"
	"encoding/json"
	"io/ioutil"
)

type Host struct {
	Protocol string
	Host string
	Port int
}

type Threads struct {
	Count int
	Delay int
	Iteration int
}

type SourceProvider struct {
	FileName string
	Delimiter string
	RestartOnEOF bool
}

type Request struct {
	Uri string
	Method string
	Headers []string
	Source SourceProvider
}

type SuccessLevel struct {
	Timeout int
	Codes []int
}

type WarningLevel struct {
	Timeout int
	Codes []int
}

type ErrorLevel struct {
	Timeout int
	Codes []int
}

type Levels struct {
	Success *SuccessLevel
	Warning *WarningLevel
	Error *ErrorLevel
}

type Settings struct {
	Remote Host
	Threads Threads
	Request Request
	Levels Levels
}

func (this *Settings) Load(fileName string) error {
	file, e := ioutil.ReadFile(fileName); if e != nil {
		return e
	}
	e = json.Unmarshal(file, this); if e != nil {
		return e
	}
	return nil
}

func (this *Settings) Save(fileName string) error {
	data, e := json.Marshal(this); if e != nil {
		return e
	}
	e = ioutil.WriteFile(fileName, data, os.ModePerm); if e != nil {
		return e
	}
	return nil
}

