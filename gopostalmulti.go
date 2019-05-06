package gopostalmulti

import (
	"log"
	"math/rand"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/dc0d/tinykv"
	"github.com/vmihailenco/msgpack"
)

var pool = sync.Pool{
	New: func() interface{} {
		return request{
			resp: make(chan [][2]string, 1),
		}
	},
}

var kv = tinykv.New(time.Minute, nil)

type request struct {
	address string
	resp    chan [][2]string
}

var workers []chan request

var path = "gopostalmultic"

// start a new underlying OS process of "path" to process locations
func startOne() {
	cmd := exec.Command(path)
	writer, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalf("Error getting writer pipe - %v", err)
	}
	reader, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Error getting reader pipe - %v", err)
	}
	err = cmd.Start()
	if err != nil {
		log.Fatalf("Unable to init gopostalmulti - %v", err)
	}
	reqChan := make(chan request)
	workers = append(workers, reqChan)
	go func() {
		decoder := msgpack.NewDecoder(reader)
		defer cmd.Wait()
		for req := range reqChan {
			writer.Write([]byte(req.address))
			writer.Write([]byte{'\n'})
			out := [][2]string{}
			err := decoder.Decode(&out)
			if err != nil {
				log.Fatalf("Error decoding address - %#v", err)
			}
			req.resp <- out
		}
	}()
}

func init() { // serve one thread that is "native" through cgo
	for x := 0; x < runtime.NumCPU(); x++ { // create a libpostal processor for each CPU
		startOne()
	}
}

// Parse will parse addresses and return postal components
// can be called concurrently
func Parse(address string) [][2]string {
	resp, ok := kv.Get(address)
	if ok {
		return resp.([][2]string)
	}
	req := pool.Get().(request)
	defer pool.Put(req)
	req.address = address
	sentIt := false
FindWorker:
	for x := range workers {
		select {
		case workers[x] <- req:
			sentIt = true
			break FindWorker // req was accepted!
		default:
		}
	}
	if !sentIt {
		workers[rand.Intn(len(workers))] <- req
	}
	newresp := <-req.resp
	kv.Put(address, newresp)
	return newresp
}
