package main

import (
	"io/ioutil"
	"bytes"
	"regexp"
)

type Data []byte

type Source []Data

func LoadSource(fileName string, delimiter string) (*Source, error) {
	file, e := ioutil.ReadFile(fileName); if e != nil {
		return nil, e
	}
	buff := bytes.NewBuffer(file)
	text := buff.String()
	els := regexp.MustCompile(delimiter).Split(text, -1)
	newThis := Source{}
	for _,el := range els {
		if len(el) == 0 {
			continue
		}
		newThis = append(newThis, bytes.NewBufferString(el).Bytes())
	}
	return &newThis, nil
}


