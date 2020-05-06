package check

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"log"
	"net/url"

	"github.com/ethersphere/beekeeper/pkg/bee/debugapi"
)

const (
	maxChunkSize = 4096
	scheme       = "http"
)

// node represents Bee node
type node struct {
	Addresses debugapi.Addresses
	Peers     debugapi.Peers
}

// contains checks if slice of strings containes given string
func contains(s []string, v string) bool {
	for _, a := range s {
		if a == v {
			return true
		}
	}
	return false
}

// createURL creates API or debug API URL
func createURL(scheme, hostnamePattern, namespace, domain string, counter int, disableNamespace bool) (nodeURL *url.URL, err error) {
	hostname := fmt.Sprintf(hostnamePattern, counter)
	if disableNamespace {
		nodeURL, err = url.Parse(fmt.Sprintf("%s://%s.%s", scheme, hostname, domain))
	} else {
		nodeURL, err = url.Parse(fmt.Sprintf("%s://%s.%s.%s", scheme, hostname, namespace, domain))
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
