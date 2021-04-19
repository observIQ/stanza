package eventhub

import (
	"testing"
	"time"

	azhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/observiq/stanza/entry"
	"github.com/stretchr/testify/require"
)

var testPartitionID int16 = 10
var testPartitionKey string = "1"
var testSequenceNum int64 = 600
var testTime time.Time = time.Now()
var testOffset int64 = 2000
var testString string = "a test string"

func TestParseEvent(t *testing.T) {
	cases := []struct {
		name           string
		inputRecord    *azhub.Event
		expectedRecord *entry.Entry
	}{
		{
			"timestamp-promotion",
			&azhub.Event{
				Data: []byte("event hub entry"),
				ID:   "000-555-666",
				SystemProperties: &azhub.SystemProperties{
					SequenceNumber: &testSequenceNum,
					EnqueuedTime:   &testTime,
					Offset:         &testOffset,
				},
			},
			&entry.Entry{
				Timestamp: testTime,
				Record: map[string]interface{}{
					"data": "event hub entry",
					"id":   "000-555-666",
					"system_properties": map[string]interface{}{
						"x-opt-sequence-number": &testSequenceNum,
						"x-opt-enqueued-time":   &testTime,
						"x-opt-offset":          &testOffset,
					},
				},
			},
		},
		{
			"full",
			&azhub.Event{
				Data:         []byte("hello world"),
				ID:           "1111",
				PartitionKey: &testPartitionKey,
				Properties: map[string]interface{}{
					"user": "stanza",
					"id":   1,
					"root": false,
				},
				SystemProperties: &azhub.SystemProperties{
					EnqueuedTime:               &testTime,
					SequenceNumber:             &testSequenceNum,
					Offset:                     &testOffset,
					PartitionID:                &testPartitionID,
					PartitionKey:               &testPartitionKey,
					IoTHubDeviceConnectionID:   &testString,
					IoTHubAuthGenerationID:     &testString,
					IoTHubConnectionAuthMethod: &testString,
					IoTHubConnectionModuleID:   &testString,
					IoTHubEnqueuedTime:         &testTime,
				},
			},
			&entry.Entry{
				Timestamp: testTime,
				Record: map[string]interface{}{
					"data":          "hello world",
					"id":            "1111",
					"partition_key": &testPartitionKey,
					"properties": map[string]interface{}{
						"user": "stanza",
						"id":   1,
						"root": false,
					},
					"system_properties": map[string]interface{}{
						"x-opt-sequence-number":                &testSequenceNum,
						"x-opt-enqueued-time":                  &testTime,
						"x-opt-offset":                         &testOffset,
						"x-opt-partition-id":                   &testPartitionID,
						"x-opt-partition-key":                  &testPartitionKey,
						"iothub-connection-device-id":          &testString,
						"iothub-connection-auth-generation-id": &testString,
						"iothub-connection-auth-method":        &testString,
						"iothub-connection-module-id":          &testString,
						"iothub-enqueuedtime":                  &testTime,
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e, err := parseEvent(tc.inputRecord)
			require.NoError(t, err)
			require.Equal(t, tc.expectedRecord, e)
		})
	}
}

func TestParse(t *testing.T) {
	cases := []struct {
		name           string
		inputRecord    *azhub.Event
		expectedRecord map[string]interface{}
	}{
		{
			"data",
			&azhub.Event{
				Data: []byte("hello world"),
			},
			map[string]interface{}{
				"data": "hello world",
				"id":   "",
			},
		},
		{
			"id",
			&azhub.Event{
				ID: "0000-1111",
			},
			map[string]interface{}{
				"data": "",
				"id":   "0000-1111",
			},
		},
		{
			"partition-key",
			&azhub.Event{
				Data:         []byte("hello world"),
				ID:           "1111",
				PartitionKey: &testPartitionKey,
			},
			map[string]interface{}{
				"data":          "hello world",
				"id":            "1111",
				"partition_key": &testPartitionKey,
			},
		},
		{
			"properties",
			&azhub.Event{
				Data: []byte("hello world"),
				ID:   "1111",
				Properties: map[string]interface{}{
					"user": "stanza",
					"id":   1,
				},
			},
			map[string]interface{}{
				"data": "hello world",
				"id":   "1111",
				"properties": map[string]interface{}{
					"user": "stanza",
					"id":   1,
				},
			},
		},
		{
			"system-properties-empty",
			&azhub.Event{
				Data:             []byte("hello world"),
				ID:               "1111",
				SystemProperties: &azhub.SystemProperties{},
			},
			map[string]interface{}{
				"data":              "hello world",
				"id":                "1111",
				"system_properties": map[string]interface{}{},
			},
		},
		{
			"system-properties",
			&azhub.Event{
				Data: []byte("hello world"),
				ID:   "1111",
				SystemProperties: &azhub.SystemProperties{
					SequenceNumber:             &testSequenceNum,
					EnqueuedTime:               &testTime,
					Offset:                     &testOffset,
					PartitionID:                &testPartitionID,
					PartitionKey:               &testPartitionKey,
					IoTHubDeviceConnectionID:   &testString,
					IoTHubAuthGenerationID:     &testString,
					IoTHubConnectionAuthMethod: &testString,
					IoTHubConnectionModuleID:   &testString,
					IoTHubEnqueuedTime:         &testTime,
				},
			},
			map[string]interface{}{
				"data": "hello world",
				"id":   "1111",
				"system_properties": map[string]interface{}{
					"x-opt-sequence-number":                &testSequenceNum,
					"x-opt-enqueued-time":                  &testTime,
					"x-opt-offset":                         &testOffset,
					"x-opt-partition-id":                   &testPartitionID,
					"x-opt-partition-key":                  &testPartitionKey,
					"iothub-connection-device-id":          &testString,
					"iothub-connection-auth-generation-id": &testString,
					"iothub-connection-auth-method":        &testString,
					"iothub-connection-module-id":          &testString,
					"iothub-enqueuedtime":                  &testTime,
				},
			},
		},
		{
			"full",
			&azhub.Event{
				Data:         []byte("hello world"),
				ID:           "1111",
				PartitionKey: &testPartitionKey,
				Properties: map[string]interface{}{
					"user": "stanza",
					"id":   1,
					"root": false,
				},
				SystemProperties: &azhub.SystemProperties{
					SequenceNumber:             &testSequenceNum,
					EnqueuedTime:               &testTime,
					Offset:                     &testOffset,
					PartitionID:                &testPartitionID,
					PartitionKey:               &testPartitionKey,
					IoTHubDeviceConnectionID:   &testString,
					IoTHubAuthGenerationID:     &testString,
					IoTHubConnectionAuthMethod: &testString,
					IoTHubConnectionModuleID:   &testString,
					IoTHubEnqueuedTime:         &testTime,
				},
			},
			map[string]interface{}{
				"data":          "hello world",
				"id":            "1111",
				"partition_key": &testPartitionKey,
				"properties": map[string]interface{}{
					"user": "stanza",
					"id":   1,
					"root": false,
				},
				"system_properties": map[string]interface{}{
					"x-opt-sequence-number":                &testSequenceNum,
					"x-opt-enqueued-time":                  &testTime,
					"x-opt-offset":                         &testOffset,
					"x-opt-partition-id":                   &testPartitionID,
					"x-opt-partition-key":                  &testPartitionKey,
					"iothub-connection-device-id":          &testString,
					"iothub-connection-auth-generation-id": &testString,
					"iothub-connection-auth-method":        &testString,
					"iothub-connection-module-id":          &testString,
					"iothub-enqueuedtime":                  &testTime,
				},
			},
		},
		{
			"empty-id-and-data",
			&azhub.Event{},
			map[string]interface{}{
				"data": "",
				"id":   "",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e, err := parse(tc.inputRecord)
			require.NoError(t, err)
			require.Equal(t, tc.expectedRecord, e)
		})
	}
}

func TestPromoteTime(t *testing.T) {
	enqueuedTime := time.Now()
	ioTHubEnqueuedTime := time.Now().Add(time.Hour * 10)

	cases := []struct {
		name           string
		inputRecord    *azhub.Event
		expectedRecord *entry.Entry
	}{
		{
			"enqueuedTime",
			&azhub.Event{
				Data: []byte("event hub entry"),
				ID:   "000-555-666",
				SystemProperties: &azhub.SystemProperties{
					EnqueuedTime: &enqueuedTime,
				},
			},
			&entry.Entry{
				Timestamp: enqueuedTime,
				Record: map[string]interface{}{
					"data": "event hub entry",
					"id":   "000-555-666",
					"system_properties": map[string]interface{}{
						"x-opt-enqueued-time": &enqueuedTime,
					},
				},
			},
		},
		{
			"ioTHubEnqueuedTime",
			&azhub.Event{
				Data: []byte("event hub entry"),
				ID:   "000-555-666",
				SystemProperties: &azhub.SystemProperties{
					IoTHubEnqueuedTime: &ioTHubEnqueuedTime,
				},
			},
			&entry.Entry{
				Timestamp: ioTHubEnqueuedTime,
				Record: map[string]interface{}{
					"data": "event hub entry",
					"id":   "000-555-666",
					"system_properties": map[string]interface{}{
						"iothub-enqueuedtime": &ioTHubEnqueuedTime,
					},
				},
			},
		},
		{
			"both-prefer-enqueuedTime",
			&azhub.Event{
				Data: []byte("event hub entry"),
				ID:   "000-555-666",
				SystemProperties: &azhub.SystemProperties{
					EnqueuedTime:       &enqueuedTime,
					IoTHubEnqueuedTime: &ioTHubEnqueuedTime,
				},
			},
			&entry.Entry{
				Timestamp: enqueuedTime,
				Record: map[string]interface{}{
					"data": "event hub entry",
					"id":   "000-555-666",
					"system_properties": map[string]interface{}{
						"x-opt-enqueued-time": &enqueuedTime,
						"iothub-enqueuedtime": &ioTHubEnqueuedTime,
					},
				},
			},
		},
		{
			"both-prefer-ioTHubEnqueuedTime",
			&azhub.Event{
				Data: []byte("event hub entry"),
				ID:   "000-555-666",
				SystemProperties: &azhub.SystemProperties{
					EnqueuedTime:       &time.Time{},
					IoTHubEnqueuedTime: &ioTHubEnqueuedTime,
				},
			},
			&entry.Entry{
				Timestamp: ioTHubEnqueuedTime,
				Record: map[string]interface{}{
					"data": "event hub entry",
					"id":   "000-555-666",
					"system_properties": map[string]interface{}{
						"x-opt-enqueued-time": &time.Time{},
						"iothub-enqueuedtime": &ioTHubEnqueuedTime,
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e, err := parseEvent(tc.inputRecord)
			require.NoError(t, err)
			require.Equal(t, tc.expectedRecord, e)
		})
	}
}
