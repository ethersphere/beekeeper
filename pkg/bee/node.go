package bee

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"text/template"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/beeclient/debugapi"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/k8s/configmap"
	"github.com/ethersphere/beekeeper/pkg/k8s/ingress"
	"github.com/ethersphere/beekeeper/pkg/k8s/secret"
	"github.com/ethersphere/beekeeper/pkg/k8s/service"
	"github.com/ethersphere/beekeeper/pkg/k8s/serviceaccount"
	"github.com/ethersphere/beekeeper/pkg/k8s/statefulset"
	bmtlegacy "github.com/ethersphere/bmt/legacy"
)

// Client manages communication with the Bee node
type Client struct {
	api   *api.Client
	debug *debugapi.Client
	k8s   *k8s.Client
}

// ClientOptions holds optional parameters for the Client.
type ClientOptions struct {
	APIURL              *url.URL
	APIInsecureTLS      bool
	DebugAPIURL         *url.URL
	DebugAPIInsecureTLS bool
	KubeconfigPath      string
}

// NewClient returns Bee client
func NewClient(opts ClientOptions) (c Client) {
	if opts.APIURL != nil {
		c.api = api.NewClient(opts.APIURL, &api.ClientOptions{HTTPClient: &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: opts.APIInsecureTLS},
		}}})
	}
	if opts.DebugAPIURL != nil {
		c.debug = debugapi.NewClient(opts.DebugAPIURL, &debugapi.ClientOptions{HTTPClient: &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: opts.DebugAPIInsecureTLS},
		}}})
	}
	if len(opts.KubeconfigPath) > 0 {
		c.k8s = k8s.NewClient(&k8s.ClientOptions{KubeconfigPath: opts.KubeconfigPath})
	}

	return
}

// StartOptions ...
type StartOptions struct {
	Name    string
	Version string
	Config  Config
	K8S     *K8SOptions
}

// K8SOptions ...
type K8SOptions struct {
	Name        string
	Namespace   string
	Annotations map[string]string
	Labels      map[string]string
}

// Start ...
func (c Client) Start(ctx context.Context, o StartOptions) (err error) {
	// configuration
	var cm bytes.Buffer
	if err := template.Must(template.New("").Parse(configTemplate)).Execute(&cm, o.Config); err != nil {
		return err
	}

	if err = c.k8s.ConfigMap.Set(ctx, o.Name, o.K8S.Namespace, configmap.Options{
		Annotations: o.K8S.Annotations,
		Labels:      o.K8S.Labels,
		Data: map[string]string{
			".bee.yaml": cm.String(),
		},
	}); err != nil {
		return fmt.Errorf("set ConfigMap: %s", err)
	}

	if err := c.k8s.Secret.Set(ctx, fmt.Sprintf("%s-libp2p", o.K8S.Name), o.K8S.Namespace, secret.Options{
		Annotations: o.K8S.Annotations,
		Labels:      o.K8S.Labels,
		StringData: map[string]string{
			"libp2pKeys": `bee-0: {"address":"aa6675fb77f3f84304a00d5ea09902d8a500364091a457cf21e05a41875d48f7","crypto":{"cipher":"aes-128-ctr","ciphertext":"93effebd3f015f496367e14218cb26d22de8f899e1d7b7686deb6ab43c876ea5","cipherparams":{"iv":"627434462c2f960d37338022d27fc92e"},"kdf":"scrypt","kdfparams":{"n":32768,"r":8,"p":1,"dklen":32,"salt":"a59e72e725fe3de25dd9c55aa55a93ed0e9090b408065a7204e2f505653acb70"},"mac":"dfb1e7ad93252928a7ff21ea5b65e8a4b9bda2c2e09cb6a8ac337da7a3568b8c"},"version":3}
bee-1: {"address":"03348ecf3adae1d05dc16e475a83c94e49e28a4d3c7db5eccbf5ca4ea7f688ddcdfe88acbebef2037c68030b1a0a367a561333e5c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470","crypto":{"cipher":"aes-128-ctr","ciphertext":"0d0ff25e9b03292e622c5a09ec00c2acb7ff5882f02dd2f00a26ac6d3292a434","cipherparams":{"iv":"cd4082caf63320b306fe885796ba224f"},"kdf":"scrypt","kdfparams":{"n":32768,"r":8,"p":1,"dklen":32,"salt":"a4d63d56c539eb3eff2a235090127486722fa2c836cf735d50d673b730cebc3f"},"mac":"aad40da9c1e742e2b01bb8f76ba99ace97ccb0539cea40e31eb6b9bb64a3f36a"},"version":3}`,
		},
	}); err != nil {
		return fmt.Errorf("set Secret: %s", err)
	}

	// services
	if err := c.k8s.ServiceAccount.Set(ctx, o.K8S.Name, o.K8S.Namespace, serviceaccount.Options{
		Annotations: o.K8S.Annotations,
		Labels:      o.K8S.Labels,
	}); err != nil {
		return fmt.Errorf("set ServiceAccount %s", err)
	}

	if err := c.k8s.Service.Set(ctx, o.K8S.Name, o.K8S.Namespace, service.Options{
		Annotations: o.K8S.Annotations,
		Labels:      o.K8S.Labels,
		Ports: []service.Port{{
			Name:       "http",
			Protocol:   "TCP",
			Port:       80,
			TargetPort: "api",
		}},
		Selector: map[string]string{
			"app.kubernetes.io/instance":   "bee",
			"app.kubernetes.io/name":       "bee",
			"app.kubernetes.io/managed-by": "beekeeper",
		},
		Type: "ClusterIP",
	}); err != nil {
		return fmt.Errorf("set Service %s", err)
	}

	if err := c.k8s.Service.Set(ctx, fmt.Sprintf("%s-headless", o.K8S.Name), o.K8S.Namespace, service.Options{
		Annotations: o.K8S.Annotations,
		Labels:      o.K8S.Labels,
		Ports: []service.Port{
			{
				Name:       "api",
				Protocol:   "TCP",
				Port:       8080,
				TargetPort: "api",
			},
			{
				Name:       "p2p",
				Protocol:   "TCP",
				Port:       7070,
				TargetPort: "p2p",
			},
			{
				Name:       "debug",
				Protocol:   "TCP",
				Port:       6060,
				TargetPort: "debug",
			},
		},
		Selector: map[string]string{
			"app.kubernetes.io/instance":   "bee",
			"app.kubernetes.io/name":       "bee",
			"app.kubernetes.io/managed-by": "beekeeper",
		},
		Type: "ClusterIP",
	}); err != nil {
		return fmt.Errorf("set Service %s", err)
	}

	// ingress
	if err := c.k8s.Ingress.Set(ctx, o.K8S.Name, o.K8S.Namespace, ingress.Options{
		Annotations: map[string]string{
			"createdBy":                                          "beekeeper",
			"kubernetes.io/ingress.class":                        "nginx-internal",
			"nginx.ingress.kubernetes.io/affinity":               "cookie",
			"nginx.ingress.kubernetes.io/affinity-mode":          "persistent",
			"nginx.ingress.kubernetes.io/proxy-body-size":        "0",
			"nginx.ingress.kubernetes.io/proxy-read-timeout":     "7200",
			"nginx.ingress.kubernetes.io/proxy-send-timeout":     "7200",
			"nginx.ingress.kubernetes.io/session-cookie-max-age": "86400",
			"nginx.ingress.kubernetes.io/session-cookie-name":    "SWARMGATEWAY",
			"nginx.ingress.kubernetes.io/session-cookie-path":    "default",
			"nginx.ingress.kubernetes.io/ssl-redirect":           "true",
		},
		Labels:      o.K8S.Labels,
		Class:       "nginx-internal",
		Host:        "bee.beekeeper.staging.internal",
		ServiceName: o.K8S.Name,
		ServicePort: "http",
		Path:        "/",
	}); err != nil {
		return fmt.Errorf("set Ingress %s", err)
	}

	// statefulset
	if err := c.k8s.StatefulSet.Set(ctx, fmt.Sprintf("%s-0", o.K8S.Name), o.K8S.Namespace, statefulset.Options{
		Annotations: o.K8S.Annotations,
		Labels:      o.K8S.Labels,
		Replicas:    1,
		Selector: map[string]string{
			"app.kubernetes.io/instance":   "bee",
			"app.kubernetes.io/name":       "bee",
			"app.kubernetes.io/managed-by": "beekeeper",
		},
		RestartPolicy:      "Always",
		ServiceAccountName: o.K8S.Name,
		ServiceName:        fmt.Sprintf("%s-headless", o.K8S.Name),
		NodeSelector: map[string]string{
			"node-group": "bee-staging",
		},
		PodManagementPolicy: "OrderedReady",
		PodSecurityContext: statefulset.PodSecurityContext{
			FSGroup: 999,
		},
		UpdateStrategy: statefulset.UpdateStrategy{
			Type: "OnDelete",
		},
		Volumes: []statefulset.Volume{
			{ConfigMap: &statefulset.ConfigMapVolume{
				Name:          "config",
				ConfigMapName: o.K8S.Name,
				DefaultMode:   420,
			}},
			{EmptyDir: &statefulset.EmptyDirVolume{
				Name: "data",
			}},
			{Secret: &statefulset.SecretVolume{
				Name:        fmt.Sprintf("%s-libp2p", o.K8S.Name),
				SecretName:  fmt.Sprintf("%s-libp2p", o.K8S.Name),
				DefaultMode: 420,
				Items: []statefulset.Item{{
					Key:   "libp2pKeys",
					Value: "libp2p.map",
				}},
			}},
		},
		InitContainers: []statefulset.InitContainer{{
			Name:    "init-libp2p",
			Image:   "busybox:1.28",
			Command: []string{"sh", "-c", `export INDEX=$(echo $(hostname) | rev | cut -d'-' -f 1 | rev); mkdir -p /home/bee/.bee/keys; chown -R 999:999 /home/bee/.bee/keys; export KEY=$(cat /tmp/bee/libp2p.map | grep bee-${INDEX}: | cut -d' ' -f2); if [ -z "${KEY}" ]; then exit 0; fi; printf '%s' "${KEY}" > /home/bee/.bee/keys/libp2p.key; echo 'node initialization done';`},
			VolumeMounts: []statefulset.VolumeMount{
				{Name: "bee-libp2p", MountPath: "/tmp/bee"},
				{Name: "data", MountPath: "home/bee/.bee"},
			},
		}},
		Containers: []statefulset.Container{{
			Name:            o.K8S.Name,
			Image:           "ethersphere/bee:latest",
			ImagePullPolicy: "Always",
			Command:         []string{"bee", "start", "--config=.bee.yaml"},
			Ports: []statefulset.Port{
				{
					Name:          "api",
					ContainerPort: 8080,
					Protocol:      "TCP",
				},
				{
					Name:          "p2p",
					ContainerPort: 7070,
					Protocol:      "TCP",
				},
				{
					Name:          "debug",
					ContainerPort: 6060,
					Protocol:      "TCP",
				},
			},
			LivenessProbe: statefulset.Probe{
				Path: "/health",
				Port: "debug",
			},
			ReadinessProbe: statefulset.Probe{
				Path: "/readiness",
				Port: "debug",
			},
			Resources: statefulset.Resources{
				LimitCPU:      "1",
				LimitMemory:   "2Gi",
				RequestCPU:    "750m",
				RequestMemory: "1Gi",
			},
			SecurityContext: statefulset.SecurityContext{
				AllowPrivilegeEscalation: false,
				RunAsUser:                999,
			},
			VolumeMounts: []statefulset.VolumeMount{
				{Name: "config", MountPath: "/home/bee/.bee.yaml", SubPath: ".bee.yaml", ReadOnly: true},
				{Name: "data", MountPath: "home/bee/.bee"},
			},
		}},
	}); err != nil {
		return fmt.Errorf("set StatefulSet %s", err)
	}

	fmt.Println("Node started")

	return
}

// Addresses represents node's addresses
type Addresses struct {
	Overlay  swarm.Address
	Underlay []string
}

// Addresses returns node's addresses
func (c *Client) Addresses(ctx context.Context) (resp Addresses, err error) {
	a, err := c.debug.Node.Addresses(ctx)
	if err != nil {
		return Addresses{}, fmt.Errorf("get addresses: %w", err)
	}

	return Addresses{
		Overlay:  a.Overlay,
		Underlay: a.Underlay,
	}, nil
}

// Balance represents node's balance with peer
type Balance struct {
	Balance int
	Peer    string
}

// Balance returns node's balance with a given peer
func (c *Client) Balance(ctx context.Context, a swarm.Address) (resp Balance, err error) {
	b, err := c.debug.Node.Balance(ctx, a)
	if err != nil {
		return Balance{}, fmt.Errorf("get balance with node %s: %w", a.String(), err)
	}

	return Balance{
		Balance: b.Balance,
		Peer:    b.Peer,
	}, nil
}

// Balances represents Balances's response
type Balances struct {
	Balances []Balance
}

// Balances returns node's balances
func (c *Client) Balances(ctx context.Context) (resp Balances, err error) {
	r, err := c.debug.Node.Balances(ctx)
	if err != nil {
		return Balances{}, fmt.Errorf("get balances: %w", err)
	}

	for _, b := range r.Balances {
		resp.Balances = append(resp.Balances, Balance{
			Balance: b.Balance,
			Peer:    b.Peer,
		})
	}

	return
}

// DownloadBytes downloads chunk from the node
func (c *Client) DownloadBytes(ctx context.Context, a swarm.Address) (data []byte, err error) {
	r, err := c.api.Bytes.Download(ctx, a)
	if err != nil {
		return nil, fmt.Errorf("download chunk %s: %w", a, err)
	}

	return ioutil.ReadAll(r)
}

// DownloadChunk downloads chunk from the node
func (c *Client) DownloadChunk(ctx context.Context, a swarm.Address, targets string) (data []byte, err error) {
	r, err := c.api.Chunks.Download(ctx, a, targets)
	if err != nil {
		return nil, fmt.Errorf("download chunk %s: %w", a, err)
	}

	return ioutil.ReadAll(r)
}

// DownloadFile downloads chunk from the node and returns it's size and hash
func (c *Client) DownloadFile(ctx context.Context, a swarm.Address) (size int64, hash []byte, err error) {
	r, err := c.api.Files.Download(ctx, a)
	if err != nil {
		return 0, nil, fmt.Errorf("download file %s: %w", a, err)
	}

	h := fileHahser()
	size, err = io.Copy(h, r)
	if err != nil {
		return 0, nil, fmt.Errorf("download file %s, hashing copy: %w", a, err)
	}

	return size, h.Sum(nil), nil
}

// HasChunk returns true/false if node has a chunk
func (c *Client) HasChunk(ctx context.Context, a swarm.Address) (bool, error) {
	return c.debug.Node.HasChunk(ctx, a)
}

// Overlay returns node's overlay address
func (c *Client) Overlay(ctx context.Context) (swarm.Address, error) {
	a, err := c.debug.Node.Addresses(ctx)
	if err != nil {
		return swarm.Address{}, fmt.Errorf("get overlay: %w", err)
	}

	return a.Overlay, nil
}

// Peers returns addresses of node's peers
func (c *Client) Peers(ctx context.Context) (peers []swarm.Address, err error) {
	ps, err := c.debug.Node.Peers(ctx)
	if err != nil {
		return nil, fmt.Errorf("get peers: %w", err)
	}

	for _, p := range ps.Peers {
		peers = append(peers, p.Address)
	}

	return
}

// PinChunk returns true/false if chunk pinning is successful
func (c *Client) PinChunk(ctx context.Context, a swarm.Address) (bool, error) {
	return c.api.Pinning.PinChunk(ctx, a)
}

// PinnedChunk represents pinned chunk
type PinnedChunk struct {
	Address    swarm.Address
	PinCounter int
}

// PinnedChunk returns pinned chunk
func (c *Client) PinnedChunk(ctx context.Context, a swarm.Address) (PinnedChunk, error) {
	p, err := c.api.Pinning.PinnedChunk(ctx, a)
	if err != nil {
		return PinnedChunk{}, fmt.Errorf("get pinned chunk: %w", err)
	}

	return PinnedChunk{
		Address:    p.Address,
		PinCounter: p.PinCounter,
	}, nil
}

// PinnedChunks represents pinned chunks
type PinnedChunks struct {
	Chunks []PinnedChunk
}

// PinnedChunks returns pinned chunks
func (c *Client) PinnedChunks(ctx context.Context) (PinnedChunks, error) {
	p, err := c.api.Pinning.PinnedChunks(ctx)
	if err != nil {
		return PinnedChunks{}, fmt.Errorf("get pinned chunks: %w", err)
	}

	r := PinnedChunks{}
	for _, c := range p.Chunks {
		r.Chunks = append(r.Chunks, PinnedChunk{
			Address:    c.Address,
			PinCounter: c.PinCounter,
		})
	}

	return r, nil
}

// Ping pings other node
func (c *Client) Ping(ctx context.Context, node swarm.Address) (rtt string, err error) {
	r, err := c.debug.PingPong.Ping(ctx, node)
	if err != nil {
		return "", fmt.Errorf("ping node %s: %w", node, err)
	}
	return r.RTT, nil
}

// PingStreamMsg represents message sent over the PingStream channel
type PingStreamMsg struct {
	Node  swarm.Address
	RTT   string
	Index int
	Error error
}

// PingStream returns stream of ping results for given nodes
func (c *Client) PingStream(ctx context.Context, nodes []swarm.Address) <-chan PingStreamMsg {
	pingStream := make(chan PingStreamMsg)

	var wg sync.WaitGroup
	for i, node := range nodes {
		wg.Add(1)
		go func(i int, node swarm.Address) {
			defer wg.Done()

			rtt, err := c.Ping(ctx, node)
			pingStream <- PingStreamMsg{
				Node:  node,
				RTT:   rtt,
				Index: i,
				Error: err,
			}
		}(i, node)
	}

	go func() {
		wg.Wait()
		close(pingStream)
	}()

	return pingStream
}

// Settlement represents node's settlement with peer
type Settlement struct {
	Peer     string
	Received int
	Sent     int
}

// Settlement returns node's settlement with a given peer
func (c *Client) Settlement(ctx context.Context, a swarm.Address) (resp Settlement, err error) {
	b, err := c.debug.Node.Settlement(ctx, a)
	if err != nil {
		return Settlement{}, fmt.Errorf("get settlement with node %s: %w", a.String(), err)
	}

	return Settlement{
		Peer:     b.Peer,
		Received: b.Received,
		Sent:     b.Sent,
	}, nil
}

// Settlements represents Settlements's response
type Settlements struct {
	Settlements   []Settlement
	TotalReceived int
	TotalSent     int
}

// Settlements returns node's settlements
func (c *Client) Settlements(ctx context.Context) (resp Settlements, err error) {
	r, err := c.debug.Node.Settlements(ctx)
	if err != nil {
		return Settlements{}, fmt.Errorf("get settlements: %w", err)
	}

	for _, b := range r.Settlements {
		resp.Settlements = append(resp.Settlements, Settlement{
			Peer:     b.Peer,
			Received: b.Received,
			Sent:     b.Sent,
		})
	}
	resp.TotalReceived = r.TotalReceived
	resp.TotalSent = r.TotalSent

	return
}

// Topology represents Kademlia topology
type Topology struct {
	Overlay        swarm.Address
	Connected      int
	Population     int
	NnLowWatermark int
	Depth          int
	Bins           map[string]Bin
}

// Bin represents Kademlia bin
type Bin struct {
	Connected         int
	ConnectedPeers    []swarm.Address
	DisconnectedPeers []swarm.Address
	Population        int
}

// Topology returns Kademlia topology
func (c *Client) Topology(ctx context.Context) (topology Topology, err error) {
	t, err := c.debug.Node.Topology(ctx)
	if err != nil {
		return Topology{}, fmt.Errorf("get topology: %w", err)
	}

	topology = Topology{
		Overlay:        t.BaseAddr,
		Connected:      t.Connected,
		Population:     t.Population,
		NnLowWatermark: t.NnLowWatermark,
		Depth:          t.Depth,
		Bins:           make(map[string]Bin),
	}

	for k, b := range t.Bins {
		if b.Population > 0 {
			topology.Bins[k] = Bin{
				Connected:         b.Connected,
				ConnectedPeers:    b.ConnectedPeers,
				DisconnectedPeers: b.DisconnectedPeers,
				Population:        b.Population,
			}
		}
	}

	return
}

// Underlay returns node's underlay addresses
func (c *Client) Underlay(ctx context.Context) ([]string, error) {
	a, err := c.debug.Node.Addresses(ctx)
	if err != nil {
		return nil, fmt.Errorf("get underlay: %w", err)
	}

	return a.Underlay, nil
}

// UnpinChunk returns true/false if chunk unpinning is successful
func (c *Client) UnpinChunk(ctx context.Context, a swarm.Address) (bool, error) {
	return c.api.Pinning.UnpinChunk(ctx, a)
}

// UploadBytes uploads chunk to the node
func (c *Client) UploadBytes(ctx context.Context, chunk *Chunk) (err error) {
	r, err := c.api.Bytes.Upload(ctx, bytes.NewReader(chunk.Data()))
	if err != nil {
		return fmt.Errorf("upload chunk: %w", err)
	}

	chunk.address = r.Reference

	return
}

// UploadChunk uploads chunk to the node
func (c *Client) UploadChunk(ctx context.Context, chunk *Chunk) (err error) {
	p := bmtlegacy.NewTreePool(chunkHahser, swarm.Branches, bmtlegacy.PoolSize)
	hasher := bmtlegacy.New(p)
	err = hasher.SetSpan(int64(chunk.Span()))
	if err != nil {
		return fmt.Errorf("upload chunk: %w", err)
	}
	_, err = hasher.Write(chunk.Data()[8:])
	if err != nil {
		return fmt.Errorf("upload chunk: %w", err)
	}
	chunk.address = swarm.NewAddress(hasher.Sum(nil))

	_, err = c.api.Chunks.Upload(ctx, chunk.address, bytes.NewReader(chunk.Data()))
	if err != nil {
		return fmt.Errorf("upload chunk: %w", err)
	}

	return
}

// RemoveChunk removes the given chunk from the node's local store
func (c *Client) RemoveChunk(ctx context.Context, chunk *Chunk) (err error) {
	return c.debug.Chunks.Remove(ctx, chunk.Address())
}

// UploadFile uploads file to the node
func (c *Client) UploadFile(ctx context.Context, f *File, pin bool) (err error) {
	h := fileHahser()
	r, err := c.api.Files.Upload(ctx, f.Name(), io.TeeReader(f.DataReader(), h), f.Size(), pin)
	if err != nil {
		return fmt.Errorf("upload file: %w", err)
	}

	f.address = r.Reference
	f.hash = h.Sum(nil)

	return
}

// UploadCollection uploads TAR collection bytes to the node
func (c *Client) UploadCollection(ctx context.Context, f *File) (err error) {
	h := fileHahser()
	r, err := c.api.Dirs.Upload(ctx, io.TeeReader(f.DataReader(), h), f.Size())
	if err != nil {
		return fmt.Errorf("upload collection: %w", err)
	}

	f.address = r.Reference
	f.hash = h.Sum(nil)

	return
}

// DownloadManifestFile downloads manifest file from the node and returns it's size and hash
func (c *Client) DownloadManifestFile(ctx context.Context, a swarm.Address, path string) (size int64, hash []byte, err error) {
	r, err := c.api.Dirs.Download(ctx, a, path)
	if err != nil {
		return 0, nil, fmt.Errorf("download manifest file %s: %w", path, err)
	}

	h := fileHahser()
	size, err = io.Copy(h, r)
	if err != nil {
		return 0, nil, fmt.Errorf("download manifest file %s: %w", path, err)
	}

	return size, h.Sum(nil), nil
}
