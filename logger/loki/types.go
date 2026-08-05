// Package loki provides a batched Loki push client and a Zap core adapter.
package loki

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Labels is the label set that identifies a Loki stream.
// Values must be low-cardinality. In particular, request IDs, paths, IPs and
// user IDs belong in a log line, not in Labels.
type Labels map[string]string

func (l Labels) canonical() (string, Labels) {
	keys := make([]string, 0, len(l))
	copy := make(Labels, len(l))
	for key, value := range l {
		keys = append(keys, key)
		copy[key] = value
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, key := range keys {
		// A separator that cannot occur in a valid Loki label name keeps the
		// batching key unambiguous even when values contain punctuation.
		fmt.Fprintf(&b, "%s\x00%s\x00", key, copy[key])
	}
	return b.String(), copy
}

type record struct {
	labels Labels
	time   time.Time
	line   []byte
}

type stream struct {
	Stream Labels      `json:"stream"`
	Values [][2]string `json:"values"`
}

type pushPayload struct {
	Streams []stream `json:"streams"`
}

// Stats is a snapshot of client-side delivery counters.
type Stats struct {
	Dropped       uint64
	Sent          uint64
	FailedBatches uint64
}
