package loki

type Batch struct {
	Streams []Stream `json:"streams"`
}

func NewBatch() *Batch {
	return &Batch{[]Stream{}}
}

func (b *Batch) AddStream(s *Stream) {
	b.Streams = append(b.Streams, *s)
}
