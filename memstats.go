package memstats

import (
	"bytes"
	"fmt"
	"io"
	"runtime"
	"time"

	"github.com/dustin/go-humanize"
)

// Stats represents the delta between two runtime.MemStats taken at two points in time.
type Diff struct {
	Delta time.Duration

	// General statistics
	Alloc      uint64 // bytes allocated and still in use
	TotalAlloc uint64 // bytes allocated (even if freed)
	Mallocs    uint64 // number of mallocs
	Frees      uint64 // number of frees

	// Main allocation heap statistics.
	HeapAlloc   uint64 // bytes allocated and still in use
	HeapSys     uint64 // bytes obtained from system
	HeapObjects uint64 // total number of allocated objects

	// Garbage collector statistics.
	PauseTotal    time.Duration
	Pause         []time.Duration
	PausesMissing bool
	NumGC         uint32
}

type Stats struct {
	ms     [2]runtime.MemStats
	t      [2]time.Time
	i      int // 0 or 1; next one to fill
	filled bool
}

func (s *Stats) Collect() {
	s.t[s.i] = time.Now()
	runtime.ReadMemStats(&s.ms[s.i])
	s.i = 1 - s.i
	if s.i == 0 {
		s.filled = true
	}
}

func (s *Stats) ReadDiff(diff *Diff) (ok bool) {
	if !s.filled {
		return false
	}

	i1, i2 := s.i, 1-s.i
	prev, cur := s.ms[i1], s.ms[i2]

	diff.Delta = s.t[i2].Sub(s.t[i1])
	if diff.Delta < 0 {
		return false
	}

	diff.Alloc = cur.Alloc
	diff.TotalAlloc = cur.TotalAlloc - prev.TotalAlloc
	diff.Mallocs = cur.Mallocs - prev.Mallocs
	diff.Frees = cur.Frees - prev.Frees

	diff.HeapAlloc = cur.HeapAlloc
	diff.HeapSys = cur.HeapSys - prev.HeapSys
	diff.HeapObjects = cur.HeapObjects

	diff.PauseTotal = time.Duration(cur.PauseTotalNs - prev.PauseTotalNs)
	diff.NumGC = cur.NumGC - prev.NumGC
	diff.PausesMissing = false
	if diff.Pause == nil {
		diff.Pause = make([]time.Duration, len(cur.PauseNs))
	}
	n := int(diff.NumGC)
	if n > len(cur.PauseNs) {
		diff.PausesMissing = true
		n = len(cur.PauseNs)
	}
	diff.Pause = diff.Pause[:n]
	for i := range diff.Pause {
		j := (int(cur.NumGC) - i + 255) % 256
		diff.Pause[i] = time.Duration(cur.PauseNs[j])
	}

	return true
}

func (d *Diff) String() string {
	var buf bytes.Buffer
	secs := d.Delta.Seconds()
	writef(&buf, "Allocated", sbytes(d.TotalAlloc))
	writef(&buf, "Alloc / sec", sbytes(float64(d.TotalAlloc)/secs))
	writef(&buf, "Alloc (still in use)", sbytes(d.Alloc))
	writef(&buf, "Heap bytes from system", sbytes(d.HeapSys))
	writef(&buf, "Heap alloc (still in use)", sbytes(d.HeapAlloc))
	writef(&buf, "Heap alloc objects", fmt.Sprintf("%d", d.HeapObjects))
	writef(&buf, "Heap alloc objects / sec", fmt.Sprintf("%.2f", float64(d.HeapObjects)/secs))
	writef(&buf, "Number of GCs", fmt.Sprintf("%d", d.NumGC))
	if d.NumGC > 0 {
		writef(&buf, "Pause time", d.PauseTotal.String())
		if d.NumGC > 1 {
			writef(&buf, "Mean pause time", (d.PauseTotal / time.Duration(d.NumGC)).String())
			writef(&buf, "Pause times (most recent first)", fmt.Sprintf("%v", d.Pause))
		}
	}
	return buf.String()
}

func writef(w io.Writer, field, value string) {
	fmt.Fprintf(w, "%-35s%s\n", field+":", value)
}

func sbytes(v interface{}) string {
	switch n := v.(type) {
	case float64:
		return humanize.Bytes(uint64(n))
	case uint64:
		return humanize.Bytes(n)
	}
	panic("bad type")
}
