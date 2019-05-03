package gopostalmulti

import (
	"log"
	"os/exec"
	"runtime"
	"sync"
	"time"

	postal "github.com/openvenues/gopostal/parser"
	"github.com/vmihailenco/msgpack"
)

var pool = sync.Pool{
	New: func() interface{} {
		return request{
			resp: make(chan []string, 2),
		}
	},
}

// var kv = tinykv.New(time.Minute, nil)

var requestChan = make(chan request, 100)

type request struct {
	address string
	resp    chan []string
}

var count = 0

var mu sync.Mutex

var path = "gopostalmultic"

// start a new underlying OS process of "path" to process locations
func startOne() {
	mu.Lock()
	defer mu.Unlock()
	if count >= runtime.NumCPU()/2 { // main thread has to produce work and cgo libpostal thread is working too
		return // can't start any more, not helpful
	}
	count++
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
		decoder := msgpack.NewDecoder(reader)
		for {
			select {
			case req := <-requestChan:
				writer.Write([]byte(req.address))
				writer.Write([]byte{'\n'})
				headers := []string{}
				values := []string{}
				err := decoder.Decode(&headers)
				if err != nil {
					log.Fatalf("Error decoding address headers - %#v", err)
				}
				req.resp <- headers
				err = decoder.Decode(&values)
				if err != nil {
					log.Fatalf("Error decoding address values - %#v", err)
				}
				req.resp <- values
			case <-time.After(time.Millisecond * 100):
				writer.Close()
				reader.Close()
				cmd.Process.Kill()
				mu.Lock()
				count--
				mu.Unlock()
				return
			}
		}
	}()
}

func init() { // serve one thread that is "native" through cgo
	go func() {
		for req := range requestChan {
			resp := postal.ParseAddress(req.address)
			headers := []string{}
			values := []string{}
			for x := range resp {
				headers = append(headers, resp[x].Label)
				values = append(values, resp[x].Value)
			}
			req.resp <- headers
			req.resp <- values
		}
	}()
}

// Parse will parse addresses and return postal components
// can be called concurrently
func Parse(address string) [][]string {
	// resp, ok := kv.Get(address)
	// if ok {
	// 	return resp.([][2]string)
	// }
	req := pool.Get().(request)
	defer pool.Put(req)
	req.address = address
	start := time.Now()
	requestChan <- req
	headers := <-req.resp
	values := <-req.resp
	// kv.Put(address, newresp)
	if time.Since(start) > time.Millisecond {
		startOne()
	}
	return [][]string{headers, values}
}
