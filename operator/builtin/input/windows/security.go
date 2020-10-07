package windows

import (
	"strings"
)

func parseSecurity(message string) (string, map[string]interface{}) {

	subject, details := message, map[string]interface{}{}

	mp := newMessageProcessor(message)

	// First line is expected to be the first return value
	l := mp.next()
	switch l.t {
	case valueType:
		subject = l.v
	case keyType:
		subject = l.k
	default:
		return message, nil
	}

	moreInfo := []string{}
	unparsed := []string{}

	for mp.hasNext() {
		l = mp.next()
		switch l.t {
		case valueType:
			moreInfo = append(moreInfo, l.v)
		case keyType:
			if !mp.hasNext() {
				// line was standalone key/value pair with an empty value
				details[l.k] = "-"
				continue
			}

			if ln := mp.peek(); ln.t == emptyType || l.i == ln.i {
				// line was standalone key/value pair with an empty value
				details[l.k] = "-"
				continue
			}

			// process indented subsection
			sub := map[string]interface{}{}
		CONSUME_SUBSECTION:
			for mp.hasNext() {
				ln := mp.next()
				switch ln.t {
				case emptyType:
					break CONSUME_SUBSECTION
				case pairType:
					sub[ln.k] = ln.v
				case keyType:
					if !mp.hasNext() {
						// line was standalone key/value pair with an empty value
						sub[ln.k] = "-"
						continue
					}

					if lnn := mp.peek(); lnn.t == emptyType || ln.i == lnn.i {
						// line was standalone key/value pair with an empty value
						sub[ln.k] = "-"
						continue
					}

					// process indented subsection as list
					sub[ln.k] = mp.consumeSublist(ln.i)
				}
			}
			details[l.k] = sub
		case pairType:
			if !mp.hasNext() {
				details[l.k] = l.v
				continue
			}
			ln := mp.peek()
			switch ln.t {
			case emptyType:
				// first line was standalone key/value pair
				details[l.k] = l.v
			case pairType:
				// first line was standalone key/value pair
				details[l.k] = l.v
			case valueType:
				// first line was key and first value of list
				list := []string{l.v}
				for mp.hasNext() && mp.peek().t == valueType {
					ln = mp.next()
					list = append(list, ln.v)
				}
				details[l.k] = list
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

func (mp *messageProcessor) consumeSublist(baseDepth int) []string {
	sublist := []string{}
	for mp.hasNext() {
		if l := mp.peek(); l.t == emptyType || l.i == baseDepth {
			// subsection has ended
			return sublist
		}
		l := mp.next()
		switch l.t {
		case valueType:
			sublist = append(sublist, l.v)
		case keyType: // not expected, but handle
			sublist = append(sublist, l.k)
		}
	}
	return sublist
}

type messageProcessor struct {
	lines []*parsedLine
	ptr   int
}

type parsedLine struct {
	t lineType
	i int
	k string
	v string
}

type lineType int

const (
	emptyType lineType = iota
	keyType
	valueType
	pairType
)

func newMessageProcessor(message string) *messageProcessor {
	unparsedLines := strings.Split(strings.TrimSpace(message), "\n")
	parsedLines := make([]*parsedLine, len(unparsedLines))
	for i, unparsedLine := range unparsedLines {
		parsedLines[i] = parse(unparsedLine)
	}
	return &messageProcessor{lines: parsedLines}
}

func parse(line string) *parsedLine {
	i := countIndent(line)
	l := strings.TrimSpace(line)
	if l == "" {
		return &parsedLine{t: emptyType, i: i}
	}

	if strings.Contains(l, ":\t") {
		k, v := parseKeyValue(l)
		return &parsedLine{t: pairType, i: i, k: k, v: v}
	}

	if strings.HasSuffix(l, ":") {
		return &parsedLine{t: keyType, i: i, k: l[:len(l)-1]}
	}

	return &parsedLine{t: valueType, i: i, v: l}
}

// return next line and increment position
func (mp *messageProcessor) next() *parsedLine {
	defer mp.step()
	return mp.lines[mp.ptr]
}

// return next line but do not increment position
func (mp *messageProcessor) peek() *parsedLine {
	return mp.lines[mp.ptr]
}

// just increment position
func (mp *messageProcessor) step() {
	mp.ptr++
}

func (mp *messageProcessor) hasNext() bool {
	return mp.ptr < len(mp.lines)
}

func countIndent(line string) int {
	i := 1
	for pre := strings.Repeat("\t", i); strings.HasPrefix(line, pre); pre = strings.Repeat("\t", i) {
		i++
	}
	return i - 1
}

func parseKeyValue(line string) (string, string) {
	kv := strings.SplitN(line, ":\t", 2)
	return strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
}
