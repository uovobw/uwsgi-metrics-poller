package uwsgi_poller

import (
	"fmt"
	"strconv"
	"strings"
)

type UwsgiHosts []*UwsgiHost

type UwsgiHost struct {
	Host string
	Port int
}

func (u *UwsgiHost) String() string {
	return fmt.Sprintf("host:%s port:%d", u.Host, u.Port)
}

func UwsgiHostFromString(str string) (u *UwsgiHost, err error) {
	host := strings.Split(str, ":")
	if len(host) != 2 {
		return nil, fmt.Errorf("error parsing string %s", str)
	}
	u = &UwsgiHost{
		Host: host[0],
	}
	port, err := strconv.ParseInt(host[1], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("error parsing port %s: %s", host[1], err)
	}
	u.Port = int(port)
	return u, nil
}
