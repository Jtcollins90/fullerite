package handler

import (
	"fullerite/metric"

	"testing"
	"time"

	l "github.com/Sirupsen/logrus"
	"github.com/samuel/go-thrift/examples/scribe"
	"github.com/stretchr/testify/assert"
)

func getTestScribeHandler(interval, buffsize, timeoutsec int) *Scribe {
	testChannel := make(chan metric.Metric)
	testLog := l.WithField("testing", "scribe_handler")
	timeout := time.Duration(timeoutsec) * time.Second

	return NewScribe(testChannel, interval, buffsize, timeout, testLog)
}

type MockScribeClient struct {
	msg []*scribe.LogEntry
}

func (m *MockScribeClient) Log(Messages []*scribe.LogEntry) (scribe.ResultCode, error) {
	m.msg = Messages
	return scribe.ResultCodeByName["ResultCode.OK"], nil
}

func TestScribeConfigureEmptyConfig(t *testing.T) {
	config := make(map[string]interface{})

	s := getTestScribeHandler(12, 13, 14)
	s.Configure(config)

	assert.Equal(t, 12, s.Interval())
	assert.Equal(t, 13, s.MaxBufferSize())
	assert.Equal(t, defaultScribeEndpoint, s.endpoint)
	assert.Equal(t, defaultScribePort, s.port)
	assert.Nil(t, s.scribeClient)
}

func TestScribeConfigure(t *testing.T) {
	config := map[string]interface{}{
		"interval":        "10",
		"timeout":         "10",
		"max_buffer_size": "100",
		"endpoint":        "1.2.3.4",
		"port":            123,
	}

	s := getTestScribeHandler(40, 50, 60)
	s.Configure(config)

	assert.Equal(t, 10, s.Interval())
	assert.Equal(t, 100, s.MaxBufferSize())
	assert.Equal(t, "1.2.3.4", s.endpoint)
	assert.Equal(t, 123, s.port)
	assert.Nil(t, s.scribeClient)
}

func TestScribeEmitMetricsNoClient(t *testing.T) {
	s := getTestScribeHandler(40, 50, 60)

	m := metric.Metric{}
	res := s.emitMetrics([]metric.Metric{m})
	assert.False(t, res, "Should not emit metrics if the scribeClient is nil")
}

func TestScribeEmitMetricsZeroMetrics(t *testing.T) {
	s := getTestScribeHandler(40, 50, 60)
	s.scribeClient = &MockScribeClient{}

	res := s.emitMetrics([]metric.Metric{})
	assert.False(t, res, "Should not emit anything if there are not metrics")
}

func TestScribeEmitMetrics(t *testing.T) {
	s := getTestScribeHandler(40, 50, 60)
	m := &MockScribeClient{}
	s.scribeClient = m

	metrics := []metric.Metric{
		metric.Metric{
			Name:       "test1",
			MetricType: metric.Gauge,
			Value:      1,
			Dimensions: map[string]string{"dim1": "val1"},
		},
		metric.Metric{
			Name:       "test2",
			Value:      2,
			MetricType: metric.Counter,
			Dimensions: map[string]string{"dim2": "val2"},
		},
	}

	res := s.emitMetrics(metrics)
	assert.True(t, res)

	assert.Equal(t, scribeStreamName, m.msg[0].Category)
	assert.Equal(t, "{\"name\":\"test1\",\"type\":\"gauge\",\"value\":1,\"dimensions\":{\"dim1\":\"val1\"}}", m.msg[0].Message)

	assert.Equal(t, scribeStreamName, m.msg[1].Category)
	assert.Equal(t, "{\"name\":\"test2\",\"type\":\"counter\",\"value\":2,\"dimensions\":{\"dim2\":\"val2\"}}", m.msg[1].Message)
}
