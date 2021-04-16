package cmd

import (
	"fmt"
	"reflect"
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
	"github.com/ethersphere/beekeeper/pkg/check/pingpong"
	"github.com/ethersphere/beekeeper/pkg/check/pss"
	"github.com/ethersphere/beekeeper/pkg/check/pullsync"
	"github.com/ethersphere/beekeeper/pkg/check/pushsync"
	"github.com/ethersphere/beekeeper/pkg/check/retrieval"
	"github.com/ethersphere/beekeeper/pkg/check/settlements"
	"github.com/ethersphere/beekeeper/pkg/check/soc"
	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/prometheus/client_golang/prometheus/push"
)

type Check struct {
	NewCheck   func() check.Check
	NewOptions func(cfg *config.Config, checkProfile config.Check) (interface{}, error)
}

func applyOptions(new, old interface{}) (err error) {
	fmt.Println("applying options")
	nv := reflect.ValueOf(new).Elem()
	nt := reflect.TypeOf(new).Elem()
	ov := reflect.ValueOf(old)
	ot := reflect.TypeOf(old)

	for i := 0; i < nv.NumField(); i++ {
		if !nv.Field(i).IsNil() {
			fieldName := nt.Field(i).Name
			fieldValue := nv.Field(i).Elem()
			fmt.Println("A", i, fieldName, nt.Field(i).Type, fieldValue)
			ft, ok := ot.FieldByName(fieldName)
			if ok && nt.Field(i).Type.Elem().AssignableTo(ft.Type) {
				fv := ov.FieldByName(fieldName)
				fmt.Println("B", i, fieldName, ft.Type, fv)
				// fv.Set(fieldValue)
			}
		}
	}

	return
}

var Checks = map[string]Check{
	"balances": {
		NewCheck: balances.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(struct {
				DryRun             *bool   `yaml:"dry-run"`
				FileName           *string `yaml:"file-name"`
				FileSize           *int64  `yaml:"file-size"`
				NodeGroup          *string `yaml:"node-group"`
				Seed               *int64  `yaml:"seed"`
				UploadNodeCount    *int    `yaml:"upload-node-count"`
				WaitBeforeDownload *int    `yaml:"wait-before-download"`
			})
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			opts := balances.DefaultOptions

			// set seed
			if o.Seed == nil && cfg.RunProfiles[cfg.Execute].Seed > 0 { // set globaly
				opts.Seed = cfg.RunProfiles[cfg.Execute].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // set localy
				opts.Seed = *o.Seed
			}
			// if o.DryRun != nil {
			// 	opts.DryRun = *o.DryRun
			// }
			// if o.FileName != nil {
			// 	opts.FileName = *o.FileName
			// }
			// if o.FileSize != nil {
			// 	opts.FileSize = *o.FileSize
			// }
			// if o.NodeGroup != nil {
			// 	opts.NodeGroup = *o.NodeGroup
			// }
			// if o.UploadNodeCount != nil {
			// 	opts.UploadNodeCount = *o.UploadNodeCount
			// }
			// if o.WaitBeforeDownload != nil {
			// 	opts.WaitBeforeDownload = *o.WaitBeforeDownload
			// }
			if err := applyOptions(o, opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}
			return opts, nil
		},
	},
	"chunk-repair": {
		NewCheck: chunkrepair.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(struct {
				MetricsEnabled         *bool   `yaml:"metrics-enabled"`
				NodeGroup              *string `yaml:"node-group"`
				NumberOfChunksToRepair *int    `yaml:"number-of-chunks-to-repair"`
				Seed                   *int64  `yaml:"seed"`
			})
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			opts := chunkrepair.DefaultOptions

			// set seed
			if o.Seed == nil && cfg.RunProfiles[cfg.Execute].Seed > 0 { // set globaly
				opts.Seed = cfg.RunProfiles[cfg.Execute].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // set localy
				opts.Seed = *o.Seed
			}
			// TODO: resolve optionNamePushGateway
			// set metrics
			if o.MetricsEnabled == nil && cfg.RunProfiles[cfg.Execute].MetricsEnabled { // set globaly
				opts.MetricsPusher = push.New("optionNamePushGateway", cfg.Cluster.Namespace)
			} else if o.MetricsEnabled != nil && *o.MetricsEnabled { // set localy
				opts.MetricsPusher = push.New("optionNamePushGateway", cfg.Cluster.Namespace)
			}
			if o.NodeGroup != nil {
				opts.NodeGroup = *o.NodeGroup
			}
			if o.NumberOfChunksToRepair != nil {
				opts.NumberOfChunksToRepair = *o.NumberOfChunksToRepair
			}
			return opts, nil
		},
	},
	"file-retrieval": {
		NewCheck: fileretrieval.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(struct {
				FileName        *string `yaml:"file-name"`
				FileSize        *int64  `yaml:"file-size"`
				FilesPerNode    *int    `yaml:"files-per-node"`
				Full            *bool   `yaml:"full"`
				MetricsEnabled  *bool   `yaml:"metrics-enabled"`
				NodeGroup       *string `yaml:"node-group"`
				Seed            *int64  `yaml:"seed"`
				UploadNodeCount *int    `yaml:"upload-node-count"`
			})
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			opts := fileretrieval.DefaultOptions

			// set seed
			if o.Seed == nil && cfg.RunProfiles[cfg.Execute].Seed > 0 { // set globaly
				opts.Seed = cfg.RunProfiles[cfg.Execute].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // set localy
				opts.Seed = *o.Seed
			}
			// TODO: resolve optionNamePushGateway
			// set metrics
			if o.MetricsEnabled == nil && cfg.RunProfiles[cfg.Execute].MetricsEnabled { // set globaly
				opts.MetricsPusher = push.New("optionNamePushGateway", cfg.Cluster.Namespace)
			} else if o.MetricsEnabled != nil && *o.MetricsEnabled { // set localy
				opts.MetricsPusher = push.New("optionNamePushGateway", cfg.Cluster.Namespace)
			}
			if o.FileName != nil {
				opts.FileName = *o.FileName
			}
			if o.FileSize != nil {
				opts.FileSize = *o.FileSize
			}
			if o.FilesPerNode != nil {
				opts.FilesPerNode = *o.FilesPerNode
			}
			if o.Full != nil {
				opts.Full = *o.Full
			}
			if o.NodeGroup != nil {
				opts.NodeGroup = *o.NodeGroup
			}
			if o.UploadNodeCount != nil {
				opts.UploadNodeCount = *o.UploadNodeCount
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
			o := new(struct {
				NodeGroup        *string `yaml:"node-group"`
				Seed             *int64  `yaml:"seed"`
				StoreSize        *int    `yaml:"store-size"`
				StoreSizeDivisor *int    `yaml:"store-size-divisor"`
				Wait             *int    `yaml:"wait"`
			})
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			opts := gc.DefaultOptions

			// set seed
			if o.Seed == nil && cfg.RunProfiles[cfg.Execute].Seed > 0 { // set globaly
				opts.Seed = cfg.RunProfiles[cfg.Execute].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // set localy
				opts.Seed = *o.Seed
			}
			if o.NodeGroup != nil {
				opts.NodeGroup = *o.NodeGroup
			}
			if o.StoreSize != nil {
				opts.StoreSize = *o.StoreSize
			}
			if o.StoreSizeDivisor != nil {
				opts.StoreSizeDivisor = *o.StoreSizeDivisor
			}
			if o.Wait != nil {
				opts.Wait = *o.Wait
			}
			return opts, nil
		},
	},
	"kademlia": {
		NewCheck: kademlia.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(struct {
				Dynamic *bool `yaml:"dynamic"`
			})
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			opts := kademlia.DefaultOptions
			if o.Dynamic != nil {
				opts.Dynamic = *o.Dynamic
			}

			return opts, nil
		},
	},
	"local-pinning": {
		NewCheck: localpinning.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(struct {
				Mode             *string `yaml:"mode"`
				NodeGroup        *string `yaml:"node-group"`
				Seed             *int64  `yaml:"seed"`
				StoreSize        *int    `yaml:"store-size"`
				StoreSizeDivisor *int    `yaml:"store-size-divisor"`
			})
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			opts := localpinning.DefaultOptions

			// set seed
			if o.Seed == nil && cfg.RunProfiles[cfg.Execute].Seed > 0 { // enabled globaly
				opts.Seed = cfg.RunProfiles[cfg.Execute].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // enabled localy
				opts.Seed = *o.Seed
			}
			if o.Mode != nil {
				opts.Mode = *o.Mode
			}
			if o.NodeGroup != nil {
				opts.NodeGroup = *o.NodeGroup
			}
			if o.StoreSize != nil {
				opts.StoreSize = *o.StoreSize
			}
			if o.StoreSizeDivisor != nil {
				opts.StoreSizeDivisor = *o.StoreSizeDivisor
			}
			return opts, nil
		},
	},
	"manifest": {
		NewCheck: manifest.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(struct {
				FilesInCollection *int    `yaml:"files-in-collection"`
				MaxPathnameLength *int32  `yaml:"max-pathname-length"`
				NodeGroup         *string `yaml:"node-group"`
				Seed              *int64  `yaml:"seed"`
			})
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			opts := manifest.DefaultOptions

			// set seed
			if o.Seed == nil && cfg.RunProfiles[cfg.Execute].Seed > 0 { // set globaly
				opts.Seed = cfg.RunProfiles[cfg.Execute].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // set localy
				opts.Seed = *o.Seed
			}
			if o.FilesInCollection != nil {
				opts.FilesInCollection = *o.FilesInCollection
			}
			if o.MaxPathnameLength != nil {
				opts.MaxPathnameLength = *o.MaxPathnameLength
			}
			if o.NodeGroup != nil {
				opts.NodeGroup = *o.NodeGroup
			}
			return opts, nil
		},
	},
	"peer-count": {
		NewCheck: peercount.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			return nil, nil
		},
	},
	"pingpong": {
		NewCheck: pingpong.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(struct {
				MetricsEnabled *bool `yaml:"metrics-enabled"`
			})
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			opts := pingpong.DefaultOptions

			// TODO: resolve optionNamePushGateway
			// set metrics
			if o.MetricsEnabled == nil && cfg.RunProfiles[cfg.Execute].MetricsEnabled { // set globaly
				opts.MetricsPusher = push.New("optionNamePushGateway", cfg.Cluster.Namespace)
			} else if o.MetricsEnabled != nil && *o.MetricsEnabled { // set localy
				opts.MetricsPusher = push.New("optionNamePushGateway", cfg.Cluster.Namespace)
			}
			return opts, nil
		},
	},
	"pss": {
		NewCheck: pss.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(struct {
				AddressPrefix  *int           `yaml:"address-prefix"`
				MetricsEnabled *bool          `yaml:"metrics-enabled"`
				NodeCount      *int           `yaml:"node-count"`
				NodeGroup      *string        `yaml:"node-group"`
				RequestTimeout *time.Duration `yaml:"request-timeout"`
				Seed           *int64         `yaml:"seed"`
			})
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			opts := pss.DefaultOptions

			// set seed
			if o.Seed == nil && cfg.RunProfiles[cfg.Execute].Seed > 0 { // set globaly
				opts.Seed = cfg.RunProfiles[cfg.Execute].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // set localy
				opts.Seed = *o.Seed
			}
			// TODO: resolve optionNamePushGateway
			// set metrics
			if o.MetricsEnabled == nil && cfg.RunProfiles[cfg.Execute].MetricsEnabled { // set globaly
				opts.MetricsPusher = push.New("optionNamePushGateway", cfg.Cluster.Namespace)
			} else if o.MetricsEnabled != nil && *o.MetricsEnabled { // set localy
				opts.MetricsPusher = push.New("optionNamePushGateway", cfg.Cluster.Namespace)
			}
			if o.AddressPrefix != nil {
				opts.AddressPrefix = *o.AddressPrefix
			}
			if o.NodeCount != nil { // TODO: check what this option represent
				opts.NodeCount = *o.NodeCount
			}
			if o.NodeGroup != nil {
				opts.NodeGroup = *o.NodeGroup
			}
			if o.RequestTimeout != nil {
				opts.RequestTimeout = *o.RequestTimeout
			}
			return opts, nil
		},
	},
	"pullsync": {
		NewCheck: pullsync.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(struct {
				ChunksPerNode              *int    `yaml:"chunks-per-node"`
				NodeGroup                  *string `yaml:"node-group"`
				ReplicationFactorThreshold *int    `yaml:"replication-factor-threshold"`
				Seed                       *int64  `yaml:"seed"`
				UploadNodeCount            *int    `yaml:"upload-node-count"`
			})
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			opts := pullsync.DefaultOptions

			// set seed
			if o.Seed == nil && cfg.RunProfiles[cfg.Execute].Seed > 0 { // set globaly
				opts.Seed = cfg.RunProfiles[cfg.Execute].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // set localy
				opts.Seed = *o.Seed
			}
			if o.ChunksPerNode != nil {
				opts.ChunksPerNode = *o.ChunksPerNode
			}
			if o.NodeGroup != nil {
				opts.NodeGroup = *o.NodeGroup
			}
			if o.ReplicationFactorThreshold != nil {
				opts.ReplicationFactorThreshold = *o.ReplicationFactorThreshold
			}
			if o.UploadNodeCount != nil {
				opts.UploadNodeCount = *o.UploadNodeCount
			}
			return opts, nil
		},
	},
	"pushsync": {
		NewCheck: pushsync.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(struct {
				ChunksPerNode   *int           `yaml:"chunks-per-node"`
				FileSize        *int64         `yaml:"file-size"`
				FilesPerNode    *int           `yaml:"files-per-node"`
				MetricsEnabled  *bool          `yaml:"metrics-enabled"`
				Mode            *string        `yaml:"mode"`
				NodeGroup       *string        `yaml:"node-group"`
				Retries         *int           `yaml:"retries"`
				RetryDelay      *time.Duration `yaml:"retry-delay"`
				Seed            *int64         `yaml:"seed"`
				UploadNodeCount *int           `yaml:"upload-node-count"`
			})
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			opts := pushsync.DefaultOptions

			// set seed
			if o.Seed == nil && cfg.RunProfiles[cfg.Execute].Seed > 0 { // enabled globaly
				opts.Seed = cfg.RunProfiles[cfg.Execute].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // enabled localy
				opts.Seed = *o.Seed
			}
			// TODO: resolve optionNamePushGateway
			// set metrics
			if o.MetricsEnabled == nil && cfg.RunProfiles[cfg.Execute].MetricsEnabled { // enabled globaly
				opts.MetricsPusher = push.New("optionNamePushGateway", cfg.Cluster.Namespace)
			} else if o.MetricsEnabled != nil && *o.MetricsEnabled { // enabled localy
				opts.MetricsPusher = push.New("optionNamePushGateway", cfg.Cluster.Namespace)
			}
			if o.ChunksPerNode != nil {
				opts.ChunksPerNode = *o.ChunksPerNode
			}
			if o.FileSize != nil {
				opts.FileSize = *o.FileSize
			}
			if o.FilesPerNode != nil {
				opts.FilesPerNode = *o.FilesPerNode
			}
			if o.Mode != nil {
				opts.Mode = *o.Mode
			}
			if o.NodeGroup != nil {
				opts.NodeGroup = *o.NodeGroup
			}
			if o.Retries != nil {
				opts.Retries = *o.Retries
			}
			if o.RetryDelay != nil {
				opts.RetryDelay = *o.RetryDelay
			}
			if o.UploadNodeCount != nil {
				opts.UploadNodeCount = *o.UploadNodeCount
			}
			return opts, nil
		},
	},
	"retrieval": {
		NewCheck: retrieval.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(struct {
				ChunksPerNode   *int    `yaml:"chunks-per-node"`
				MetricsEnabled  *bool   `yaml:"metrics-enabled"`
				NodeGroup       *string `yaml:"node-group"`
				Seed            *int64  `yaml:"seed"`
				UploadNodeCount *int    `yaml:"upload-node-count"`
			})
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			opts := retrieval.DefaultOptions

			// set seed
			if o.Seed == nil && cfg.RunProfiles[cfg.Execute].Seed > 0 { // enabled globaly
				opts.Seed = cfg.RunProfiles[cfg.Execute].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // enabled localy
				opts.Seed = *o.Seed
			}
			// TODO: resolve optionNamePushGateway
			// set metrics
			if o.MetricsEnabled == nil && cfg.RunProfiles[cfg.Execute].MetricsEnabled { // enabled globaly
				opts.MetricsPusher = push.New("optionNamePushGateway", cfg.Cluster.Namespace)
			} else if o.MetricsEnabled != nil && *o.MetricsEnabled { // enabled localy
				opts.MetricsPusher = push.New("optionNamePushGateway", cfg.Cluster.Namespace)
			}
			if o.ChunksPerNode != nil {
				opts.ChunksPerNode = *o.ChunksPerNode
			}
			if o.NodeGroup != nil {
				opts.NodeGroup = *o.NodeGroup
			}
			if o.UploadNodeCount != nil {
				opts.UploadNodeCount = *o.UploadNodeCount
			}
			return opts, nil
		},
	},
	"settlements": {
		NewCheck: settlements.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(struct {
				DryRun             *bool   `yaml:"dry-run"`
				ExpectSettlements  *bool   `yaml:"expect-settlements"`
				FileName           *string `yaml:"file-name"`
				FileSize           *int64  `yaml:"file-size"`
				NodeGroup          *string `yaml:"node-group"`
				Seed               *int64  `yaml:"seed"`
				Threshold          *int64  `yaml:"threshold"`
				UploadNodeCount    *int    `yaml:"upload-node-count"`
				WaitBeforeDownload *int    `yaml:"wait-before-download"`
			})
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			opts := settlements.DefaultOptions

			// set seed
			if o.Seed == nil && cfg.RunProfiles[cfg.Execute].Seed > 0 { // enabled globaly
				opts.Seed = cfg.RunProfiles[cfg.Execute].Seed
			} else if o.Seed != nil && *o.Seed > 0 { // enabled localy
				opts.Seed = *o.Seed
			}
			if o.DryRun != nil {
				opts.DryRun = *o.DryRun
			}
			if o.ExpectSettlements != nil {
				opts.ExpectSettlements = *o.ExpectSettlements
			}
			if o.FileName != nil {
				opts.FileName = *o.FileName
			}
			if o.FileSize != nil {
				opts.FileSize = *o.FileSize
			}
			if o.NodeGroup != nil {
				opts.NodeGroup = *o.NodeGroup
			}
			if o.Threshold != nil {
				opts.Threshold = *o.Threshold
			}
			if o.UploadNodeCount != nil {
				opts.UploadNodeCount = *o.UploadNodeCount
			}
			if o.WaitBeforeDownload != nil {
				opts.WaitBeforeDownload = *o.WaitBeforeDownload
			}
			return opts, nil
		},
	},
	"soc": {
		NewCheck: soc.NewCheck,
		NewOptions: func(cfg *config.Config, checkProfile config.Check) (interface{}, error) {
			o := new(struct {
				NodeGroup *string `yaml:"node-group"`
			})
			if err := checkProfile.Options.Decode(o); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", checkProfile.Name, err)
			}
			opts := soc.DefaultOptions
			if o.NodeGroup != nil {
				opts.NodeGroup = *o.NodeGroup
			}
			return opts, nil
		},
	},
}
