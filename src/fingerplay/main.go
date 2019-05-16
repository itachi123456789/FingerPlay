package main

import (
	"flag"
	"gamemania/libs/signal"
	"runtime"
	"time"

	log "code.google.com/p/log4go"
)

var (
	Debug = true
)

func main() {

	flag.Parse()

	if err := InitConfig(); err != nil {
		panic(err)
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	if err := Init(); err != nil {
		panic(err)
	}

	signal.Block(quit, reload)

	time.Sleep(100 * time.Millisecond)

}

func quit() {
	// TODO
	log.Debug("Get quit signal")
	log.Close()
}

func reload() {
	// TODO
	log.Debug("Get reload signal")
}
