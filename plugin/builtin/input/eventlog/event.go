package eventlog

// Event holds the data from a log record.
type Event struct {
	Computer        string          `xml:"System>Computer"`
	Channel         string          `xml:"System>Channel"`
	RecordID        uint64          `xml:"System>EventRecordID"`


	// Need a special struct for this
	TimeCreated     TimeCreated     `xml:"System>TimeCreated"`




	// RenderingInfo -- Keep
	Message  string   `xml:"RenderingInfo>Message"`
	Level    string   `xml:"RenderingInfo>Level"`
	Task     string   `xml:"RenderingInfo>Task"`
	Opcode   string   `xml:"RenderingInfo>Opcode"`
	Keywords []string `xml:"RenderingInfo>Keywords>Keyword"`
}


