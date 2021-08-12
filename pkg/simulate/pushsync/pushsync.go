package pushsync

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/random"
	proxyClient "github.com/ethersphere/ethproxy/pkg/api/client"
	"github.com/prometheus/client_golang/prometheus/push"
)

// Options represents simulation options
type Options struct {
	GasPrice         string
	MetricsPusher    *push.Pusher
	PostageAmount    int64
	PostageDepth     uint64
	PostageLabel     string
	DownloadWait     time.Duration
	UploadWait       time.Duration
	DownloadCount    int64
	DownloadRetry    int64
	Seed             int64
	ProxyApiEndpoint string
	ChunkCount       int64
	StartPercentage  float64
	EndPercentage    float64
	StepPercentage   float64
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		GasPrice:         "",
		MetricsPusher:    nil,
		PostageAmount:    1000,
		PostageDepth:     20,
		PostageLabel:     "test-label",
		UploadWait:       30 * time.Second,
		DownloadWait:     15 * time.Second,
		DownloadRetry:    3,
		DownloadCount:    3,
		Seed:             0,
		ProxyApiEndpoint: "http://ethproxy.localhost",
		ChunkCount:       500,
		StartPercentage:  0.0,
		EndPercentage:    0.6,
		StepPercentage:   0.1,
	}
}

// compile simulation whether Upload implements interface
var _ beekeeper.Action = (*Simulation)(nil)

// Simulation instance
type Simulation struct{}

// NewSimulation returns new upload simulation
func NewSimulation() beekeeper.Action {
	return &Simulation{}
}

// Run executes upload stress
func (s *Simulation) Run(ctx context.Context, cluster *bee.Cluster, opts interface{}) (err error) {

	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	metrics := newMetrics(cluster.Name()+"-"+time.Now().UTC().Format("2006-01-02-15-04-05-000000000"), o.MetricsPusher)

	rnd := random.PseudoGenerator(o.Seed)

	// begin recording block number
	proxy := proxyClient.NewClient(o.ProxyApiEndpoint)
	cancelID, err := proxy.Execute(proxyClient.BlockNumberRecord)
	defer func(cancelID int) {
		_ = proxy.Cancel(cancelID)
	}(cancelID)

	if err != nil {
		return fmt.Errorf("proxy execute %w", err)
	}
	time.Sleep(time.Second * 5)

	// set up clients
	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return fmt.Errorf("node clients: %w", err)
	}

	if len(clients) <= 2 {
		return errors.New("not enough nodes")
	}

	names := shuffle(rnd, cluster.NodeNames(), o.Seed)

	buckets, leftovers := ToBuckets(names, o.StartPercentage, o.EndPercentage, o.StepPercentage)

	malfunctionEth := 0

	for _, bucket := range buckets {

		// STEP: freeze nodes in bucket
		for _, n := range bucket {
			node := clients[n]

			nodeAddr, err := node.Addresses(ctx)
			if err != nil {
				return err
			}

			if len(nodeAddr.Underlay) == 0 {
				return errors.New("not enough underlay addresses")
			}

			nodeIP := GetIPFromUnderlays(nodeAddr.Underlay)

			fmt.Printf("freezing block number for node %s ip %s\n", n, nodeIP)

			cancelID, err := proxy.Execute(proxyClient.BlockNumberFreeze, nodeIP)
			if err != nil {
				return err
			}

			defer func(cancelID int) {
				_ = proxy.Cancel(cancelID)
			}(cancelID)
		}

		malfunctionEth += len(bucket)

		// Upload nodes are not in the list of malfunctioning nodes
		// so upload errors should not be expected

		chunks := chunkBatch(rnd, int(o.ChunkCount))

		index := rnd.Intn(len(leftovers))
		uploadName := leftovers[index]
		uploadNode := clients[uploadName]
		fmt.Printf("using node %s as uploader\n", uploadName)

		uploaded, err := uploadChunks(ctx, rnd, o, uploadNode, chunks)
		if err != nil {
			return fmt.Errorf("upload: %w", err)
		}

		metricStr := fmt.Sprintf("%d_%d_malfunctioning_backends", malfunctionEth, len(names))
		metrics.uploadedChunks.WithLabelValues(metricStr).Add(float64(uploaded))

		var downloaded int
		for i := 0; i < int(o.DownloadCount); i++ {
			downloadName := randomCmp(rnd, uploadName, names)
			downloadNode := clients[downloadName]

			fmt.Printf("using node %s as downloader\n", downloadName)

			downloaded = downloadChunks(ctx, o, uploaded, downloadNode, chunks)

			fmt.Printf("%d out of %d_malfunctioning backends\n", malfunctionEth, len(names))
			fmt.Printf("uploaded to %s %d chunks\n", uploadName, uploaded)
			fmt.Printf("downloaded from %s %d chunks\n", downloadName, downloaded)

			metrics.downloadCount.WithLabelValues(metricStr).Inc()

			if downloaded == uploaded {
				break
			}

			time.Sleep(o.DownloadWait)
		}

		metrics.downloadedChunks.WithLabelValues(metricStr).Add(float64(downloaded))
	}

	return nil
}

func uploadChunks(ctx context.Context, rnd *rand.Rand, o Options, client *bee.Client, chunks []swarm.Chunk) (int, error) {

	batchID, err := client.CreatePostageBatch(ctx, o.PostageAmount, o.PostageDepth, o.GasPrice, o.PostageLabel, false)
	if err != nil {
		return 0, fmt.Errorf("batch create %w", err)
	}

	count := 0
	for _, chunk := range chunks {
		_, err := client.UploadChunk(ctx, chunk.Data(), api.UploadOptions{BatchID: batchID})
		if err == nil {
			count++
		}
	}

	// wait for uploader to sync to network
	time.Sleep(o.UploadWait)

	return count, nil
}

func downloadChunks(ctx context.Context, o Options, uploadCount int, client *bee.Client, chunks []swarm.Chunk) int {

	var count int

	for i := 0; i < int(o.DownloadRetry); i++ {
		count := 0
		for _, chunk := range chunks {
			_, err := client.DownloadChunk(ctx, chunk.Address(), "")
			if err == nil {
				count++
			}
		}

		if count == uploadCount {
			return count
		}
		time.Sleep(o.DownloadWait)
	}

	return count
}

func randomCmp(rnd *rand.Rand, cmp string, names []string) string {

	var str string

	for {
		index := rnd.Intn(len(names))
		str = names[index]
		if str != cmp {
			break
		}
	}

	return str
}

func shuffle(rnd *rand.Rand, names []string, seed int64) []string {
	rnd.Shuffle(len(names), func(i, j int) {
		names[i], names[j] = names[j], names[i]
	})
	return names
}

// toBuckets splits arr into buckets where the first bucket is 0-th index upto start percentage number of elements,
// subsequent buckets are step percentage number of elements until the end percentage is reached,
// leftover is the elements from stop percentage up to the last element,
// ex: arr = [1,2,3,4,5,6,7,8,9, 10], start=0.4, end=0.8, step=0.2,
// returned is [[1,2,3,4], [5,6], [7,8]], [9,10]
func ToBuckets(arr []string, start float64, end float64, step float64) ([][]string, []string) {

	var ret [][]string

	stepCount := int(float64(len(arr)) * step)
	startCount := int(float64(len(arr)) * start)
	endCount := int(float64(len(arr)) * end)

	ret = append(ret, arr[0:startCount])

	for startCount < endCount {
		ret = append(ret, arr[startCount:startCount+stepCount])
		startCount += stepCount
	}

	return ret, arr[startCount:]
}

func GetIPFromUnderlays(addrs []string) string {

	underlayRegex, _ := regexp.Compile(`(\/ip4\/)([0-9]+[.][0-9]+[.][0-9]+[.][0-9]+)`)

	for _, addr := range addrs {
		m := underlayRegex.FindStringSubmatch(addr)

		if len(m) > 0 {
			ip := m[len(m)-1]
			if !strings.HasPrefix(ip, "127") {
				return ip
			}
		}
	}

	return ""
}

func chunkBatch(rnd *rand.Rand, count int) []swarm.Chunk {
	chunks := make([]swarm.Chunk, count)
	for i := range chunks {
		chunks[i] = bee.NewRandSwarmChunk(rnd)
	}
	return chunks
}
