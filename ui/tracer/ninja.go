package tracer

import (
	"bufio"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type eventEntry struct {
	Name  string
	Begin uint64
	End   uint64
}

func (t *tracerImpl) importEvents(entries []*eventEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Begin < entries[j].Begin
	})

	cpus := []uint64{}
	for _, entry := range entries {
		tid := -1
		for cpu, endTime := range cpus {
			if endTime <= entry.Begin {
				tid = cpu
				cpus[cpu] = entry.End
				break
			}
		}
		if tid == -1 {
			tid = len(cpus)
			cpus = append(cpus, entry.End)
		}

		t.writeEvent(&viewerEvent{
			Name:  entry.Name,
			Phase: "X",
			Time:  entry.Begin,
			Dur:   entry.End - entry.Begin,
			Pid:   1,
			Tid:   uint64(tid),
		})
	}
}

// ImportNinjaLog reads a .ninja_log file from ninja and writes the events out
// to the trace.
//
// startOffset is when the ninja process started, and is used to position the
// relative times from the ninja log into the trace. It's also used to skip
// reading the ninja log if nothing was run.
func (t *tracerImpl) ImportNinjaLog(thread Thread, filename string, startOffset time.Time) {
	t.Begin("ninja log import", thread)
	defer t.End(thread)

	if stat, err := os.Stat(filename); err != nil {
		t.log.Println("Missing ninja log:", err)
		return
	} else if stat.ModTime().Before(startOffset) {
		t.log.Verboseln("Ninja log not modified, not importing any entries.")
		return
	}

	f, err := os.Open(filename)
	if err != nil {
		t.log.Println("Error opening ninja log:", err)
		return
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	header := true
	entries := []*eventEntry{}
	prevEnd := 0
	offset := uint64(startOffset.UnixNano()) / 1000
	for s.Scan() {
		if header {
			hdr := s.Text()
			if hdr != "# ninja log v5" {
				t.log.Printf("Unknown ninja log header: %q", hdr)
				return
			}
			header = false
			continue
		}

		fields := strings.Split(s.Text(), "\t")
		begin, err := strconv.Atoi(fields[0])
		if err != nil {
			t.log.Printf("Unable to parse ninja entry %q: %v", s.Text(), err)
			return
		}
		end, err := strconv.Atoi(fields[1])
		if err != nil {
			t.log.Printf("Unable to parse ninja entry %q: %v", s.Text(), err)
			return
		}
		if end < prevEnd {
			entries = nil
		}
		prevEnd = end
		entries = append(entries, &eventEntry{
			Name:  fields[3],
			Begin: offset + uint64(begin)*1000,
			End:   offset + uint64(end)*1000,
		})
	}
	if err := s.Err(); err != nil {
		t.log.Println("Unable to parse ninja log:", err)
		return
	}

	t.importEvents(entries)
}
