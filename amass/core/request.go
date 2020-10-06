// Copyright 2017 Jeff Foley. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package core

import (
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/root-secure/Amass/amass/utils"
)

// Request tag types.
const (
	ALT      = "alt"
	ARCHIVE  = "archive"
	API      = "api"
	AXFR     = "axfr"
	BRUTE    = "brute"
	CERT     = "cert"
	DNS      = "dns"
	EXTERNAL = "ext"
	SCRAPE   = "scrape"
)

// Request Pub/Sub topics used across Amass.
const (
	NewNameTopic      = "amass:newname"
	NewAddrTopic      = "amass:newaddr"
	NewSubdomainTopic = "amass:newsub"
	ResolveNameTopic  = "amass:resolve"
	NameResolvedTopic = "amass:resolved"
	ReverseSweepTopic = "amass:sweep"
	ActiveCertTopic   = "amass:activecert"
	OutputTopic       = "amass:output"
	IPToASNTopic      = "amass:iptoasn"
	NewASNTopic       = "amass:asn"
	WhoisRequestTopic = "amass:whoisreq"
	NewWhoisTopic     = "amass:whoisinfo"
)

// DNSAnswer is the type used by Amass to represent a DNS record.
type DNSAnswer struct {
	Name string `json:"name"`
	Type int    `json:"type"`
	TTL  int    `json:"TTL"`
	Data string `json:"data"`
}

// DNSRequest handles data needed throughout Service processing of a DNS name.
type DNSRequest struct {
	Name    string
	Domain  string
	Records []DNSAnswer
	Tag     string
	Source  string
}

// AddrRequest handles data needed throughout Service processing of a network address.
type AddrRequest struct {
	Address string
	Domain  string
	Tag     string
	Source  string
}

// ASNRequest handles all autonomous system information needed by Amass.
type ASNRequest struct {
	Address        string
	ASN            int
	Prefix         string
	CC             string
	Registry       string
	AllocationDate time.Time
	Description    string
	Netblocks      []string
	Tag            string
	Source         string
}

// WhoisRequest handles data needed throughout Service processing of reverse whois.
type WhoisRequest struct {
	Domain     string
	Company    string
	Email      string
	NewDomains []string
	Tag        string
	Source     string
}

// Output contains all the output data for an enumerated DNS name.
type Output struct {
	Timestamp time.Time
	Name      string        `json:"name"`
	Domain    string        `json:"domain"`
	Addresses []AddressInfo `json:"addresses"`
	Tag       string        `json:"tag"`
	Source    string        `json:"source"`
}

// AddressInfo stores all network addressing info for the Output type.
type AddressInfo struct {
	Address     net.IP     `json:"ip"`
	Netblock    *net.IPNet `json:"-"`
	CIDRStr     string     `json:"cidr"`
	ASN         int        `json:"asn"`
	Description string     `json:"desc"`
}

type pubReq struct {
	Topic string
	Args  []reflect.Value
}

// EventBus handles sending and receiving events across Amass.
type EventBus struct {
	sync.Mutex
	topics map[string][]reflect.Value
	max    utils.Semaphore
	queue  *utils.Queue
	done   chan struct{}
}

// NewEventBus initializes and returns an EventBus object.
func NewEventBus() *EventBus {
	eb := &EventBus{
		topics: make(map[string][]reflect.Value),
		max:    utils.NewSimpleSemaphore(1000000),
		queue:  utils.NewQueue(),
		done:   make(chan struct{}, 2),
	}
	go eb.processRequests()
	return eb
}

// Subscribe registers callback to be executed for all requests on the channel.
func (eb *EventBus) Subscribe(topic string, fn interface{}) {
	if topic == "" || reflect.TypeOf(fn).Kind() != reflect.Func {
		return
	}

	callback := reflect.ValueOf(fn)

	eb.Lock()
	eb.topics[topic] = append(eb.topics[topic], callback)
	eb.Unlock()
}

// Unsubscribe deregisters the callback from the channel.
func (eb *EventBus) Unsubscribe(topic string, fn interface{}) {
	if topic == "" || reflect.TypeOf(fn).Kind() != reflect.Func {
		return
	}

	callback := reflect.ValueOf(fn)

	eb.Lock()
	defer eb.Unlock()

	var channels []reflect.Value
	for _, c := range eb.topics[topic] {
		if c != callback {
			channels = append(channels, c)
		}
	}
	eb.topics[topic] = channels
}

// Publish sends req on the channel labeled with name.
func (eb *EventBus) Publish(topic string, args ...interface{}) {
	if topic == "" {
		return
	}

	passedArgs := make([]reflect.Value, 0)
	for _, arg := range args {
		passedArgs = append(passedArgs, reflect.ValueOf(arg))
	}

	eb.queue.Append(&pubReq{
		Topic: topic,
		Args:  passedArgs,
	})
}

func (eb *EventBus) processRequests() {
	curIdx := 0
	maxIdx := 7
	delays := []int{10, 25, 50, 75, 100, 150, 250, 500}

	for {
		select {
		case <-eb.done:
			return
		default:
			element, ok := eb.queue.Next()
			if !ok {
				if curIdx < maxIdx {
					curIdx++
				}
				time.Sleep(time.Duration(delays[curIdx]) * time.Millisecond)
				continue
			}

			curIdx = 0
			p := element.(*pubReq)

			eb.Lock()
			callbacks, found := eb.topics[p.Topic]
			eb.Unlock()

			if found {
				eb.max.Acquire(1)
				go eb.executeCallbacks(callbacks, p.Args)
			}
		}
	}
}

func (eb *EventBus) executeCallbacks(callbacks, args []reflect.Value) {
	defer eb.max.Release(1)

	for _, cb := range callbacks {
		cb.Call(args)
	}
}

// Stop prevents any additional requests from being sent.
func (eb *EventBus) Stop() {
	close(eb.done)
}
