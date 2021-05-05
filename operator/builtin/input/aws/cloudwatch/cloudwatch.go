package Cloudwatch

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	"go.uber.org/zap"
)

const operatorName = "aws_cloudwatch_input"

func init() {
	operator.Register(operatorName, func() operator.Builder { return NewCloudwatchConfig("") })
}

// NewCloudwatchConfig creates a new AWS Cloudwatch Logs input config with default values
func NewCloudwatchConfig(operatorID string) *CloudwatchInputConfig {
	return &CloudwatchInputConfig{
		InputConfig:  helper.NewInputConfig(operatorID, operatorName),
		Limit:        10000,
		StartAt:      "end",
		PollInterval: helper.Duration{Duration: time.Minute * 1},
	}
}

// CloudwatchInputConfig is the configuration of a AWS Cloudwatch Logs input operator.
type CloudwatchInputConfig struct {
	helper.InputConfig `yaml:",inline"`

	// required
	LogGroupName string `json:"log_group_name,omitempty" yaml:"log_group_name,omitempty"`
	Region       string `json:"region,omitempty" yaml:"region,omitempty"`
	Profile      string `json:"profile,omitempty" yaml:"profile,omitempty"`

	// optional
	LogStreamNamePrefix string          `json:"log_stream_name_prefix,omitempty" yaml:"log_stream_name_prefix,omitempty"`
	Limit               int64           `json:"limit,omitempty" yaml:"limit,omitempty"`
	PollInterval        helper.Duration `json:"poll_interval,omitempty" yaml:"poll_interval,omitempty"`
	StartAt             string          `json:"start_at,omitempty" yaml:"start_at,omitempty"`
}

// Build will build a AWS Cloudwatch Logs input operator.
func (c *CloudwatchInputConfig) Build(buildContext operator.BuildContext) ([]operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(buildContext)
	if err != nil {
		return nil, err
	}

	if c.LogGroupName == "" {
		return nil, fmt.Errorf("missing required %s parameter 'log_group_name'", operatorName)
	}

	if c.Region == "" {
		return nil, fmt.Errorf("missing required %s parameter 'region'", operatorName)
	}

	if c.Limit < 1 {
		return nil, fmt.Errorf("invalid value '%d' for %s parameter 'limit'. Parameter 'limit' must be a value between 1 - 10000", c.Limit, operatorName)
	}

	if c.Limit > 10000 {
		return nil, fmt.Errorf("invalid value '%d' for %s parameter 'limit'. Parameter 'limit' must be a value between 1 - 10000", c.Limit, operatorName)
	}

	if c.StartAt != "beginning" && c.StartAt != "end" {
		return nil, fmt.Errorf("invalid value for parameter 'start_at'")
	}

	var startAtEnd bool
	switch c.StartAt {
	case "beginning":
		startAtEnd = false
	case "end":
		startAtEnd = true
	default:
		return nil, fmt.Errorf("invalid value '%s' for %s parameter 'start_at'", c.StartAt, operatorName)
	}

	cloudwatchInput := &CloudwatchInput{
		InputOperator:       inputOperator,
		logGroupName:        c.LogGroupName,
		logStreamNamePrefix: c.LogStreamNamePrefix,
		region:              c.Region,
		limit:               c.Limit,
		profile:             c.Profile,
		pollInterval:        c.PollInterval,
		startAtEnd:          startAtEnd,
		startTime:           0,
		persist:             *helper.NewScopedDBPersister(buildContext.Database, c.ID()),
	}
	return []operator.Operator{cloudwatchInput}, nil
}

// CloudwatchInput is an operator that reads input from AWS Cloudwatch Logs.
type CloudwatchInput struct {
	helper.InputOperator
	cancel       context.CancelFunc
	pollInterval helper.Duration

	logGroupName        string
	logStreamNamePrefix string
	region              string
	limit               int64
	profile             string
	startAtEnd          bool
	startTime           int64
	persist             helper.ScopedBBoltPersister
	wg                  sync.WaitGroup
}

// Start will start generating log entries.
func (c *CloudwatchInput) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel

	if err := c.persist.Load(); err != nil {
		return err
	}

	err := c.pollEvents(ctx)
	if err != nil {
		return err
	}

	return nil
}

// Stop will stop generating logs.
func (c *CloudwatchInput) Stop() error {
	c.cancel()
	c.wg.Wait()
	fmt.Printf("Closed all connections to Cloudwatch Logs")
	return nil
}

// pollEvents gets events from AWS Cloudwatch Logs every poll interval.
func (c *CloudwatchInput) pollEvents(ctx context.Context) error {
	c.Info("Started polling AWS Cloudwatch using poll interval of ", c.pollInterval)
	defer c.wg.Done()

	region := aws.String(c.region)
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config: aws.Config{Region: region},

		Profile: c.profile,
	}))

	svc := cloudwatchlogs.New(sess)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(c.pollInterval.Duration):
			c.Debug("Getting events from AWS Cloudwatch Logs")
			nextToken := ""
			st := c.persist.Get(c.logGroupName)
			c.startTime = BytesToInt64(st)
			if c.startAtEnd && c.startTime == 0 {
				c.Info("Setting start time to current time")
				c.startTime = currentTimeInUnixMilliseconds()
			}
			c.Info("Start Time: ", c.startTime)
			for {
				input := c.filterLogEventsInputBuilder(nextToken)

				resp, err := svc.FilterLogEvents(&input)
				if err != nil {
					c.Errorf("failed to get events: %s", err)
					continue
				}

				if len(resp.Events) == 0 {
					c.Debug("No events from AWS Cloudwatch Logs")
				}

				c.handleBatchedEvents(ctx, resp.Events)

				if resp.NextToken == nil {
					c.Debug("Finished getting events")
					break
				}
				nextToken = *resp.NextToken
				c.Debug("Reached event limit '%d'", c.limit)
				c.persist.Sync()
			}
		}
	}
}

// filterLogEventsInputBuilder builds AWS Cloudwatch Logs Filter Log Events Input based on provided values
// and returns completed input.
func (c *CloudwatchInput) filterLogEventsInputBuilder(nextToken string) cloudwatchlogs.FilterLogEventsInput {
	logGroupNamePtr := aws.String(c.logGroupName)
	limit := aws.Int64(c.limit)

	if c.logStreamNamePrefix != "" && nextToken != "" {
		logStreamNamePrefixPtr := aws.String(c.logStreamNamePrefix)
		nextTokenPtr := aws.String(nextToken)
		return cloudwatchlogs.FilterLogEventsInput{
			Limit:               limit,
			LogGroupName:        logGroupNamePtr,
			LogStreamNamePrefix: logStreamNamePrefixPtr,
			StartTime:           aws.Int64(c.startTime),
			NextToken:           nextTokenPtr,
		}
	}

	if c.logStreamNamePrefix != "" {
		logStreamNamePrefixPtr := aws.String(c.logStreamNamePrefix)
		return cloudwatchlogs.FilterLogEventsInput{
			Limit:               limit,
			LogGroupName:        logGroupNamePtr,
			LogStreamNamePrefix: logStreamNamePrefixPtr,
			StartTime:           aws.Int64(c.startTime),
		}
	}

	if nextToken != "" {
		nextTokenPtr := aws.String(nextToken)
		return cloudwatchlogs.FilterLogEventsInput{
			Limit:        limit,
			LogGroupName: logGroupNamePtr,
			StartTime:    aws.Int64(c.startTime),
			NextToken:    nextTokenPtr,
		}
	}

	return cloudwatchlogs.FilterLogEventsInput{
		Limit:        limit,
		LogGroupName: logGroupNamePtr,
		StartTime:    aws.Int64(c.startTime),
	}
}

// handleEvent is the handler for a AWS Cloudwatch Logs Filtered Event.
func (c *CloudwatchInput) handleEvent(ctx context.Context, event *cloudwatchlogs.FilteredLogEvent) error {
	// c.wg.Add(1)

	e := make(map[string]interface{})
	e["message"] = event.Message
	e["event_id"] = event.EventId
	e["log_stream_name"] = event.LogStreamName
	e["timestamp"] = event.Timestamp
	e["ingestion_time"] = event.IngestionTime

	entry, err := c.NewEntry(nil)
	if err != nil {
		c.Error("Failed to create new entry from record", zap.Error(err))
	}
	// entry.Timestamp = time.Unix(0, *event.Timestamp*int64(time.Millisecond))
	entry.Timestamp = FromUnixMilli(*event.Timestamp)
	entry.Record = e
	// Persist
	if *event.IngestionTime > c.startTime {
		c.persist.Set(c.logGroupName, Int64ToBytes(*event.IngestionTime))
	}
	// Write Entry
	c.Write(ctx, entry)
	// c.wg.Done()
	return nil
}

func (c *CloudwatchInput) handleBatchedEvents(ctx context.Context, events []*cloudwatchlogs.FilteredLogEvent) error {
	c.wg.Add(1)
	defer c.wg.Done()

	// Create an entry for each log in the batch.
	wg := sync.WaitGroup{}
	max := 10
	gaurd := make(chan struct{}, max)
	for i := 0; i < len(events); i++ {
		e := events[i]
		wg.Add(1)
		gaurd <- struct{}{}
		go func() {
			defer func() {
				wg.Done()
				<-gaurd
			}()
			c.handleEvent(ctx, e)
		}()
	}
	wg.Wait()
	return nil
}

// Returns time.Now() as Unix Time in Milliseconds
func currentTimeInUnixMilliseconds() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// Helper function to persist start time
func Int64ToBytes(i int64) []byte {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

// Helper function to get persisted start time
func BytesToInt64(buf []byte) int64 {
	var startTime int64
	buffer := bytes.NewBuffer(buf)
	binary.Read(buffer, binary.BigEndian, &startTime)
	return startTime
}

// Helper function to convert Unix epoch time in milliseconds to go time
func FromUnixMilli(ms int64) time.Time {
	const millisInSecond = 1000
	const nsInSecond = 1000000
	return time.Unix(ms/int64(millisInSecond), (ms%int64(millisInSecond))*int64(nsInSecond))
}
