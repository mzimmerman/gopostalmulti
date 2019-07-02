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

// var pool = sync.Pool{
// 	New: func() interface{} {
// 		return request{
// 			resp: make(chan [][2]string, 1),
// 		}
// 	},
// }

// var kv = tinykv.New(10*time.Minute, nil)

type request struct {
	address string
	resp    chan [][2]string
}

type Libpostal struct {
	Path        string
	MaxBackends int
	pool        sync.Pool
	kv          tinykv.KV
	workers     []chan request
}

// var workers []chan request

// var path = "./gopostalmultic"

// start a new underlying OS process of "path" to process locations
func (l *Libpostal) startOne() {
	cmd := exec.Command(l.Path)
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
	l.workers = append(l.workers, reqChan)
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
	// maxBackends := runtime.NumCPU() / 2
	// if maxBackends < 0 {
	// 	maxBackends = 1
	// }
	// 	for x := 0; x < runtime.NumCPU(); x++ { // create a libpostal processor for each CPU
	// 		time.Sleep(100 * time.Millisecond)
	// 		startOne()
	// 	}
}

func (l *Libpostal) Init() {

	if l.MaxBackends < 0 {
		l.MaxBackends = runtime.NumCPU()
	}
	if l.Path == "" {
		l.Path = "./gopostalmultic"
	}
	l.pool = sync.Pool{
		New: func() interface{} {
			return request{
				resp: make(chan [][2]string, 1),
			}
		},
	}
	l.kv = tinykv.New(10*time.Minute, nil)
	l.workers = make([]chan request, 0)

	for x := 0; x < l.MaxBackends; x++ { // create a libpostal processor for each CPU
		// time.Sleep(100 * time.Millisecond)
		l.startOne()
	}

}

// Parse will parse addresses and return postal components
// can be called concurrently
func (l *Libpostal) Parse(address string) [][2]string {
	resp, ok := l.kv.Get(address)
	if ok {
		return resp.([][2]string)
	}
	req := l.pool.Get().(request)
	defer l.pool.Put(req)
	req.address = address
	sentIt := false
FindWorker:
	for x := range l.workers {
		select {
		case l.workers[x] <- req:
			sentIt = true
			break FindWorker // req was accepted!
		default:
		}
	}
	if !sentIt {
		l.workers[rand.Intn(len(l.workers))] <- req
	}
	newresp := <-req.resp
	l.kv.Put(address, newresp)
	return newresp
}
