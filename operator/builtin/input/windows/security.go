package windows

import (
	"strings"
)

func parseSecurity(message string) (string, map[string]interface{}) {

	mp := newMessageParser(message)

	t, _, subject := mp.parseNext()
	if t != valueType {
		return message, nil
	}

	details := map[string]interface{}{}
	moreInfo := []string{}
	unparsed := []string{}

	for t, k, v := mp.parseNext(); t != endType; t, k, v = mp.parseNext() {
		switch t {
		case endType:
			break
		case emptyType:
			continue
		case valueType:
			moreInfo = append(moreInfo, v)
		case keyType:
			// expect subsequent lines to be pairs until empty/end
			pairs := map[string]interface{}{}
		CONSUME_PAIRS:
			for {
				tn, kn, vn := mp.parseNext()
				switch tn {
				case endType, emptyType:
					break CONSUME_PAIRS
				case pairType:
					pairs[kn] = vn
				case keyType:
					pairs[kn] = "-" // sometimes the value is blank
				case valueType:
					pairs[vn] = "-" // unexpected, but handle anyways
				}
			}
			details[k] = pairs
		case pairType:
			tn, kn, vn := mp.parseNext()
			switch tn {
			case endType, emptyType:
				// first line was standalone key/value pair
				details[k] = v
			case pairType:
				// first line was standalone key/value pair and so is this one
				// presumably, next one will be same or empty, but allow outer to handle
				details[k] = v
				details[kn] = vn
			case keyType:
				details[k] = v
				details[kn] = "-" // unexpected, but handle anyways
			case valueType:
				// first line was key and first value of list
				list := []string{v, vn}
			CONSUME_LIST:
				for {
					tn, kn, vn = mp.parseNext()
					switch tn {
					case endType, emptyType:
						break CONSUME_LIST
					case valueType:
						list = append(list, vn)
					}
				}
				details[k] = list
			}
		}
	}

	if len(moreInfo) > 0 {
		details["Additional Context"] = moreInfo
	}

	if len(unparsed) > 0 {
		details["Unparsed"] = unparsed
	}

	return subject, details
}

type messageParser struct {
	lines []string
	next  int
}

type lineType int

const (
	emptyType lineType = iota
	keyType
	valueType
	pairType
	endType
)

func newMessageParser(message string) *messageParser {
	return &messageParser{lines: strings.Split(strings.TrimSpace(message), "\n")}
}

func (mp *messageParser) parseNext() (lineType, string, string) {
	if mp.next >= len(mp.lines) {
		return endType, "", ""
	}
	defer func() { mp.next++ }()

	line := strings.TrimSpace(mp.lines[mp.next])
	if line == "" {
		return emptyType, "", ""
	}
	if !strings.Contains(line, ":") {
		return valueType, "", line
	}
	k, v := parseKeyValue(line)
	if v == "" {
		return keyType, k, ""
	}
	return pairType, k, v
}

func parseKeyValue(line string) (string, string) {
	kv := strings.Split(line, ":")
	return strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
}
