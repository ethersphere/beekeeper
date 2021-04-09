package cmd

import (
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/check"
	"github.com/ethersphere/beekeeper/pkg/check/balances"
	"github.com/ethersphere/beekeeper/pkg/check/chunkrepair"
	"github.com/ethersphere/beekeeper/pkg/check/fileretrieval"
	"github.com/ethersphere/beekeeper/pkg/check/fullconnectivity"
	"github.com/ethersphere/beekeeper/pkg/check/gc"
	"github.com/ethersphere/beekeeper/pkg/check/kademlia"
	"github.com/ethersphere/beekeeper/pkg/check/localpinning"
	"github.com/ethersphere/beekeeper/pkg/check/manifest"
	"github.com/ethersphere/beekeeper/pkg/check/peercount"
	"github.com/ethersphere/beekeeper/pkg/check/ping"
	"github.com/ethersphere/beekeeper/pkg/check/pss"
	"github.com/ethersphere/beekeeper/pkg/check/pullsync"
	"github.com/ethersphere/beekeeper/pkg/check/pushsync"
	"github.com/ethersphere/beekeeper/pkg/check/retrieval"
	"github.com/ethersphere/beekeeper/pkg/check/settlements"
	"github.com/ethersphere/beekeeper/pkg/check/soc"
	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
)

var Checks = map[string]Check{
	"balances": {
		NewCheck: balances.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(balancesOptions)
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			var opts balances.Options
			// TODO: improve Run["profile"] selection
			// set seed
			if o.Seed == nil && cfg.Run["default"].Seed > 0 { // enabled globaly
				opts.Seed = cfg.Run["default"].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // enabled localy
				opts.Seed = *o.Seed
			} else { // randomly generated
				opts.Seed = random.Int64()
			}
			if o.DryRun != nil {
				opts.DryRun = *o.DryRun
			}
			if o.FileName != nil {
				opts.FileName = *o.FileName
			} else {
				opts.FileName = "balances"
			}
			if o.FileSize != nil {
				opts.FileSize = *o.FileSize
			} else {
				opts.FileSize = 1 * 1024 * 1024 // 1mb
			}
			if o.NodeGroup != nil {
				opts.NodeGroup = *o.NodeGroup
			} else {
				opts.NodeGroup = "bee"
			}
			if o.UploadNodeCount != nil {
				opts.UploadNodeCount = *o.UploadNodeCount
			} else {
				opts.UploadNodeCount = 1
			}
			if o.WaitBeforeDownload != nil {
				opts.WaitBeforeDownload = *o.WaitBeforeDownload
			} else {
				opts.WaitBeforeDownload = 5 // seconds
			}
			return opts, nil
		},
	},
	"chunk-repair": {
		NewCheck: chunkrepair.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(chunkRepairOptions)
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			var opts chunkrepair.Options
			// TODO: improve Run["profile"] selection
			// set seed
			if o.Seed == nil && cfg.Run["default"].Seed > 0 { // enabled globaly
				opts.Seed = cfg.Run["default"].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // enabled localy
				opts.Seed = *o.Seed
			} else { // randomly generated
				opts.Seed = random.Int64()
			}
			// TODO: resolve optionNamePushGateway
			// set metrics
			if o.MetricsEnabled == nil && cfg.Run["default"].MetricsEnabled { // enabled globaly
				opts.MetricsPusher = push.New("optionNamePushGateway", cfg.Cluster.Namespace)
			} else if o.MetricsEnabled != nil && *o.MetricsEnabled { // enabled localy
				opts.MetricsPusher = push.New("optionNamePushGateway", cfg.Cluster.Namespace)
			}
			if o.NodeGroup != nil {
				opts.NodeGroup = *o.NodeGroup
			} else {
				opts.NodeGroup = "bee"
			}
			if o.NumberOfChunksToRepair != nil {
				opts.NumberOfChunksToRepair = *o.NumberOfChunksToRepair
			} else {
				opts.NumberOfChunksToRepair = 1
			}
			return opts, nil
		},
	},
	"file-retrieval": {
		NewCheck: fileretrieval.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(fileRetrievalOptions)
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			var opts fileretrieval.Options
			// TODO: improve Run["profile"] selection
			// set seed
			if o.Seed == nil && cfg.Run["default"].Seed > 0 { // enabled globaly
				opts.Seed = cfg.Run["default"].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // enabled localy
				opts.Seed = *o.Seed
			} else { // randomly generated
				opts.Seed = random.Int64()
			}
			// TODO: resolve optionNamePushGateway
			// set metrics
			if o.MetricsEnabled == nil && cfg.Run["default"].MetricsEnabled { // enabled globaly
				opts.MetricsPusher = push.New("optionNamePushGateway", cfg.Cluster.Namespace)
			} else if o.MetricsEnabled != nil && *o.MetricsEnabled { // enabled localy
				opts.MetricsPusher = push.New("optionNamePushGateway", cfg.Cluster.Namespace)
			}
			if o.FileName != nil {
				opts.FileName = *o.FileName
			} else {
				opts.FileName = "file-retrieval"
			}
			if o.FileSize != nil {
				opts.FileSize = *o.FileSize
			} else {
				opts.FileSize = 1 * 1024 * 1024 // 1mb
			}
			if o.FilesPerNode != nil {
				opts.FilesPerNode = *o.FilesPerNode
			} else {
				opts.FilesPerNode = 1
			}
			if o.Full != nil {
				opts.Full = *o.Full
			}
			if o.NodeGroup != nil {
				opts.NodeGroup = *o.NodeGroup
			} else {
				opts.NodeGroup = "bee"
			}
			if o.UploadNodeCount != nil {
				opts.UploadNodeCount = *o.UploadNodeCount
			} else {
				opts.UploadNodeCount = 1
			}
			return opts, nil
		},
	},
	"full-connectivity": {
		NewCheck: fullconnectivity.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			return nil, nil
		},
	},
	"gc": {
		NewCheck: gc.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(gcOptions)
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			var opts gc.Options
			// TODO: improve Run["profile"] selection
			// set seed
			if o.Seed == nil && cfg.Run["default"].Seed > 0 { // enabled globaly
				opts.Seed = cfg.Run["default"].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // enabled localy
				opts.Seed = *o.Seed
			} else { // randomly generated
				opts.Seed = random.Int64()
			}
			if o.NodeGroup != nil {
				opts.NodeGroup = *o.NodeGroup
			} else {
				opts.NodeGroup = "bee"
			}
			if o.StoreSize != nil {
				opts.StoreSize = *o.StoreSize
			} else {
				opts.StoreSize = 1000 // DB capacity in chunks
			}
			if o.StoreSizeDivisor != nil {
				opts.StoreSizeDivisor = *o.StoreSizeDivisor
			} else {
				opts.StoreSizeDivisor = 3 // divide store size by which value when uploading bytes
			}
			if o.Wait != nil {
				opts.Wait = *o.Wait
			} else {
				opts.Wait = 5 // wait before check
			}
			return opts, nil
		},
	},
	"kademlia": {
		NewCheck: kademlia.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(kademliaOptions)
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			var opts kademlia.Options
			if o.Dynamic != nil {
				opts.Dynamic = *o.Dynamic
			}

			return opts, nil
		},
	},
	"local-pinning": {
		NewCheck: localpinning.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(localpinningOptions)
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			var opts localpinning.Options
			// TODO: improve Run["profile"] selection
			// set seed
			if o.Seed == nil && cfg.Run["default"].Seed > 0 { // enabled globaly
				opts.Seed = cfg.Run["default"].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // enabled localy
				opts.Seed = *o.Seed
			} else { // randomly generated
				opts.Seed = random.Int64()
			}
			if o.Mode != nil {
				opts.Mode = *o.Mode
			} else {
				opts.Mode = "pin-chunk"
			}
			if o.NodeGroup != nil {
				opts.NodeGroup = *o.NodeGroup
			} else {
				opts.NodeGroup = "bee"
			}
			if o.StoreSize != nil {
				opts.StoreSize = *o.StoreSize
			} else {
				opts.StoreSize = 1000 // DB capacity in chunks
			}
			if o.StoreSizeDivisor != nil {
				opts.StoreSizeDivisor = *o.StoreSizeDivisor
			} else {
				opts.StoreSizeDivisor = 3 // divide store size by which value when uploading bytes
			}
			return opts, nil
		},
	},
	"local-pinning-bytes": {
		NewCheck: localpinning.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(localpinningOptions)
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			var opts localpinning.Options
			// TODO: improve Run["profile"] selection
			// set seed
			if o.Seed == nil && cfg.Run["default"].Seed > 0 { // enabled globaly
				opts.Seed = cfg.Run["default"].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // enabled localy
				opts.Seed = *o.Seed
			} else { // randomly generated
				opts.Seed = random.Int64()
			}
			if o.NodeGroup != nil {
				opts.NodeGroup = *o.NodeGroup
			} else {
				opts.NodeGroup = "bee"
			}
			if o.StoreSize != nil {
				opts.StoreSize = *o.StoreSize
			} else {
				opts.StoreSize = 1000 // DB capacity in chunks
			}
			if o.StoreSizeDivisor != nil {
				opts.StoreSizeDivisor = *o.StoreSizeDivisor
			} else {
				opts.StoreSizeDivisor = 3 // divide store size by which value when uploading bytes
			}
			return opts, nil
		},
	},
	"local-pinning-remote": {
		NewCheck: localpinning.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(localpinningOptions)
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			var opts localpinning.Options
			// TODO: improve Run["profile"] selection
			// set seed
			if o.Seed == nil && cfg.Run["default"].Seed > 0 { // enabled globaly
				opts.Seed = cfg.Run["default"].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // enabled localy
				opts.Seed = *o.Seed
			} else { // randomly generated
				opts.Seed = random.Int64()
			}
			if o.NodeGroup != nil {
				opts.NodeGroup = *o.NodeGroup
			} else {
				opts.NodeGroup = "bee"
			}
			if o.StoreSize != nil {
				opts.StoreSize = *o.StoreSize
			} else {
				opts.StoreSize = 1000 // DB capacity in chunks
			}
			if o.StoreSizeDivisor != nil {
				opts.StoreSizeDivisor = *o.StoreSizeDivisor
			} else {
				opts.StoreSizeDivisor = 3 // divide store size by which value when uploading bytes
			}
			return opts, nil
		},
	},
	"manifest": {
		NewCheck: manifest.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(manifestOptions)
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			var opts manifest.Options
			// TODO: improve Run["profile"] selection
			// set seed
			if o.Seed == nil && cfg.Run["default"].Seed > 0 { // enabled globaly
				opts.Seed = cfg.Run["default"].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // enabled localy
				opts.Seed = *o.Seed
			} else { // randomly generated
				opts.Seed = random.Int64()
			}
			return opts, nil
		},
	},
	"peercount": {
		NewCheck: peercount.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(peercountOptions)
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			var opts peercount.Options
			// TODO: improve Run["profile"] selection
			// set seed
			if o.Seed == nil && cfg.Run["default"].Seed > 0 { // enabled globaly
				opts.Seed = cfg.Run["default"].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // enabled localy
				opts.Seed = *o.Seed
			} else { // randomly generated
				opts.Seed = random.Int64()
			}
			return opts, nil
		},
	},
	"ping": {
		NewCheck: ping.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(pingOptions)
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s optiosns: %w", checkProfile.Name, err)
			}
			var opts ping.Options
			// TODO: improve Run["profile"] selection
			// TODO: resolve optionNamePushGateway
			// set metrics
			if o.MetricsEnabled == nil && cfg.Run["default"].MetricsEnabled { // enabled globaly
				opts.MetricsPusher = push.New("optionNamePushGateway", cfg.Cluster.Namespace)
			} else if o.MetricsEnabled != nil && *o.MetricsEnabled { // enabled localy
				opts.MetricsPusher = push.New("optionNamePushGateway", cfg.Cluster.Namespace)
			}
			return opts, nil
		},
	},
	"pss": {
		NewCheck: pss.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(pssOptions)
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s optiosns: %w", checkProfile.Name, err)
			}
			var opts pss.Options
			// TODO: improve Run["profile"] selection
			// set seed
			if o.Seed == nil && cfg.Run["default"].Seed > 0 { // enabled globaly
				opts.Seed = cfg.Run["default"].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // enabled localy
				opts.Seed = *o.Seed
			} else { // randomly generated
				opts.Seed = random.Int64()
			}
			return opts, nil
		},
	},
	"pullsync": {
		NewCheck: pullsync.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(pullSyncOptions)
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s optiosns: %w", checkProfile.Name, err)
			}
			var opts pullsync.Options
			// TODO: improve Run["profile"] selection
			// set seed
			if o.Seed == nil && cfg.Run["default"].Seed > 0 { // enabled globaly
				opts.Seed = cfg.Run["default"].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // enabled localy
				opts.Seed = *o.Seed
			} else { // randomly generated
				opts.Seed = random.Int64()
			}
			return opts, nil
		},
	},
	"pushsync": {
		NewCheck: pushsync.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(pushSyncOptions)
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s optiosns: %w", checkProfile.Name, err)
			}
			var opts pushsync.Options
			// TODO: improve Run["profile"] selection
			// set seed
			if o.Seed == nil && cfg.Run["default"].Seed > 0 { // enabled globaly
				opts.Seed = cfg.Run["default"].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // enabled localy
				opts.Seed = *o.Seed
			} else { // randomly generated
				opts.Seed = random.Int64()
			}
			return opts, nil
		},
	},
	"retrieval": {
		NewCheck: retrieval.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(retrievalOptions)
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s optiosns: %w", checkProfile.Name, err)
			}
			var opts retrieval.Options
			// TODO: improve Run["profile"] selection
			// set seed
			if o.Seed == nil && cfg.Run["default"].Seed > 0 { // enabled globaly
				opts.Seed = cfg.Run["default"].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // enabled localy
				opts.Seed = *o.Seed
			} else { // randomly generated
				opts.Seed = random.Int64()
			}
			return opts, nil
		},
	},
	"settlements": {
		NewCheck: settlements.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(settlementsOptions)
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s optiosns: %w", checkProfile.Name, err)
			}
			var opts settlements.Options
			// TODO: improve Run["profile"] selection
			// set seed
			if o.Seed == nil && cfg.Run["default"].Seed > 0 { // enabled globaly
				opts.Seed = cfg.Run["default"].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // enabled localy
				opts.Seed = *o.Seed
			} else { // randomly generated
				opts.Seed = random.Int64()
			}
			return opts, nil
		},
	},
	"soc": {
		NewCheck: soc.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(socOptions)
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s optiosns: %w", checkProfile.Name, err)
			}
			var opts soc.Options
			// TODO: improve Run["profile"] selection
			// set seed
			if o.Seed == nil && cfg.Run["default"].Seed > 0 { // enabled globaly
				opts.Seed = cfg.Run["default"].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // enabled localy
				opts.Seed = *o.Seed
			} else { // randomly generated
				opts.Seed = random.Int64()
			}
			return opts, nil
		},
	},
}

type Check struct {
	NewCheck   func() check.Check
	NewOptions func(cfg *config.Config, checkProfile config.Check) (interface{}, error)
}

type balancesOptions struct {
	DryRun             *bool   `yaml:"dry-run"`
	FileName           *string `yaml:"file-name"`
	FileSize           *int64  `yaml:"file-size"`
	NodeGroup          *string `yaml:"node-group"`
	Seed               *int64  `yaml:"seed"`
	UploadNodeCount    *int    `yaml:"upload-node-count"`
	WaitBeforeDownload *int    `yaml:"wait-before-download"`
}

type chunkRepairOptions struct {
	MetricsEnabled         *bool   `yaml:"metrics-enabled"`
	NodeGroup              *string `yaml:"node-group"`
	NumberOfChunksToRepair *int    `yaml:"number-of-chunks-to-repair"`
	Seed                   *int64  `yaml:"seed"`
}

type fileRetrievalOptions struct {
	FileName        *string `yaml:"file-name"`
	FileSize        *int64  `yaml:"file-size"`
	FilesPerNode    *int    `yaml:"files-per-node"`
	Full            *bool   `yaml:"full"`
	MetricsEnabled  *bool   `yaml:"metrics-enabled"`
	NodeGroup       *string `yaml:"node-group"`
	UploadNodeCount *int    `yaml:"upload-node-count"`
	Seed            *int64  `yaml:"seed"`
}

type gcOptions struct {
	NodeGroup        *string `yaml:"node-group"`
	Seed             *int64  `yaml:"seed"`
	StoreSize        *int    `yaml:"store-size"`
	StoreSizeDivisor *int    `yaml:"store-size-divisor"`
	Wait             *int    `yaml:"wait"`
}

type kademliaOptions struct {
	Dynamic *bool `yaml:"dynamic"`
}

type localpinningOptions struct {
	Mode             *string `yaml:"mode"`
	NodeGroup        *string `yaml:"node-group"`
	Seed             *int64  `yaml:"seed"`
	StoreSize        *int    `yaml:"store-size"`
	StoreSizeDivisor *int    `yaml:"store-size-divisor"`
}

type manifestOptions struct {
	NodeGroup         *string `yaml:"node-group"`
	FilesInCollection *int    `yaml:"files-in-collection"`
	MaxPathNameLength *int32  `yaml:"max-path-name-length"`
	Seed              *int64  `yaml:"seed"`
}

type peercountOptions struct {
	MetricsEnabled *bool  `yaml:"metrics-enabled"`
	Seed           *int64 `yaml:"seed"`
}

type pingOptions struct {
	MetricsEnabled *bool `yaml:"metrics-enabled"`
}

type pssOptions struct {
	NodeGroup      *string        `yaml:"node-group"`
	NodeCount      *int           `yaml:"node-count"`
	RequestTimeout *time.Duration `yaml:"request-timeout"`
	AddressPrefix  *int           `yaml:"address-prefix"`
	Seed           *int64         `yaml:"seed"`
}

type pullSyncOptions struct {
	NodeGroup                  *string `yaml:"node-group"`
	UploadNodeCount            *int    `yaml:"upload-node-count"`
	ChunksPerNode              *int    `yaml:"chunks-per-node"`
	ReplicationFactorThreshold *int    `yaml:"replication-factor-threshold"`
	Seed                       *int64  `yaml:"seed"`
}

type pushSyncOptions struct {
	NodeGroup       *string        `yaml:"node-group"`
	UploadNodeCount *int           `yaml:"upload-node-count"`
	ChunksPerNode   *int           `yaml:"chunks-per-node"`
	FilesPerNode    *int           `yaml:"files-per-node"`
	FileSize        *int64         `yaml:"file-size"`
	Retries         *int           `yaml:"retries"`
	RetryDelay      *time.Duration `yaml:"retry-delay"`
	Seed            *int64         `yaml:"seed"`
}

type retrievalOptions struct {
	NodeGroup       *string `yaml:"node-group"`
	UploadNodeCount *int    `yaml:"upload-node-count"`
	ChunksPerNode   *int    `yaml:"chunks-per-node"`
	Seed            *int64  `yaml:"seed"`
}

type settlementsOptions struct {
	NodeGroup          *string `yaml:"node-group"`
	UploadNodeCount    *int    `yaml:"upload-node-count"`
	FileName           *string `yaml:"file-name"`
	FileSize           *int64  `yaml:"file-size"`
	Seed               *int64  `yaml:"seed"`
	Threshold          *int64  `yaml:"threshold"`
	WaitBeforeDownload *int    `yaml:"wait-before-download"`
	ExpectSettlements  *int    `yaml:"expect-settlements"`
}

type socOptions struct {
	NodeGroup *string `yaml:"node-group"`
	Seed      *int64  `yaml:"seed"`
}
