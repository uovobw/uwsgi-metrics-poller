package uwsgi_poller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

const (
	PARSE_ERROR = iota
	HOST_UNREACHABLE
	QUIT_RECEIVED

	maxHostRetries = 10
)

type UwsgiPoller struct {
	Address    *net.TCPAddr
	Period     time.Duration
	StatsChan  chan<- *UwsgiStats
	EventsChan chan<- *UwsgiEvent
	quitChan   chan int
	ticker     *time.Ticker
}

type UwsgiEvent struct {
	reason int
}

func New(addr string, period int, outdata chan<- *UwsgiStats, events chan<- *UwsgiEvent, quit chan int) (p *UwsgiPoller, err error) {
	a, err := net.ResolveTCPAddr("tcp4", addr)
	if err != nil {
		return nil, err
	}
	pd := time.Duration(period) * time.Second
	p = &UwsgiPoller{
		Address:    a,
		Period:     pd,
		StatsChan:  outdata,
		EventsChan: events,
		quitChan:   quit,
		ticker:     time.NewTicker(pd),
	}
	log.Printf("created poller for %s interval %s", addr, period)
	return p, nil
}

func (p *UwsgiPoller) getStats() (s *UwsgiStats, err error) {
	conn, err := net.DialTCP("tcp4", nil, p.Address)
	if err != nil {
		log.Printf("error reading from remote: %s", err)
		return nil, err
	}
	defer conn.Close()
	var buf bytes.Buffer
	io.Copy(&buf, conn)
	s = &UwsgiStats{}
	err = json.Unmarshal(buf.Bytes(), s)
	if err != nil {
		log.Printf("error loading stats, this will probably repeat in the future, quitting")
		p.quitChan <- 1
		return nil, fmt.Errorf("error loading uwsgi response: %s.", err)
	}
	return s, nil
}

func (p *UwsgiPoller) Run() {
	log.Printf("poller for %s running", p.Address.String())
	go func(poller *UwsgiPoller) {
		unreachableCount := 0
		for {
			select {
			case <-poller.ticker.C:
				data, err := poller.getStats()
				if err != nil {
					log.Printf("error getting stats: %s. host might be down", err)
					unreachableCount += 1
					if unreachableCount == maxHostRetries {
						e := &UwsgiEvent{
							reason: HOST_UNREACHABLE,
						}
						poller.EventsChan <- e
						log.Printf("maximum number of retries exceeded, goroutine for host %s quitting", poller.Address.String())
						return
					}
				} else {
					poller.StatsChan <- data
				}
			case <-poller.quitChan:
				e := &UwsgiEvent{
					reason: QUIT_RECEIVED,
				}
				poller.EventsChan <- e
				log.Printf("poller for %s quitting", poller.Address.String())
				return
			}
		}
	}(p)
}
