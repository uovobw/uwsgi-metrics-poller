package cloudwatch_pusher

import (
	"log"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	u "github.com/uovobw/uwsgi-metrics-poller/uwsgi_poller"
)

const (
	hostLatenessTimeoutMinutes = 3
)

type CloudWatchPusher struct {
	client *cloudwatch.CloudWatch
	sync.Mutex
	NameSpace             string
	AutoscalingGroupName  string
	hostLastSeen          map[string]time.Time
	totalWorkers          map[string]float64
	idleWorkers           map[string]float64
	busyWorkers           map[string]float64
	exceptionsCount       map[string]float64
	busyWorkersPercentage map[string]float64
}

func (c *CloudWatchPusher) newDatapoint(metricName, namespace, autoscalingGroupName, unit string, value float64) (err error) {
	params := &cloudwatch.PutMetricDataInput{
		MetricData: []*cloudwatch.MetricDatum{
			{
				MetricName: aws.String(metricName),
				Dimensions: []*cloudwatch.Dimension{
					{
						Name:  aws.String("AutoscalingGroupName"),
						Value: aws.String(autoscalingGroupName),
					},
				},
				Value: aws.Float64(value),
				Unit:  aws.String(unit),
			},
		},
		Namespace: aws.String(namespace),
	}
	_, err = c.client.PutMetricData(params)
	if err != nil {
		log.Printf("error pushing metrics: %s", err)
		return err
	}
	return nil
}

func (c *CloudWatchPusher) pushAggregateMetric(name string, data map[string]float64) {
	total := float64(0.0)
	for _, datapoint := range data {
		total += datapoint
	}
	err := c.newDatapoint(name, c.NameSpace, c.AutoscalingGroupName, "Count", total)
	if err != nil {
		log.Printf("error pushing %s metric: %s", name, err)
	}
}

func (c *CloudWatchPusher) Run() {
	ticker := time.NewTicker(time.Duration(1) * time.Minute)
	for {
		select {
		case <-ticker.C:
			c.pushAggregateMetric("total-workers", c.totalWorkers)
			c.pushAggregateMetric("idle-workers", c.idleWorkers)
			c.pushAggregateMetric("busy-workers", c.busyWorkers)
			c.pushAggregateMetric("exceptions-count", c.exceptionsCount)
			c.pushAggregateMetric("busy-workers-percentage", c.busyWorkersPercentage)
		}
	}
}

func (c *CloudWatchPusher) expireOldHosts() {
	for {
		now := time.Now().Add(-hostLatenessTimeoutMinutes * time.Minute)
		for host, lastSeen := range c.hostLastSeen {
			if lastSeen.Before(now) {
				log.Printf("removing host with id %s since it has been missing for %d minutes", host, hostLatenessTimeoutMinutes)
				delete(c.totalWorkers, host)
				delete(c.busyWorkers, host)
				delete(c.idleWorkers, host)
				delete(c.busyWorkersPercentage, host)
				delete(c.exceptionsCount, host)
				delete(c.hostLastSeen, host)
			}
		}
		time.Sleep(time.Duration(30) * time.Second)
	}
}

func (c *CloudWatchPusher) HandleStat(stat *u.UwsgiStats) {
	id := stat.UniqueID()
	c.hostLastSeen[id] = time.Now()
	c.totalWorkers[id] = stat.TotalWorkers()
	c.idleWorkers[id] = stat.IdleWorkers()
	c.busyWorkers[id] = stat.BusyWorkers()
	c.busyWorkersPercentage[id] = stat.BusyWorkersPercentage()
	c.exceptionsCount[id] = stat.ExceptionsCount()
}

func New(key, secret, region, namespace, autoscalingGroupName string) (c *CloudWatchPusher, err error) {
	creds := credentials.NewStaticCredentials(key, secret, "")
	c = &CloudWatchPusher{
		client:                cloudwatch.New(session.New(), aws.NewConfig().WithRegion(region).WithCredentials(creds)),
		NameSpace:             namespace,
		AutoscalingGroupName:  autoscalingGroupName,
		totalWorkers:          make(map[string]float64),
		idleWorkers:           make(map[string]float64),
		busyWorkers:           make(map[string]float64),
		busyWorkersPercentage: make(map[string]float64),
		exceptionsCount:       make(map[string]float64),
		hostLastSeen:          make(map[string]time.Time),
	}
	err = c.checkClient()
	if err != nil {
		log.Printf("error creating cloudwatch client: %s", err)
		return nil, err
	}
	log.Printf("created cloudwatch client for region %s", region)
	go c.expireOldHosts()
	return c, nil
}

func (c *CloudWatchPusher) checkClient() (err error) {
	params := &cloudwatch.ListMetricsInput{}
	_, err = c.client.ListMetrics(params)
	return err
}
