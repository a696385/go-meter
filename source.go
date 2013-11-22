package main

import (
	"bytes"
	"io/ioutil"
	"regexp"
	"sync"
)

type Source struct {
	lock  sync.Mutex
	Data  [][]byte
	Index int
}

func LoadSource(fileName string, delimiter string) (*Source, error) {
	file, e := ioutil.ReadFile(fileName)
	if e != nil {
		return nil, e
	}
	buff := bytes.NewBuffer(file)
	text := buff.String()
	els := regexp.MustCompile(delimiter).Split(text, -1)
	newThis := Source{}
	for _, el := range els {
		if len(el) == 0 {
			continue
		}
		newThis.Data = append(newThis.Data, bytes.NewBufferString(el).Bytes())
	}
	return &newThis, nil
}

func (this *Source) GetNext() *[]byte {
	if len(this.Data) == 1 {
		return &this.Data[0]
	} else if len(this.Data) == 0 {
		return nil
	}
	//Lock index field and inc
	this.lock.Lock()
	defer this.lock.Unlock()

	this.Index++
	if this.Index >= len(this.Data) {
		this.Index = 0
	}
	return &this.Data[this.Index]
}
