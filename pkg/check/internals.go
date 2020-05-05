package check

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"log"
	"net/url"
)

const (
	maxChunkSize = 4096
	scheme       = "http"
)

func nodeURL(scheme, hostnamePattern, namespace, domain string, counter int) (nodeURL *url.URL, err error) {
	hostname := fmt.Sprintf(hostnamePattern, counter)
	if len(namespace) > 0 {
		nodeURL, err = url.Parse(fmt.Sprintf("%s://%s.%s.%s", scheme, hostname, namespace, domain))
	} else {
		nodeURL, err = url.Parse(fmt.Sprintf("%s://%s.%s", scheme, hostname, domain))
	}
	return
}

// cryptoSource is used to create truly random source
type cryptoSource struct{}

func (s cryptoSource) Seed(seed int64) {}

func (s cryptoSource) Int63() int64 {
	return int64(s.Uint64() & ^uint64(1<<63))
}

func (s cryptoSource) Uint64() (v uint64) {
	err := binary.Read(rand.Reader, binary.BigEndian, &v)
	if err != nil {
		log.Fatal(err)
	}
	return v
}
