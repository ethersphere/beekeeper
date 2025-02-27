package gsoc

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/ethersphere/bee/v2/pkg/cac"
	"github.com/ethersphere/bee/v2/pkg/crypto"
	"github.com/ethersphere/bee/v2/pkg/soc"
	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/ethersphere/beekeeper/pkg/wslistener"
	"golang.org/x/sync/errgroup"
)

// Options represents check options
type Options struct {
	PostageTTL   time.Duration
	PostageDepth uint64
	PostageLabel string
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		PostageTTL:   24 * time.Hour,
		PostageDepth: 17,
		PostageLabel: "test-label",
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance.
type Check struct {
	logger logging.Logger
}

type socData struct {
	Owner string
	Sig   string
	Data  []byte
}

// NewCheck returns a new check instance.
func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{
		logger: logger,
	}
}

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	rnd := random.PseudoGenerator(time.Now().UnixNano())
	fullNodeClients, err := cluster.ShuffledFullNodeClients(ctx, rnd)
	if err != nil {
		return err
	}

	if len(fullNodeClients) < 2 {
		return fmt.Errorf("gsoc test require at least 2 full nodes")
	}

	uploadClient := fullNodeClients[0]
	listenClient, err := cluster.ClosestFullNodeClient(ctx, uploadClient)
	if err != nil {
		return err
	}

	batches := make([]string, 2)
	for i := 0; i < 2; i++ {
		c.logger.Infof("gsoc: creating postage batch. duration=%d, depth=%d, label=%s", o.PostageTTL, o.PostageDepth, o.PostageLabel)
		batchID, err := uploadClient.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
		if err != nil {
			return err
		}
		c.logger.Infof("gsoc: postage batch created: %s", batchID)
		batches[i] = batchID
	}

	c.logger.Infof("send messages with different postage batches sequentially...")
	err = run(ctx, uploadClient, listenClient, batches, c.logger, false)
	if err != nil {
		return fmt.Errorf("sequential: %w", err)
	}
	c.logger.Infof("done")

	c.logger.Infof("send messages with different postage batches parallel...")
	err = run(ctx, uploadClient, listenClient, batches, c.logger, true)
	if err != nil {
		return fmt.Errorf("parallel: %w", err)
	}
	c.logger.Infof("done")

	return nil
}

func run(ctx context.Context, uploadClient *bee.Client, listenClient *bee.Client, batches []string, logger logging.Logger, parallel bool) error {
	const numChunks = 10
	privKey, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		return err
	}

	addresses, err := listenClient.Addresses(ctx)
	if err != nil {
		return err
	}
	resourceId, socAddress, err := mineResourceId(ctx, addresses.Overlay, privKey, 1)
	if err != nil {
		return err
	}
	logger.Infof("gsoc: socAddress=%s, listener node address=%s, node=%s", socAddress, addresses.Overlay, listenClient.Name())

	ctx, cancel := context.WithCancel(ctx)
	ch, closer, err := wslistener.ListenWebSocket(ctx, listenClient, fmt.Sprintf("/gsoc/subscribe/%s", socAddress), logger)
	if err != nil {
		cancel()
		return fmt.Errorf("listen websocket: %w", err)
	}

	defer func() {
		cancel()
		closer()
	}()

	received := make(map[string]bool, numChunks)
	receivedMtx := new(sync.Mutex)
	done := make(chan struct{})

	go func() {
		for p := range ch {
			receivedMtx.Lock()
			if !received[p] {
				logger.Infof("gsoc: received message %s on node %s", p, listenClient.Name())
			}
			received[p] = true
			if len(received) == numChunks {
				close(done)
				receivedMtx.Unlock()
				return
			}
			receivedMtx.Unlock()
		}
	}()

	if parallel {
		err = runInParallel(ctx, uploadClient, numChunks, batches, resourceId, privKey, logger)
	} else {
		err = runInSequence(ctx, uploadClient, numChunks, batches, resourceId, privKey, logger)
	}
	if err != nil {
		return err
	}

	select {
	case <-time.After(1 * time.Minute):
		return fmt.Errorf("timeout: not all messages received")
	case <-done:
	}

	receivedMtx.Lock()
	defer receivedMtx.Unlock()
	for i := 0; i < numChunks; i++ {
		want := fmt.Sprintf("data %d", i)
		if !received[want] {
			return fmt.Errorf("message '%s' not received", want)
		}
	}
	return nil
}

func uploadSoc(ctx context.Context, client *bee.Client, payload string, resourceId []byte, batchID string, privKey *ecdsa.PrivateKey) error {
	d, err := makeSoc(payload, resourceId, privKey)
	if err != nil {
		return fmt.Errorf("make soc: %w", err)
	}
	_, err = client.UploadSOC(ctx, d.Owner, hex.EncodeToString(resourceId), d.Sig, d.Data, batchID)
	if err != nil {
		return fmt.Errorf("upload soc: %w", err)
	}
	return nil
}

func runInSequence(ctx context.Context, client *bee.Client, numChunks int, batches []string, resourceId []byte, privKey *ecdsa.PrivateKey, logger logging.Logger) error {
	for i := 0; i < numChunks; i++ {
		payload := fmt.Sprintf("data %d", i)
		logger.Infof("gsoc: submitting soc to node=%s, payload=%s", client.Name(), payload)
		err := uploadSoc(ctx, client, payload, resourceId, batches[i%2], privKey)
		if err != nil {
			return err
		}
	}
	return nil
}

func runInParallel(ctx context.Context, client *bee.Client, numChunks int, batches []string, resourceId []byte, privKey *ecdsa.PrivateKey, logger logging.Logger) error {
	var errG errgroup.Group
	for i := 0; i < numChunks; i++ {
		errG.Go(func() error {
			payload := fmt.Sprintf("data %d", i)
			logger.Infof("gsoc: submitting soc to node=%s, payload=%s", client.Name(), payload)
			return uploadSoc(ctx, client, payload, resourceId, batches[i%2], privKey)
		})
	}
	return errG.Wait()
}

func getTargetNeighborhood(address swarm.Address, depth int) (string, error) {
	var targetNeighborhood string
	for i := 0; i < depth; i++ {
		hexChar := address.String()[i : i+1]
		value, err := strconv.ParseUint(hexChar, 16, 4)
		if err != nil {
			return "", err
		}
		targetNeighborhood += fmt.Sprintf("%04b", value)
	}
	return targetNeighborhood, nil
}

func mineResourceId(ctx context.Context, overlay swarm.Address, privKey *ecdsa.PrivateKey, depth int) ([]byte, swarm.Address, error) {
	targetNeighborhood, err := getTargetNeighborhood(overlay, depth)
	if err != nil {
		return nil, swarm.ZeroAddress, err
	}

	neighborhood, err := swarm.ParseBitStrAddress(targetNeighborhood)
	if err != nil {
		return nil, swarm.ZeroAddress, err
	}
	nonce := make([]byte, 32)
	prox := len(targetNeighborhood)
	owner, err := crypto.NewEthereumAddress(privKey.PublicKey)
	if err != nil {
		return nil, swarm.ZeroAddress, err
	}

	i := uint64(0)
	for {
		select {
		case <-ctx.Done():
			return nil, swarm.ZeroAddress, ctx.Err()
		default:
		}

		binary.LittleEndian.PutUint64(nonce, i)
		address, err := soc.CreateAddress(nonce, owner)
		if err != nil {
			return nil, swarm.ZeroAddress, err
		}

		if swarm.Proximity(address.Bytes(), neighborhood.Bytes()) >= uint8(prox) {
			return nonce, address, nil
		}
		i++
	}
}

func makeSoc(msg string, id []byte, privKey *ecdsa.PrivateKey) (*socData, error) {
	signer := crypto.NewDefaultSigner(privKey)

	ch, err := cac.New([]byte(msg))
	if err != nil {
		return nil, err
	}

	sch, err := soc.New(id, ch).Sign(signer)
	if err != nil {
		return nil, err
	}

	chunkData := sch.Data()
	signatureBytes := chunkData[swarm.HashSize : swarm.HashSize+swarm.SocSignatureSize]

	publicKey, err := signer.PublicKey()
	if err != nil {
		return nil, err
	}

	ownerBytes, err := crypto.NewEthereumAddress(*publicKey)
	if err != nil {
		return nil, err
	}

	return &socData{
		Owner: hex.EncodeToString(ownerBytes),
		Sig:   hex.EncodeToString(signatureBytes),
		Data:  ch.Data(),
	}, nil
}
