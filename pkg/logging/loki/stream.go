package loki

import (
	"strconv"
	"time"
)

type Stream struct {
	Labels  map[string]string `json:"stream"`
	Entries [][]string        `json:"values"`
}

func NewStream() *Stream {
	return &Stream{
		map[string]string{},
		[][]string{},
	}
}

func (s *Stream) AddLabel(key string, value string) {
	s.Labels[key] = value
}

func (s *Stream) AddEntry(t time.Time, entry string) {
	timeStr := strconv.FormatInt(t.UnixNano(), 10)
	s.Entries = append(s.Entries, []string{timeStr, entry})
}
