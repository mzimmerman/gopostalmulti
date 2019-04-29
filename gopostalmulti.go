package gopostalmulti

import (
	"encoding/gob"
	"log"
	"os/exec"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dc0d/tinykv"

	postal "github.com/openvenues/gopostal/parser"
)

var pool = sync.Pool{
	New: func() interface{} {
		return request{
			resp: make(chan []postal.ParsedComponent, 1),
		}
	},
}

var kv = tinykv.New(time.Minute, nil)

var requestChan = make(chan request, 100)

type request struct {
	address string
	resp    chan []postal.ParsedComponent
}

var count = uint32(0)

var path = "gopostalmulticmd"

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

	go func() {
		encoder := gob.NewEncoder(writer)
		decoder := gob.NewDecoder(reader)
		for {
			select {
			case req := <-requestChan:
				encoder.Encode(req.address)
				out := []postal.ParsedComponent{}
				err := decoder.Decode(&out)
				if err != nil {
					log.Fatalf("Error decoding address result - %v", err)
				}
				req.resp <- out
			case <-time.After(time.Minute):
				atomic.AddUint32(&count, ^uint32(0))
				writer.Close()
				reader.Close()
				cmd.Process.Kill()
				return
			}
		}
	}()
}

func init() { // serve one thread that is "native" through cgo
	go func() {
		for req := range requestChan {
			resp := postal.ParseAddress(req.address)
			req.resp <- resp
		}
	}()
}

// Parse will parse addresses and return postal components
// can be called concurrently
func Parse(address string) []postal.ParsedComponent {
	resp, ok := kv.Get(address)
	if ok {
		return resp.([]postal.ParsedComponent)
	}
	req := pool.Get().(request)
	defer pool.Put(req)
	req.address = address
	start := time.Now()
	requestChan <- req
	newresp := <-req.resp
	kv.Put(address, newresp)
	if time.Since(start) > time.Millisecond {
		cnt := atomic.LoadUint32(&count)
		if cnt < uint32(runtime.NumCPU())-2 { // main thread has to produce work and cgo libpostal thread is working too
			cnt = atomic.AddUint32(&count, 1)
			if cnt < uint32(runtime.NumCPU())-2 {
				startOne()
			} else {
				atomic.AddUint32(&count, ^uint32(0))
			}
		}
	}
	return newresp
}