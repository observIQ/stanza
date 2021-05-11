package cloudwatch

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/observiq/stanza/errors"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
)

const operatorName = "aws_cloudwatch_input"
const eventLimit = 10000 //The maximum number of events to return. The default is up to 10,000 events or max of 1mb.

func init() {
	operator.Register(operatorName, func() operator.Builder { return NewCloudwatchConfig("") })
}

// NewCloudwatchConfig creates a new AWS Cloudwatch Logs input config with default values
func NewCloudwatchConfig(operatorID string) *CloudwatchInputConfig {
	return &CloudwatchInputConfig{
		InputConfig: helper.NewInputConfig(operatorID, operatorName),

		EventLimit:   eventLimit,
		PollInterval: helper.Duration{Duration: time.Minute * 1},
		StartAt:      "end",
	}
}

// CloudwatchInputConfig is the configuration of a AWS Cloudwatch Logs input operator.
type CloudwatchInputConfig struct {
	helper.InputConfig `yaml:",inline"`

	// required
	LogGroupName string `json:"log_group_name,omitempty" yaml:"log_group_name,omitempty"`
	Region       string `json:"region,omitempty" yaml:"region,omitempty"`

	// optional
	LogStreamNamePrefix string          `json:"log_stream_name_prefix,omitempty" yaml:"log_stream_name_prefix,omitempty"`
	LogStreamNames      []*string       `json:"log_stream_names,omitempty" yaml:"log_stream_names,omitempty"`
	Profile             string          `json:"profile,omitempty" yaml:"profile,omitempty"`
	EventLimit          int64           `json:"event_limit,omitempty" yaml:"event_limit,omitempty"`
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

	if len(c.LogStreamNames) > 0 && c.LogStreamNamePrefix != "" {
		return nil, fmt.Errorf("must only use 'log_stream_names' or 'log_stream_name_prefix' %s parameters, cannot use both", operatorName)
	}

	if c.Region == "" {
		return nil, fmt.Errorf("missing required %s parameter 'region'", operatorName)
	}

	if c.EventLimit < 1 || c.EventLimit > 10000 {
		return nil, fmt.Errorf("invalid value '%d' for %s parameter 'event_limit'. Parameter 'event_limit' must be a value between 1 - 10000", c.EventLimit, operatorName)
	}

	if c.PollInterval.Raw() < time.Second*10 {
		return nil, fmt.Errorf("invalid value '%s' for %s parameter 'poll_interval'. Parameter 'poll_interval' must be a value of 10 seconds or greater", c.PollInterval.String(), operatorName)
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
		logStreamNames:      c.LogStreamNames,
		logStreamNamePrefix: c.LogStreamNamePrefix,
		region:              c.Region,
		eventLimit:          c.EventLimit,
		profile:             c.Profile,
		pollInterval:        c.PollInterval,
		startAtEnd:          startAtEnd,
		persist: Persister{
			DB: helper.NewScopedDBPersister(buildContext.Database, c.ID()),
		},
	}
	return []operator.Operator{cloudwatchInput}, nil
}

// CloudwatchInput is an operator that reads input from AWS Cloudwatch Logs.
type CloudwatchInput struct {
	helper.InputOperator
	cancel       context.CancelFunc
	pollInterval helper.Duration

	logGroupName        string
	logStreamNames      []*string
	logStreamNamePrefix string
	region              string
	eventLimit          int64
	profile             string
	startAtEnd          bool
	startTime           int64
	persist             Persister
	wg                  sync.WaitGroup
}

// Start will start generating log entries.
func (c *CloudwatchInput) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel

	if err := c.persist.DB.Load(); err != nil {
		return err
	}

	return c.pollEvents(ctx)
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
	c.Infof("Started polling AWS Cloudwatch Logs group '%s' using poll interval of '%s'", c.logGroupName, c.pollInterval)
	defer c.wg.Done()

	// Create session to use when connecting to AWS Cloudwatch Logs
	svc := c.sessionBuilder()

	// Get events immediately when operator starts then use poll_interval duration.
	c.getEvents(ctx, svc)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(c.pollInterval.Duration):
			c.getEvents(ctx, svc)
		}
	}
}

// sessionBuilder builds a session for AWS Cloudwatch Logs
func (c *CloudwatchInput) sessionBuilder() *cloudwatchlogs.CloudWatchLogs {

	region := aws.String(c.region)
	var sess *session.Session
	if c.profile != "" {
		sess = session.Must(session.NewSessionWithOptions(session.Options{
			Config: aws.Config{Region: region},

			Profile: c.profile,
		}))
	}

	if c.profile == "" {
		sess = session.Must(session.NewSession(&aws.Config{
			Region: region,
		}))
	}

	return cloudwatchlogs.New(sess)
}

// getEvents uses a session to get events from AWS Cloudwatch Logs
func (c *CloudwatchInput) getEvents(ctx context.Context, svc *cloudwatchlogs.CloudWatchLogs) {
	nextToken := ""
	st, err := c.persist.Read(c.logGroupName)
	if err != nil {
		c.Errorf("failed to get persist: %s", err)
	}
	c.startTime = st
	if c.startAtEnd && c.startTime == 0 {
		c.startTime = currentTimeInUnixMilliseconds()
		c.Debugf("Setting start time to current time: %s", c.startTime)
	}
	c.Debugf("Getting events from AWS Cloudwatch Logs groupname '%s' using start time of %s", c.logGroupName, fromUnixMilli(c.startTime))
	for {
		input := c.filterLogEventsInputBuilder(nextToken)

		resp, err := svc.FilterLogEvents(&input)
		if err != nil {
			c.Errorf("failed to get events: %s", err)
			break
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
		c.Debug("Reached event limit '%d'", c.eventLimit)
		c.persist.DB.Sync()
	}
}

// filterLogEventsInputBuilder builds AWS Cloudwatch Logs Filter Log Events Input based on provided values
// and returns completed input.
func (c *CloudwatchInput) filterLogEventsInputBuilder(nextToken string) cloudwatchlogs.FilterLogEventsInput {
	logGroupNamePtr := aws.String(c.logGroupName)
	limit := aws.Int64(c.eventLimit)
	startTime := aws.Int64(c.startTime)

	if c.logStreamNamePrefix != "" && nextToken != "" {
		tmp := c.timeLayoutParser(c.logStreamNamePrefix)
		logStreamNamePrefixPtr := aws.String(tmp)
		nextTokenPtr := aws.String(nextToken)
		return cloudwatchlogs.FilterLogEventsInput{
			Limit:               limit,
			LogGroupName:        logGroupNamePtr,
			LogStreamNamePrefix: logStreamNamePrefixPtr,
			StartTime:           startTime,
			NextToken:           nextTokenPtr,
		}
	}

	if c.logStreamNamePrefix != "" {
		tmp := c.timeLayoutParser(c.logStreamNamePrefix)
		logStreamNamePrefixPtr := aws.String(tmp)
		return cloudwatchlogs.FilterLogEventsInput{
			Limit:               limit,
			LogGroupName:        logGroupNamePtr,
			LogStreamNamePrefix: logStreamNamePrefixPtr,
			StartTime:           startTime,
		}
	}

	if len(c.logStreamNames) > 0 && nextToken != "" {
		nextTokenPtr := aws.String(nextToken)
		return cloudwatchlogs.FilterLogEventsInput{
			Limit:          limit,
			LogGroupName:   logGroupNamePtr,
			LogStreamNames: c.logStreamNames,
			StartTime:      startTime,
			NextToken:      nextTokenPtr,
		}
	}

	if len(c.logStreamNames) > 0 {
		return cloudwatchlogs.FilterLogEventsInput{
			Limit:          limit,
			LogGroupName:   logGroupNamePtr,
			LogStreamNames: c.logStreamNames,
			StartTime:      startTime,
		}
	}

	if nextToken != "" {
		nextTokenPtr := aws.String(nextToken)
		return cloudwatchlogs.FilterLogEventsInput{
			Limit:        limit,
			LogGroupName: logGroupNamePtr,
			StartTime:    startTime,
			NextToken:    nextTokenPtr,
		}
	}

	return cloudwatchlogs.FilterLogEventsInput{
		Limit:        limit,
		LogGroupName: logGroupNamePtr,
		StartTime:    startTime,
	}
}

// handleEvent is the handler for a AWS Cloudwatch Logs Filtered Event.
func (c *CloudwatchInput) handleEvent(ctx context.Context, event *cloudwatchlogs.FilteredLogEvent) error {
	e := make(map[string]interface{})
	e["message"] = event.Message
	e["event_id"] = event.EventId
	e["log_stream_name"] = event.LogStreamName
	e["timestamp"] = event.Timestamp
	e["ingestion_time"] = event.IngestionTime

	entry, err := c.NewEntry(nil)
	if err != nil {
		return errors.Wrap(err, "Failed to create new entry from record")
	}

	entry.Timestamp = fromUnixMilli(*event.Timestamp)
	entry.Record = e
	// Persist
	if *event.IngestionTime > c.startTime {
		c.persist.Write(c.logGroupName, *event.IngestionTime)
	}
	// Write Entry
	c.Write(ctx, entry)
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

// Helper function to convert Unix epoch time in milliseconds to go time
func fromUnixMilli(ms int64) time.Time {
	const millisInSecond = 1000
	const nsInSecond = 1000000
	return time.Unix(ms/int64(millisInSecond), (ms%int64(millisInSecond))*int64(nsInSecond))
}

// timeLayoutParser parses set of predefined variables and replaces with date equivalent
func (c *CloudwatchInput) timeLayoutParser(layout string) string {
	if strings.Contains(layout, "%") {
		replace := map[string]string{
			"%Y": "2006",    // Year, zero-padded
			"%y": "06",      // Year, last two digits, zero-padded
			"%m": "01",      // Month as a decimal number
			"%q": "1",       // Month as a unpadded number
			"%b": "Jan",     // Abbreviated month name
			"%h": "Jan",     // Abbreviated month name
			"%B": "January", // Full month name
			"%d": "02",      // Day of the month, zero-padded
			"%g": "2",       // Day of the month, unpadded
			"%a": "Mon",     // Abbreviated weekday name
			"%A": "Monday",  // Full weekday name
		}

		for key, value := range replace {
			layout = strings.Replace(layout, key, value, 1)
		}
		return time.Now().Format(layout)
	}
	return layout
}
