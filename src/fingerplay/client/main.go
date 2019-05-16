package main

import (
	"math/rand"
	"time"

	log "code.google.com/p/log4go"
)

func main() {
	s := rand.NewSource(123456789)
	r := rand.New(s)
	log.Debug("rand: %d", r.Intn(10000))
	log.Debug("rand: %d", r.Intn(10000))
	log.Debug("rand: %d", r.Intn(10000))
	log.Debug("rand: %d", r.Intn(10000))

	s = rand.NewSource(123456789)
	r = rand.New(s)
	log.Debug("rand: %d", r.Intn(10000))
	log.Debug("rand: %d", r.Intn(10000))
	log.Debug("rand: %d", r.Intn(10000))
	log.Debug("rand: %d", r.Intn(10000))
	time.Sleep(200 * time.Second)
}
