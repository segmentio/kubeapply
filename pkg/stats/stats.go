package stats

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/segmentio/stats/v4"
	godatadog "gopkg.in/zorkian/go-datadog-api.v2"
)

// StatType is a string type that stores the type of stat being exported.
type StatType string

const (
	// StatTypeGauge represents the type of a guage stat.
	StatTypeGauge StatType = "gauge"

	// StatTypeCount represents the type of a count stat.
	StatTypeCount StatType = "count"
)

// StatsClient is an interface for exposing stats from kubeapply lambda runs.
type StatsClient interface {
	Update(names []string, values []float64, tags []string, statType StatType) error
}

// NullStatsClient is a StatsClient implementation that does not export stats.
type NullStatsClient struct {
}

// Update updates the given stats.
func (n *NullStatsClient) Update(
	names []string,
	values []float64,
	tags []string,
	statType StatType,
) error {
	return nil
}

// FakeStatsClient is a fake implementation of StatsClient for testing purposes.
type FakeStatsClient struct {
	Stats map[string]float64
}

// NewFakeStatsClient returns a new FakeStatsClient instance.
func NewFakeStatsClient() *FakeStatsClient {
	return &FakeStatsClient{
		Stats: map[string]float64{},
	}
}

// Update updates the given stats.
func (s *FakeStatsClient) Update(
	names []string,
	values []float64,
	tags []string,
	statType StatType,
) error {
	if len(names) != len(values) {
		return errors.New("Names and values must be same length")
	}

	for n := 0; n < len(names); n++ {
		s.Stats[names[n]] += values[n]
	}

	return nil
}

// DatadogStatsClient is an implementation of StatsClient that sends stats to the Datadog API
// directly.
type DatadogStatsClient struct {
	prefix      string
	baseTags    []string
	datadogHost string
	client      *godatadog.Client
}

// NewDatadogStatsClient creates a new DatadogStatsClient instance.
func NewDatadogStatsClient(
	prefix string,
	baseTags []string,
	datadogHost string,
	apiKey string,
) *DatadogStatsClient {
	return &DatadogStatsClient{
		prefix:      prefix,
		baseTags:    baseTags,
		datadogHost: datadogHost,
		client:      godatadog.NewClient(apiKey, ""),
	}
}

// Update sends the argument stat values to Datadog.
func (d *DatadogStatsClient) Update(
	names []string,
	values []float64,
	tags []string,
	statType StatType,
) error {
	if len(names) != len(values) {
		return errors.New("Names and values must be same length")
	}

	metrics := []godatadog.Metric{}
	now := float64(time.Now().UTC().Unix())

	combinedTags := []string{}

	for _, tag := range d.baseTags {
		combinedTags = append(combinedTags, tag)
	}
	for _, tag := range tags {
		combinedTags = append(combinedTags, tag)
	}

	for n, name := range names {
		metrics = append(
			metrics,
			godatadog.Metric{
				Metric: aws.String(d.prefix + name),
				Points: []godatadog.DataPoint{
					{
						aws.Float64(now),
						aws.Float64(values[n]),
					},
				},
				Tags: combinedTags,
				Type: aws.String(string(statType)),
				Host: aws.String(d.datadogHost),
			},
		)
	}

	return d.client.PostMetrics(metrics)
}

// SegmentStatsClient is an implementation of StatsClient that wraps a Segment stats.Engine.
type SegmentStatsClient struct {
	engine *stats.Engine
}

// NewSegmentStatsClient returns a new SegmentStatsClient instance.
func NewSegmentStatsClient(engine *stats.Engine) *SegmentStatsClient {
	return &SegmentStatsClient{
		engine: engine,
	}
}

// Update updates the argument stat values by calling the appropriate methods
// in the underlying stats engine.
func (s *SegmentStatsClient) Update(
	names []string,
	values []float64,
	tags []string,
	statType StatType,
) error {
	segmentTags := []stats.Tag{}

	for _, tag := range tags {
		components := strings.SplitN(tag, ":", 2)
		if len(components) != 2 {
			continue
		}

		segmentTags = append(
			segmentTags,
			stats.Tag{
				Name:  components[0],
				Value: components[1],
			},
		)
	}

	for _, name := range names {
		for _, value := range values {
			switch statType {
			case StatTypeCount:
				s.engine.Add(name, value, segmentTags...)
			case StatTypeGauge:
				s.engine.Set(name, value, segmentTags...)
			default:
				return fmt.Errorf("Unrecognized stat type: %+v", statType)
			}
		}
	}

	return nil
}

// GetDatadogAPIKey gets the Datadog API key from an AWS SSM parameter.
func GetDatadogAPIKey(sess *session.Session, apiKeySSMParam string) (string, error) {
	ssmClient := ssm.New(sess)
	result, err := ssmClient.GetParameter(
		&ssm.GetParameterInput{
			Name:           aws.String(apiKeySSMParam),
			WithDecryption: aws.Bool(true),
		},
	)
	if err != nil {
		return "", err
	}
	return aws.StringValue(result.Parameter.Value), nil
}
