package etcd_watcher

import (
	"fmt"
	"log"
	"time"

	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
	"gopkg.in/fatih/set.v0"
)

const (
	HOST_ADDED = iota
	HOST_REMOVED
	HOST_PARSE_ERROR
	ETCD_UNREACHABLE
	ETCD_DIR_ERROR
	ETCD_KEY_ERROR
)

type EtcdEvent struct {
	Reason int
	Data   interface{}
}

func (e *EtcdEvent) String() string {
	evts := map[int]string{
		HOST_ADDED:       "host added",
		HOST_REMOVED:     "host removed",
		HOST_PARSE_ERROR: "error parsing host format",
		ETCD_UNREACHABLE: "etcd is unreachable",
		ETCD_DIR_ERROR:   "unable to reach etcd dir",
		ETCD_KEY_ERROR:   "unable to read key",
	}
	msg, _ := evts[e.Reason]
	return fmt.Sprintf("%s. data: %+v", msg, e.Data)
}

func makeEtcdEvent(reason int, data interface{}) *EtcdEvent {
	return &EtcdEvent{
		Reason: reason,
		Data:   data,
	}
}

type EtcdWatcher struct {
	Endpoints  []string
	Dir        string
	PollTime   time.Duration
	ticker     *time.Ticker
	client     client.KeysAPI
	hosts      *set.Set
	EventsChan chan<- *EtcdEvent
}

func NewEtcdWatcher(endpoints []string, dir string, pollTime int, eventsChan chan<- *EtcdEvent) (e *EtcdWatcher, err error) {
	e = &EtcdWatcher{
		Endpoints:  endpoints,
		Dir:        dir,
		PollTime:   time.Duration(pollTime),
		ticker:     time.NewTicker(time.Duration(pollTime) * time.Second),
		hosts:      set.New(),
		EventsChan: eventsChan,
	}

	cfg := client.Config{
		Endpoints:               e.Endpoints,
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	}
	c, err := client.New(cfg)
	if err != nil {
		log.Fatalf("error creating etcd watcher: %s", err)
	}
	e.client = client.NewKeysAPI(c)
	log.Printf("created etcd watcher on directory %s for hosts %s polling time %d seconds", dir, endpoints, pollTime)
	return e, nil
}

func (e *EtcdWatcher) getSingleNode(n *client.Node) (s string, err error) {
	resp, err := e.client.Get(context.Background(), n.Key, nil)
	if err != nil {
		log.Printf("error reading key %s: %s", n.Key, err)
		return "", err
	}
	return resp.Node.Value, nil
}

func (e *EtcdWatcher) handleHosts(newSet *set.Set) {
	for _, host := range newSet.List() {
		if !e.hosts.Has(host) {
			log.Printf("ADD evt %s", host)
			e.EventsChan <- makeEtcdEvent(HOST_ADDED, host)
			e.hosts.Add(host)
		}
	}
	for _, host := range e.hosts.List() {
		if !newSet.Has(host) {
			log.Printf("REMOVE evt %s", host)
			e.EventsChan <- makeEtcdEvent(HOST_REMOVED, host)
			e.hosts.Remove(host)
		}
	}
}

func (e *EtcdWatcher) Run() {
	firstRun := true
	for {
		select {
		case <-e.ticker.C:
			resp, err := e.client.Get(context.Background(), e.Dir, nil)
			if err != nil {
				log.Printf("error reading key %s: %s", e.Dir, err)
				e.EventsChan <- makeEtcdEvent(ETCD_DIR_ERROR, err)
				return
			}
			if resp.Node.Dir {
				newSet := set.New()
				for _, k := range resp.Node.Nodes {
					if k.Dir {
						// we do not need to recurse into any dir here
						continue
					}
					str, err := e.getSingleNode(k)
					if err != nil {
						log.Printf("error getting key %s: %s", k.Key, err)
						e.EventsChan <- makeEtcdEvent(ETCD_KEY_ERROR, err)
						continue
					}
					newSet.Add(str)
				}
				if firstRun {
					for _, h := range newSet.List() {
						log.Printf("found initial host: %s", h)
						e.hosts.Add(h)
						e.EventsChan <- makeEtcdEvent(HOST_ADDED, h)
					}
					firstRun = false
				} else {
					e.handleHosts(newSet)
				}
			} else {
				log.Printf("the key provided is not a directory: %s", e.Dir)
				e.EventsChan <- makeEtcdEvent(ETCD_DIR_ERROR, e.Dir)
				return
			}
		}
	}
}
