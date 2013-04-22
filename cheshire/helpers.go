package cheshire

import (
	"crypto/rand"
	// "crypto/sha256"
	"encoding/base32"
	// "fmt"
	// "hash"
	"io"
	"strings"
	// "strconv"
	// "time"
	"sync"
)

// Sends an error response to the channel
func SendError(txn *Txn, code int, message string) (int, error) {
	resp := NewError(txn, code, message)
	c, err := txn.Write(resp)
	return c, err
}

// Creates a new response based on this request txn.
// auto fills the txn id
func NewResponse(txn RequestTxnId) *Response {
	response := newResponse()
	response.SetTxnId(txn.TxnId())
	return response
}

func NewError(txn RequestTxnId, code int, message string) *Response {
	response := NewResponse(txn)
	response.SetStatus(code, message)
	return response
}

func RandString(length int) string {
	k := make([]byte, length*2)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return ""
	}
	str := strings.TrimRight(
		base32.StdEncoding.EncodeToString(k), "=")
	return str[:length]
}

// Simple multiplex event handler
type Events struct {
	inchan chan Event
	outchans []chan Event
	lock sync.Mutex
}

type Event struct {
	eventType string
	eventMessage string
}

// Creates a new Events object
func NewEvents() *Events{
	events := &Events{
		inchan := make(chan Event, 10),
		outchans := make([]chan Event, 0),
	}
	go func(events *Events) {
		for {
			e := <- events.inchan
			events.lock.Lock()
			for _, out := range(events.outchans) {
				go func(event Event) {
					select {
						case err := out <- event:
							if err != nil {
								//remove this channel
								event.remove(out)
							}
						default: //the event skips if the channel is unavail. 
					}
				}(e)
			}
			events.lock.Unlock()
		}
	}(events)	
	return events
}

// Emits this message to all the listening channels
func (this *Events) Emit(eventType, eventMessage string) {
	inchan <- Event{eventType: eventType, eventMessage:eventMessage}
}

func (this *Events) Listen(eventchan chan Event) {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.outchans = append(this.outchans, eventchan)
}

func (this *Events) Unlisten(eventchan chan Event) {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.remove(eventchan)
}

func (this *Events) remove(eventchan chan Event) {
	ch := make([]chan Event, 0)
	for _, c := range(this.outchans) {
		if c != eventType {
			ch = append(ch, c)
		} else {
			log.Println("Found channel, removing...")
		}
	}
	this.outchans = ch
}