package file

import (
	"bufio"
	"regexp"
)

// NewLineStartSplitFunc creates a bufio.SplitFunc that splits an incoming stream into
// tokens that start with a match to the regex pattern provided
func NewLineStartSplitFunc(re *regexp.Regexp) bufio.SplitFunc {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		firstLoc := re.FindIndex(data)
		if firstLoc == nil {
			return 0, nil, nil // read more data and try again.
		}
		firstMatchStart := firstLoc[0]
		firstMatchEnd := firstLoc[1]

		if firstMatchStart != 0 {
			// the beginning of the file does not match the start pattern, so return a token up to the first match so we don't lose data
			advance = firstMatchStart
			token = data[0:firstMatchStart]
			return
		}

		if firstMatchEnd == len(data) {
			// the first match goes to the end of the buffer, so don't look for a second match
			if atEOF {
				return len(data), data[firstMatchStart:], nil // return the rest of the file and advance to end
			}
			return 0, nil, nil // read more data and try again
		}

		secondLocOffset := firstMatchEnd + 1
		secondLoc := re.FindIndex(data[secondLocOffset:])
		if secondLoc == nil {
			if atEOF {
				return len(data), data[firstMatchStart:], nil // return the rest of the file and advance to end
			}
			return 0, nil, nil // read more data and try again
		}
		secondMatchStart := secondLoc[0] + secondLocOffset

		advance = secondMatchStart                     // start scanning at the beginning of the second match
		token = data[firstMatchStart:secondMatchStart] // the token begins at the first match, and ends at the beginning of the second match
		err = nil
		return
	}
}

// NewLineEndSplitFunc creates a bufio.SplitFunc that splits an incoming stream into
// tokens that end with a match to the regex pattern provided
func NewLineEndSplitFunc(re *regexp.Regexp) bufio.SplitFunc {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		loc := re.FindIndex(data)
		if loc == nil {
			return 0, nil, nil // read more data and try again
		}

		// If the match goes up to the end of the current buffer, do another
		// read until we can capture the entire match
		if loc[1] == len(data)-1 && !atEOF {
			return 0, nil, nil
		}

		advance = loc[1]
		token = data[:loc[1]]
		err = nil
		return
	}
}
