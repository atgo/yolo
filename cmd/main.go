package main

import (
	"encoding/json"
	"io/ioutil"

	"github.com/andytruong/yolo/pkg"
)

func main() {
	raw, err := ioutil.ReadFile("composer.lock")
	if nil != err {
		panic(err)
	}

	lock := pkg.Lock{}
	if err := json.Unmarshal(raw, &lock); nil != err {
		panic(err)
	}

	if err := lock.Install(); nil != err {
		panic(err)
	}
}
