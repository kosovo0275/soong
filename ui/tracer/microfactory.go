package tracer

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

func (t *tracerImpl) ImportMicrofactoryLog(filename string) {
	if _, err := os.Stat(filename); err != nil {
		return
	}

	f, err := os.Open(filename)
	if err != nil {
		t.log.Verboseln("Error opening microfactory trace:", err)
		return
	}
	defer f.Close()

	entries := []*eventEntry{}
	begin := map[string][]uint64{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		fields := strings.SplitN(s.Text(), " ", 3)
		if len(fields) != 3 {
			t.log.Verboseln("Unknown line in microfactory trace:", s.Text())
			continue
		}
		timestamp, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			t.log.Verboseln("Failed to parse timestamp in microfactory trace:", err)
		}

		if fields[1] == "B" {
			begin[fields[2]] = append(begin[fields[2]], timestamp)
		} else if beginTimestamps, ok := begin[fields[2]]; ok {
			entries = append(entries, &eventEntry{
				Name:  fields[2],
				Begin: beginTimestamps[len(beginTimestamps)-1],
				End:   timestamp,
			})
			begin[fields[2]] = beginTimestamps[:len(beginTimestamps)-1]
		}
	}

	t.importEvents(entries)
}
