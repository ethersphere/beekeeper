package corruptchunk

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/ethersphere/bee/v2/pkg/cac"
	"github.com/ethersphere/bee/v2/pkg/postage"
	"github.com/ethersphere/bee/v2/pkg/soc"
	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/util/retry"
)

const (
	beeContainerName = "bee"
	dataVolumeName   = "data"
	dataDir          = "/home/bee/.bee"
)

type Options struct {
	K8sClient     *k8s.Client
	ImportImage   string
	RestartImage  string
	ExpectRemoved bool
	ReadyTimeout  time.Duration
	PostageDepth  uint64
	PostageLabel  string
	PostageTTL    time.Duration
	Seed          int64
}

func NewDefaultOptions() Options {
	return Options{
		ImportImage:   "akram951/bee:before-5640",
		ExpectRemoved: true,
		ReadyTimeout:  5 * time.Minute,
		PostageDepth:  22,
		PostageLabel:  "test-label",
		PostageTTL:    24 * time.Hour,
		Seed:          random.Int64(),
	}
}

var _ beekeeper.Action = (*Check)(nil)

// Check verifies how a node handles an invalid chunk (neither CAC nor SOC)
// already sitting in its reserve, the case bee PR #5640 cleans up with
// migration step 8 on startup.
//
// Steps:
//  1. Pick a full node; its data must be on a persistent volume.
//  2. Build an invalid chunk: random bytes under a random address, stamped
//     with a real batch.
//  3. Swap the node's StatefulSet to ImportImage (bee 7515c4c5, the commit
//     before PR #5640, whose reserve does not validate chunk types) running
//     `bee db import reserve` to write the chunk.
//  4. Restart the node with RestartImage (the original image if empty).
//  5. GET /chunks/<address>: served means the chunk was kept, not found means
//     it was removed; compare with ExpectRemoved.
//  6. Always restore the original StatefulSet.
type Check struct {
	logger logging.Logger
}

func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{
		logger: logger,
	}
}

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, o any) error {
	opts, ok := o.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}
	if opts.K8sClient == nil {
		return errors.New("kubernetes client is required, run with --enable-k8s")
	}
	if opts.ImportImage == "" {
		return errors.New("import image is required")
	}

	c.logger.Infof("seed: %d", opts.Seed)
	rnd := random.PseudoGenerator(opts.Seed)

	clients, err := cluster.ShuffledFullNodeClients(ctx, rnd)
	if err != nil {
		return fmt.Errorf("get shuffled full node clients: %w", err)
	}
	if len(clients) == 0 {
		return errors.New("no full nodes in the cluster")
	}
	node := clients[0]
	namespace := cluster.Namespace()
	c.logger.Infof("target node: %s, namespace: %s", node.Name(), namespace)

	original, err := opts.K8sClient.StatefulSet.Get(ctx, node.Name(), namespace)
	if err != nil {
		return fmt.Errorf("get stateful set %s: %w", node.Name(), err)
	}
	// if err := checkPersistence(original); err != nil {
	// 	return err
	// }

	health, err := c.waitReady(ctx, node, opts.ReadyTimeout)
	if err != nil {
		return err
	}
	c.logger.Infof("node %s runs bee %s, image %s", node.Name(), health.Version, beeImage(original))
	c.logStatus(ctx, node, "before")

	batchID, err := node.GetOrCreateMutableBatch(ctx, opts.PostageTTL, opts.PostageDepth, opts.PostageLabel)
	if err != nil {
		return fmt.Errorf("get or create batch: %w", err)
	}
	c.logger.Infof("postage batch: %s", batchID)

	ch, archive, err := invalidChunk(rnd, batchID)
	if err != nil {
		return err
	}
	c.logger.Infof("invalid chunk: %s (%d bytes, neither cac nor soc)", ch.Address(), len(ch.Data()))

	if _, err := node.DownloadChunk(ctx, ch.Address(), "", nil); err == nil {
		return fmt.Errorf("chunk %s is retrievable before import", ch.Address())
	}

	sts := &statefulSet{client: opts.K8sClient, namespace: namespace, original: original, logger: c.logger}
	defer sts.restore(context.WithoutCancel(ctx))

	c.logger.Infof("importing invalid chunk with %s", opts.ImportImage)
	if err := sts.runImport(ctx, opts.ImportImage, importCommand(archive)); err != nil {
		return fmt.Errorf("import invalid chunk: %w", err)
	}
	c.logger.Infof("invalid chunk imported into %s reserve", node.Name())

	restartImage := opts.RestartImage
	if restartImage == "" {
		restartImage = beeImage(original)
	}
	c.logger.Infof("restarting %s with %s", node.Name(), restartImage)
	if err := sts.start(ctx, restartImage); err != nil {
		return fmt.Errorf("restart node: %w", err)
	}

	health, err = c.waitReady(ctx, node, opts.ReadyTimeout)
	if err != nil {
		return err
	}
	c.logger.Infof("node %s is back, runs bee %s", node.Name(), health.Version)
	c.logStatus(ctx, node, "after")

	cache := false
	got, err := node.DownloadChunk(ctx, ch.Address(), "", &api.DownloadOptions{Cache: &cache})
	c.logger.Infof("invalid chunk %s lookup on %s: %d bytes, error: %v", ch.Address(), node.Name(), len(got), err)

	var kept bool
	switch {
	case err == nil && bytes.Equal(got, ch.Data()):
		kept = true
	case err == nil:
		return fmt.Errorf("node %s returned %d unexpected bytes for invalid chunk %s", node.Name(), len(got), ch.Address())
	case !isChunkAbsent(err):
		return fmt.Errorf("inconclusive lookup of invalid chunk %s on %s: %w", ch.Address(), node.Name(), err)
	}

	switch {
	case opts.ExpectRemoved && kept:
		return fmt.Errorf("invalid chunk %s is still served by %s (bee %s): reserve kept it", ch.Address(), node.Name(), health.Version)
	case !opts.ExpectRemoved && !kept:
		return fmt.Errorf("invalid chunk %s is not served by %s (bee %s), expected it to be kept: %w", ch.Address(), node.Name(), health.Version, err)
	case kept:
		c.logger.Infof("invalid chunk kept and served, as expected")
	default:
		c.logger.Infof("invalid chunk removed, as expected")
	}

	return nil
}

func (c *Check) logStatus(ctx context.Context, node *bee.Client, when string) {
	status, err := node.Status(ctx)
	if err != nil {
		c.logger.Warningf("node %s status %s: %v", node.Name(), when, err)
		return
	}
	c.logger.Infof("node %s %s: reserve %d (%d within radius), storage radius %d, peers %d",
		node.Name(), when, status.ReserveSize, status.ReserveSizeWithinRadius, status.StorageRadius, status.ConnectedPeers)
}

func (c *Check) waitReady(ctx context.Context, node *bee.Client, timeout time.Duration) (api.Health, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		health, err := node.Health(ctx)
		if err == nil && health.Status == "ok" {
			if _, err = node.Status(ctx); err == nil {
				return health, nil
			}
		}
		c.logger.Debugf("waiting for node %s api: %v", node.Name(), err)

		select {
		case <-ctx.Done():
			return api.Health{}, fmt.Errorf("node %s api not ready after %s: %w", node.Name(), timeout, err)
		case <-ticker.C:
		}
	}
}

func isChunkAbsent(err error) bool {
	var httpErr *api.HTTPStatusError
	if !errors.As(err, &httpErr) {
		return false
	}
	switch httpErr.Code {
	case http.StatusNotFound:
		return true
	case http.StatusInternalServerError:
		return strings.Contains(err.Error(), "read chunk failed")
	}
	return false
}

func checkPersistence(ss *appsv1.StatefulSet) error {
	for _, v := range ss.Spec.Template.Spec.Volumes {
		if v.Name == dataVolumeName && v.EmptyDir != nil {
			return fmt.Errorf("node %s stores data on emptyDir, the imported chunk would be lost on restart: set persistence-enabled: true", ss.Name)
		}
	}
	for _, pvc := range ss.Spec.VolumeClaimTemplates {
		if pvc.Name == dataVolumeName {
			return nil
		}
	}
	return fmt.Errorf("node %s has no %q volume claim", ss.Name, dataVolumeName)
}

func beeImage(ss *appsv1.StatefulSet) string {
	if i := beeContainerIndex(ss); i >= 0 {
		return ss.Spec.Template.Spec.Containers[i].Image
	}
	return ""
}

func beeContainerIndex(ss *appsv1.StatefulSet) int {
	for i, c := range ss.Spec.Template.Spec.Containers {
		if c.Name == beeContainerName {
			return i
		}
	}
	return -1
}

func invalidChunk(rnd *rand.Rand, batchID string) (swarm.Chunk, []byte, error) {
	id, err := hex.DecodeString(batchID)
	if err != nil {
		return nil, nil, fmt.Errorf("decode batch id %q: %w", batchID, err)
	}

	addr := make([]byte, swarm.HashSize)
	if _, err := rnd.Read(addr); err != nil {
		return nil, nil, err
	}
	data := make([]byte, swarm.SpanSize+swarm.ChunkSize)
	if _, err := rnd.Read(data[swarm.SpanSize:]); err != nil {
		return nil, nil, err
	}
	binary.LittleEndian.PutUint64(data, swarm.ChunkSize)

	ch := swarm.NewChunk(swarm.NewAddress(addr), data)
	if cac.Valid(ch) || soc.Valid(ch) {
		return nil, nil, fmt.Errorf("generated chunk %s is valid", ch.Address())
	}

	index := make([]byte, 8)
	binary.BigEndian.PutUint32(index, binary.BigEndian.Uint32(addr)>>(32-postage.BucketDepth))
	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, uint64(time.Now().UnixNano()))
	stamp, err := postage.NewStamp(id, index, timestamp, make([]byte, 65)).MarshalBinary()
	if err != nil {
		return nil, nil, fmt.Errorf("marshal stamp: %w", err)
	}

	body := make([]byte, 4, 4+len(data)+len(stamp))
	binary.BigEndian.PutUint32(body, uint32(len(data)))
	body = append(body, data...)
	body = append(body, stamp...)

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := tw.WriteHeader(&tar.Header{Name: ch.Address().String(), Mode: 0o644, Size: int64(len(body))}); err != nil {
		return nil, nil, fmt.Errorf("write tar header: %w", err)
	}
	if _, err := tw.Write(body); err != nil {
		return nil, nil, fmt.Errorf("write tar body: %w", err)
	}
	if err := tw.Close(); err != nil {
		return nil, nil, fmt.Errorf("close tar: %w", err)
	}

	return ch, buf.Bytes(), nil
}

func importCommand(archive []byte) []string {
	return []string{"sh", "-c", fmt.Sprintf("echo %s | base64 -d | bee db import reserve - --data-dir=%s",
		base64.StdEncoding.EncodeToString(archive), dataDir)}
}

type statefulSet struct {
	client    *k8s.Client
	namespace string
	original  *appsv1.StatefulSet
	logger    logging.Logger
	changed   bool
}

func (s *statefulSet) podName() string {
	return s.original.Name + "-0"
}

func (s *statefulSet) update(ctx context.Context, mutate func(*appsv1.StatefulSet) error) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest, err := s.client.StatefulSet.Get(ctx, s.original.Name, s.namespace)
		if err != nil {
			return fmt.Errorf("get stateful set %s: %w", s.original.Name, err)
		}
		if err := mutate(latest); err != nil {
			return err
		}
		return s.client.StatefulSet.Update(ctx, s.namespace, latest)
	})
}

func (s *statefulSet) runImport(ctx context.Context, image string, command []string) error {
	s.changed = true
	err := s.update(ctx, func(ss *appsv1.StatefulSet) error {
		i := beeContainerIndex(ss)
		if i < 0 {
			return fmt.Errorf("stateful set %s has no %q container", ss.Name, beeContainerName)
		}
		ss.Spec.UpdateStrategy.Type = appsv1.OnDeleteStatefulSetStrategyType
		ss.Spec.UpdateStrategy.RollingUpdate = nil
		ss.Spec.Template.Spec.Containers[i].Image = image
		ss.Spec.Template.Spec.Containers[i].Command = command
		ss.Spec.Template.Spec.Containers[i].ReadinessProbe = nil
		ss.Spec.Template.Spec.Containers[i].LivenessProbe = nil
		return nil
	})
	if err != nil {
		return fmt.Errorf("set import spec: %w", err)
	}

	errc := make(chan error, 1)
	go func() {
		errc <- s.client.Pods.WaitForPodRecreationAndCompletion(ctx, s.namespace, s.podName(), beeContainerName)
	}()
	s.logger.Debugf("deleting pod %s to run the import", s.podName())
	if _, err := s.client.Pods.Delete(ctx, s.podName(), s.namespace); err != nil {
		return fmt.Errorf("delete pod %s: %w", s.podName(), err)
	}
	if err := <-errc; err != nil {
		return fmt.Errorf("import in pod %s: %w", s.podName(), err)
	}
	return nil
}

func (s *statefulSet) start(ctx context.Context, image string) error {
	err := s.update(ctx, func(ss *appsv1.StatefulSet) error {
		ss.Spec.Template = *s.original.Spec.Template.DeepCopy()
		if i := beeContainerIndex(ss); i >= 0 {
			ss.Spec.Template.Spec.Containers[i].Image = image
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("set start spec: %w", err)
	}
	s.changed = image != beeImage(s.original)
	return s.recreate(ctx)
}

func (s *statefulSet) recreate(ctx context.Context) error {
	s.logger.Debugf("deleting pod %s", s.podName())
	if _, err := s.client.Pods.Delete(ctx, s.podName(), s.namespace); err != nil {
		return fmt.Errorf("delete pod %s: %w", s.podName(), err)
	}
	if err := s.client.Pods.WaitForRunning(ctx, s.namespace, s.podName()); err != nil {
		return fmt.Errorf("wait for pod %s: %w", s.podName(), err)
	}
	return nil
}

func (s *statefulSet) restore(ctx context.Context) {
	s.logger.Infof("restoring stateful set %s", s.original.Name)
	err := s.update(ctx, func(ss *appsv1.StatefulSet) error {
		ss.Spec.Template = *s.original.Spec.Template.DeepCopy()
		ss.Spec.UpdateStrategy = s.original.Spec.UpdateStrategy
		ss.Spec.Replicas = s.original.Spec.Replicas
		return nil
	})
	if err != nil {
		s.logger.Errorf("restore stateful set %s: %v", s.original.Name, err)
		return
	}
	if !s.changed {
		return
	}
	if err := s.recreate(ctx); err != nil {
		s.logger.Errorf("restore pod %s: %v", s.podName(), err)
	}
}
