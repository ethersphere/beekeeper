package config

import (
	"fmt"
	"reflect"
	"time"

	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/check/balances"
	"github.com/ethersphere/beekeeper/pkg/check/cashout"
	"github.com/ethersphere/beekeeper/pkg/check/chunkrepair"
	"github.com/ethersphere/beekeeper/pkg/check/fileretrieval"
	"github.com/ethersphere/beekeeper/pkg/check/fullconnectivity"
	"github.com/ethersphere/beekeeper/pkg/check/gc"
	"github.com/ethersphere/beekeeper/pkg/check/kademlia"
	"github.com/ethersphere/beekeeper/pkg/check/manifest"
	"github.com/ethersphere/beekeeper/pkg/check/peercount"
	"github.com/ethersphere/beekeeper/pkg/check/pingpong"
	"github.com/ethersphere/beekeeper/pkg/check/pss"
	"github.com/ethersphere/beekeeper/pkg/check/pullsync"
	"github.com/ethersphere/beekeeper/pkg/check/pushsync"
	"github.com/ethersphere/beekeeper/pkg/check/retrieval"
	"github.com/ethersphere/beekeeper/pkg/check/settlements"
	"github.com/ethersphere/beekeeper/pkg/check/smoke"
	"github.com/ethersphere/beekeeper/pkg/check/soc"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
	"gopkg.in/yaml.v3"
)

type Check struct {
	Options yaml.Node      `yaml:"options"`
	Timeout *time.Duration `yaml:"timeout"`
	Type    string         `yaml:"type"`
}

type CheckType struct {
	NewAction  func() beekeeper.Action
	NewOptions func(CheckGlobalConfig, Check) (interface{}, error)
}

type CheckGlobalConfig struct {
	MetricsEnabled bool
	MetricsPusher  *push.Pusher
	Seed           int64
}

var Checks = map[string]CheckType{
	"balances": {
		NewAction: balances.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			checkOpts := new(struct {
				DryRun             *bool          `yaml:"dry-run"`
				FileName           *string        `yaml:"file-name"`
				FileSize           *int64         `yaml:"file-size"`
				PostageAmount      *int64         `yaml:"postage-amount"`
				PostageWait        *time.Duration `yaml:"postage-wait"`
				Seed               *int64         `yaml:"seed"`
				UploadNodeCount    *int           `yaml:"upload-node-count"`
				WaitBeforeDownload *int           `yaml:"wait-before-download"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := balances.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"cashout": {
		NewAction: cashout.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			checkOpts := new(struct {
				NodeGroup *string `yaml:"node-group"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := cashout.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"chunk-repair": {
		NewAction: chunkrepair.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			checkOpts := new(struct {
				MetricsEnabled         *bool   `yaml:"metrics-enabled"`
				NodeGroup              *string `yaml:"node-group"`
				NumberOfChunksToRepair *int    `yaml:"number-of-chunks-to-repair"`
				Seed                   *int64  `yaml:"seed"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := chunkrepair.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"file-retrieval": {
		NewAction: fileretrieval.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			checkOpts := new(struct {
				FileName        *string        `yaml:"file-name"`
				FileSize        *int64         `yaml:"file-size"`
				FilesPerNode    *int           `yaml:"files-per-node"`
				Full            *bool          `yaml:"full"`
				MetricsEnabled  *bool          `yaml:"metrics-enabled"`
				NodeGroup       *string        `yaml:"node-group"`
				PostageAmount   *int64         `yaml:"postage-amount"`
				PostageWait     *time.Duration `yaml:"postage-wait"`
				Seed            *int64         `yaml:"seed"`
				UploadNodeCount *int           `yaml:"upload-node-count"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := fileretrieval.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"full-connectivity": {
		NewAction: fullconnectivity.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			return nil, nil
		},
	},
	"gc": {
		NewAction: gc.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			checkOpts := new(struct {
				CacheSize     *int           `yaml:"cache-size"`
				Seed          *int64         `yaml:"seed"`
				PostageAmount *int64         `yaml:"postage-amount"`
				PostageWait   *time.Duration `yaml:"postage-wait"`
				ReserveSize   *int           `yaml:"reserve-size"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := gc.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"kademlia": {
		NewAction: kademlia.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			checkOpts := new(struct {
				Dynamic *bool `yaml:"dynamic"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := kademlia.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"manifest": {
		NewAction: manifest.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			checkOpts := new(struct {
				FilesInCollection *int           `yaml:"files-in-collection"`
				MaxPathnameLength *int32         `yaml:"max-pathname-length"`
				PostageAmount     *int64         `yaml:"postage-amount"`
				PostageDepth      *uint64        `yaml:"postage-depth"`
				PostageWait       *time.Duration `yaml:"postage-wait"`
				Seed              *int64         `yaml:"seed"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := manifest.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"peer-count": {
		NewAction: peercount.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			return nil, nil
		},
	},
	"pingpong": {
		NewAction: pingpong.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			checkOpts := new(struct {
				MetricsEnabled *bool `yaml:"metrics-enabled"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := pingpong.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"pss": {
		NewAction: pss.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			checkOpts := new(struct {
				AddressPrefix  *int           `yaml:"address-prefix"`
				MetricsEnabled *bool          `yaml:"metrics-enabled"`
				NodeCount      *int           `yaml:"node-count"`
				PostageAmount  *int64         `yaml:"postage-amount"`
				PostageDepth   *uint64        `yaml:"postage-depth"`
				PostageWait    *time.Duration `yaml:"postage-wait"`
				RequestTimeout *time.Duration `yaml:"request-timeout"`
				Seed           *int64         `yaml:"seed"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := pss.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"pullsync": {
		NewAction: pullsync.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			checkOpts := new(struct {
				ChunksPerNode              *int           `yaml:"chunks-per-node"`
				PostageAmount              *int64         `yaml:"postage-amount"`
				PostageWait                *time.Duration `yaml:"postage-wait"`
				ReplicationFactorThreshold *int           `yaml:"replication-factor-threshold"`
				Seed                       *int64         `yaml:"seed"`
				UploadNodeCount            *int           `yaml:"upload-node-count"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := pullsync.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"pushsync": {
		NewAction: pushsync.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			checkOpts := new(struct {
				ChunksPerNode   *int           `yaml:"chunks-per-node"`
				MetricsEnabled  *bool          `yaml:"metrics-enabled"`
				Mode            *string        `yaml:"mode"`
				PostageAmount   *int64         `yaml:"postage-amount"`
				PostageDepth    *uint64        `yaml:"postage-depth"`
				PostageWait     *time.Duration `yaml:"postage-wait"`
				Retries         *int           `yaml:"retries"`
				RetryDelay      *time.Duration `yaml:"retry-delay"`
				Seed            *int64         `yaml:"seed"`
				UploadNodeCount *int           `yaml:"upload-node-count"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := pushsync.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"retrieval": {
		NewAction: retrieval.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			checkOpts := new(struct {
				ChunksPerNode   *int           `yaml:"chunks-per-node"`
				MetricsEnabled  *bool          `yaml:"metrics-enabled"`
				PostageAmount   *int64         `yaml:"postage-amount"`
				PostageDepth    *uint64        `yaml:"postage-depth"`
				PostageWait     *time.Duration `yaml:"postage-wait"`
				Seed            *int64         `yaml:"seed"`
				UploadNodeCount *int           `yaml:"upload-node-count"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := retrieval.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"settlements": {
		NewAction: settlements.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			checkOpts := new(struct {
				DryRun             *bool          `yaml:"dry-run"`
				ExpectSettlements  *bool          `yaml:"expect-settlements"`
				FileName           *string        `yaml:"file-name"`
				FileSize           *int64         `yaml:"file-size"`
				PostageAmount      *int64         `yaml:"postage-amount"`
				PostageDepth       *uint64        `yaml:"postage-depth"`
				PostageWait        *time.Duration `yaml:"postage-wait"`
				Seed               *int64         `yaml:"seed"`
				Threshold          *int64         `yaml:"threshold"`
				UploadNodeCount    *int           `yaml:"upload-node-count"`
				WaitBeforeDownload *int           `yaml:"wait-before-download"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := settlements.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"smoke": {
		NewAction: smoke.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			checkOpts := new(struct {
				Bytes           *int           `yaml:"bytes"`
				NodeGroup       *string        `yaml:"node-group"`
				Runs            *int           `yaml:"runs"`
				Seed            *int64         `yaml:"seed"`
				Timeout         *time.Duration `yaml:"timeout"`
				UploadNodeCount *int           `yaml:"upload-node-count"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := smoke.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"soc": {
		NewAction: soc.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			checkOpts := new(struct {
				PostageAmount  *int64         `yaml:"postage-amount"`
				PostageDepth   *uint64        `yaml:"postage-depth"`
				PostageWait    *time.Duration `yaml:"postage-wait"`
				RequestTimeout *time.Duration `yaml:"request-timeout"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := soc.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
}

func applyCheckConfig(global CheckGlobalConfig, local, opts interface{}) (err error) {
	lv := reflect.ValueOf(local).Elem()
	lt := reflect.TypeOf(local).Elem()
	ov := reflect.Indirect(reflect.ValueOf(opts).Elem())
	ot := reflect.TypeOf(opts).Elem()

	for i := 0; i < lv.NumField(); i++ {
		fieldName := lt.Field(i).Name
		switch fieldName {
		case "MetricsEnabled":
			// if (set globally) || (set locally)
			if (lv.Field(i).IsNil() && global.MetricsEnabled) || (!lv.Field(i).IsNil() && lv.FieldByName(fieldName).Elem().Bool()) {
				if global.MetricsPusher == nil {
					return fmt.Errorf("metrics pusher is nil (not set)")
				}
				v := reflect.ValueOf(global.MetricsPusher)
				ov.FieldByName("MetricsPusher").Set(v)
			}
		case "Seed":
			if lv.Field(i).IsNil() { // set globally
				if global.Seed >= 0 {
					v := reflect.ValueOf(global.Seed)
					ov.FieldByName(fieldName).Set(v)
				} else {
					v := reflect.ValueOf(random.Int64())
					ov.FieldByName(fieldName).Set(v)
				}
			} else { // set locally
				fieldType := lt.Field(i).Type
				fieldValue := lv.FieldByName(fieldName).Elem()
				ft, ok := ot.FieldByName(fieldName)
				if ok && fieldType.Elem().AssignableTo(ft.Type) {
					ov.FieldByName(fieldName).Set(fieldValue)
				}
			}
		default:
			if lv.Field(i).IsNil() {
				fmt.Printf("field %s not set, using default value\n", fieldName)
			} else {
				fieldType := lt.Field(i).Type
				fieldValue := lv.FieldByName(fieldName).Elem()
				ft, ok := ot.FieldByName(fieldName)
				if ok && fieldType.Elem().AssignableTo(ft.Type) {
					ov.FieldByName(fieldName).Set(fieldValue)
				}
			}
		}
	}

	return
}
