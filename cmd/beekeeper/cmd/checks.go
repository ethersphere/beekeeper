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
			if o.Seed != nil {
				opts.Seed = *o.Seed
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
			if o.Seed != nil {
				opts.Seed = *o.Seed
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
			if o.Seed != nil {
				opts.Seed = *o.Seed
			}
			return opts, nil
		},
	},
	"full-connectivity": {
		NewCheck: fullconnectivity.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(fullConnectivityOptions)
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			var opts fullconnectivity.Options
			if o.Seed != nil {
				opts.Seed = *o.Seed
			}
			return opts, nil
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
			if o.Seed != nil {
				opts.Seed = *o.Seed
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
			if o.Seed != nil {
				opts.Seed = *o.Seed
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
			if o.Seed != nil {
				opts.Seed = *o.Seed
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
			if o.Seed != nil {
				opts.Seed = *o.Seed
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
			if o.Seed != nil {
				opts.Seed = *o.Seed
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
			if o.Seed != nil {
				opts.Seed = *o.Seed
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
			if o.Seed != nil {
				opts.Seed = *o.Seed
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
			if o.Seed != nil {
				opts.Seed = *o.Seed
			}
			if o.MetricsEnabled != nil && *o.MetricsEnabled {
				// TODO: make pusher and set it to opts.MetricsPusher
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
			if o.Seed != nil {
				opts.Seed = *o.Seed
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
			if o.Seed != nil {
				opts.Seed = *o.Seed
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
			if o.Seed != nil {
				opts.Seed = *o.Seed
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
			if o.Seed != nil {
				opts.Seed = *o.Seed
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
			if o.Seed != nil {
				opts.Seed = *o.Seed
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
			if o.Seed != nil {
				opts.Seed = *o.Seed
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
	NodeGroup          *string `yaml:"node-group"`
	UploadNodeCount    *int    `yaml:"upload-node-count"`
	FileName           *string `yaml:"file-name"`
	FileSize           *int64  `yaml:"file-size"`
	Seed               *int64  `yaml:"seed"`
	WaitBeforeDownload *int    `yaml:"wait-before-download"`
}

type chunkRepairOptions struct {
	NodeGroup              *string `yaml:"node-group"`
	NumberOfChunksToRepair *int    `yaml:"number-of-chunks-to-repair"`
	Seed                   *int64  `yaml:"seed"`
}

type fileRetrievalOptions struct {
	NodeGroup       *string `yaml:"node-group"`
	UploadNodeCount *int    `yaml:"upload-node-count"`
	FilesPerNode    *int    `yaml:"files-per-node"`
	FileName        *string `yaml:"file-name"`
	FileSize        *int64  `yaml:"file-size"`
	MetricsEnabled  *bool   `yaml:"metrics-enabled"`
	Seed            *int64  `yaml:"seed"`
}

type fullConnectivityOptions struct {
	MetricsEnabled *bool  `yaml:"metrics-enabled"`
	Seed           *int64 `yaml:"seed"`
}

type gcOptions struct {
	NodeGroup        *string `yaml:"node-group"`
	StoreSize        *int    `yaml:"store-size"`
	StoreSizeDivisor *int    `yaml:"store-size-divisor"`
	Seed             *int64  `yaml:"seed"`
	Wait             *int    `yaml:"wait"`
}

type kademliaOptions struct {
	MetricsEnabled *bool  `yaml:"metrics-enabled"`
	Seed           *int64 `yaml:"seed"`
}

type localpinningOptions struct {
	NodeGroup        *string `yaml:"node-group"`
	StoreSize        *int    `yaml:"store-size"`
	StoreSizeDivisor *int    `yaml:"store-size-divisor"`
	Seed             *int64  `yaml:"seed"`
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
	MetricsEnabled *bool  `yaml:"metrics-enabled"`
	Seed           *int64 `yaml:"seed"`
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
