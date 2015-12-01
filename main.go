package main

import (
	"fmt"
	"log"
	"strings"

	cw "github.com/uovobw/uwsgi-metrics-poller/cloudwatch_pusher"
	etcd "github.com/uovobw/uwsgi-metrics-poller/etcd_watcher"
	uwsgi "github.com/uovobw/uwsgi-metrics-poller/uwsgi_poller"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	version = "0.1"
)

var (
	debug               = kingpin.Flag("debug", "enable debug mode.").Short('d').Bool()
	etcdHosts           = kingpin.Flag("etcd-hosts", "comma separated etcd hosts in the format host:port").Short('e').Default("localhost:4001").Strings()
	etcdWatchKeys       = kingpin.Flag("etcd-watch-dirs", "comma separated etcd directories to watch for hosts").Short('k').Default("/").Strings()
	etcdWatchPeriod     = kingpin.Flag("etcd-watch-period", "polling period for the etcd key in seconds").Short('p').Default("30").Int()
	uwsgiPollingPeriod  = kingpin.Flag("uwsgi-polling-period", "polling period in seconds for the uwsgi stats").Short('u').Default("30").Int()
	uwsgiStatsPort      = kingpin.Flag("uwsgi-stats-port", "port to hit for the uwsgi stats").Short('P').Default("12321").Int()
	awsSecretKey        = kingpin.Flag("aws-secret-key", "AWS account secret").String()
	awsAccessKey        = kingpin.Flag("aws-access-key", "AWS account key").String()
	awsRegion           = kingpin.Flag("aws-region", "AWS region in which to log").Default("eu-west-1").String()
	awsNamespace        = kingpin.Flag("aws-namespace", "AWS namespace name for the cloudwatch metric").String()
	awsAutoscalingGroup = kingpin.Flag("aws-autoscaling-group", "AWS autoscaling group name").String()

	cloudwatchPusher *cw.CloudWatchPusher
	etcdWatchers     map[string]*etcd.EtcdWatcher
	etcdEventsChan   chan *etcd.EtcdEvent
	uwsgiStatsChan   chan *uwsgi.UwsgiStats
	uwsgiEventsChan  chan *uwsgi.UwsgiEvent
	quitChan         chan int
	err              error
)

func init() {
	etcdWatchers = make(map[string]*etcd.EtcdWatcher, 100)
	etcdEventsChan = make(chan *etcd.EtcdEvent, 100)
	uwsgiStatsChan = make(chan *uwsgi.UwsgiStats, 100)
	uwsgiEventsChan = make(chan *uwsgi.UwsgiEvent, 100)
	quitChan = make(chan int)
}

func getUwsgiStatsConnectionString(host string) string {
	parts := strings.Split(host, ":")
	if len(parts) != 2 {
		panic(fmt.Sprintf("received host specification different from <host>:<port> (was %s). aborting", host))
	}
	return fmt.Sprintf("%s:%d", parts[0], *uwsgiStatsPort)
}

func main() {
	kingpin.Version(version)
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.Parse()

	defer log.Println("terminating")

	log.Printf("starting uwsgi poller (%s)", version)
	if *debug {
		log.Printf("running against etcd host(s) %s with key %s period %d uwsgi polling time %d uwsgi port %d", *etcdHosts, *etcdWatchKeys, *etcdWatchPeriod, *uwsgiPollingPeriod, *uwsgiStatsPort)
	}

	cloudwatchPusher, err = cw.New(*awsAccessKey, *awsSecretKey, *awsRegion, *awsNamespace, *awsAutoscalingGroup)
	if err != nil {
		log.Fatalf("cannot create cloudwatch pusher: %s", err)
	}
	go cloudwatchPusher.Run()

	for _, key := range *etcdWatchKeys {
		watcher, err := etcd.NewEtcdWatcher(*etcdHosts, key, *etcdWatchPeriod, etcdEventsChan)
		if err != nil {
			log.Fatalf("could not initialize etcd watcher: %s", err)
		}
		go watcher.Run()
		etcdWatchers[key] = watcher
	}

	// handle events from the etcd watcher
	go func(evtChan chan *etcd.EtcdEvent) {
		for evt := range evtChan {
			log.Printf("received event from etcd watcher %s", evt)
			switch evt.Reason {
			case etcd.HOST_ADDED:
				p, err := uwsgi.New(getUwsgiStatsConnectionString(evt.Data.(string)), *uwsgiPollingPeriod, uwsgiStatsChan, uwsgiEventsChan, quitChan)
				if err != nil {
					log.Printf("error creating new uwsgi poller for %s", evt)
				}
				p.Run()
			case etcd.HOST_REMOVED:
				log.Printf("host removed %s", evt)
			case etcd.HOST_PARSE_ERROR:
				log.Fatalf("unexpected format in host string %s aborting", evt)
			case etcd.ETCD_UNREACHABLE:
				log.Fatalf("cannot reach etcd host(s), aborting")
			case etcd.ETCD_DIR_ERROR:
				log.Fatalf("directory error, aborting: %s", evt)
			case etcd.ETCD_KEY_ERROR:
				log.Fatalf("key value error, aborting: %s", evt)
			}
		}
	}(etcdEventsChan)

	// handle events from uwsgi pollers
	go func(evtChan chan *uwsgi.UwsgiEvent) {
		for uwsgiEvent := range evtChan {
			log.Printf("received event from uwsgi poller: %s", uwsgiEvent)
		}
	}(uwsgiEventsChan)

	// handle stats coming from the uwsgi pollers
	go func(statsChan chan *uwsgi.UwsgiStats) {
		for stat := range statsChan {
			//log.Printf("received stats from uwsgi poller: %s", stat)
			cloudwatchPusher.HandleStat(stat)
		}
	}(uwsgiStatsChan)

	select {}
}
