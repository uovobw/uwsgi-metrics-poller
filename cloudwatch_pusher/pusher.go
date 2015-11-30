package cloudwatch_pusher

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

type CloudWatchPusher struct {
	client *cloudwatch.CloudWatch
}

func (c *CloudWatchPusher) PushMetric(metricName, namespace, autoscalingGroupName, unit string, value float64, min, max, count, sum float64) (err error) {
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
				StatisticValues: &cloudwatch.StatisticSet{
					Maximum:     aws.Float64(max),
					Minimum:     aws.Float64(min),
					SampleCount: aws.Float64(count),
					Sum:         aws.Float64(sum),
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

func New(key, secret, region string) (c *CloudWatchPusher, err error) {
	creds := credentials.NewStaticCredentials(key, secret, "")
	c = &CloudWatchPusher{
		client: cloudwatch.New(session.New(), aws.NewConfig().WithRegion(region).WithCredentials(creds)),
	}
	err = c.checkClient()
	if err != nil {
		log.Printf("error creating cloudwatch client: %s", err)
		return nil, err
	}
	log.Printf("created cloudwatch client for region %s", region)
	return c, nil
}

func (c *CloudWatchPusher) checkClient() (err error) {
	params := &cloudwatch.ListMetricsInput{}
	_, err = c.client.ListMetrics(params)
	return err
}
