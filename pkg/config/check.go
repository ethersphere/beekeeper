package config

import (
	"fmt"
	"reflect"
	"time"

	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/check/authenticated"
	"github.com/ethersphere/beekeeper/pkg/check/balances"
	"github.com/ethersphere/beekeeper/pkg/check/cashout"
	"github.com/ethersphere/beekeeper/pkg/check/chunkrepair"
	"github.com/ethersphere/beekeeper/pkg/check/contentavailability"
	"github.com/ethersphere/beekeeper/pkg/check/fileretrieval"
	"github.com/ethersphere/beekeeper/pkg/check/fullconnectivity"
	"github.com/ethersphere/beekeeper/pkg/check/gc"
	"github.com/ethersphere/beekeeper/pkg/check/kademlia"
	"github.com/ethersphere/beekeeper/pkg/check/manifest"
	"github.com/ethersphere/beekeeper/pkg/check/peercount"
	"github.com/ethersphere/beekeeper/pkg/check/pingpong"
	"github.com/ethersphere/beekeeper/pkg/check/postage"
	"github.com/ethersphere/beekeeper/pkg/check/pss"
	"github.com/ethersphere/beekeeper/pkg/check/pullsync"
	"github.com/ethersphere/beekeeper/pkg/check/pushsync"
	"github.com/ethersphere/beekeeper/pkg/check/retrieval"
	"github.com/ethersphere/beekeeper/pkg/check/settlements"
	"github.com/ethersphere/beekeeper/pkg/check/smoke"
	"github.com/ethersphere/beekeeper/pkg/check/soc"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/random"
	"gopkg.in/yaml.v3"
)

// Check represents check configuration
type Check struct {
	Options yaml.Node      `yaml:"options"`
	Timeout *time.Duration `yaml:"timeout"`
	Type    string         `yaml:"type"`
}

// CheckType is used for linking beekeeper actions with check and it's proper options
type CheckType struct {
	NewAction  func(logging.Logger) beekeeper.Action               // links check with beekeeper action
	NewOptions func(CheckGlobalConfig, Check) (interface{}, error) // check options
}

// CheckGlobalConfig represents global configs for all checks
type CheckGlobalConfig struct {
	Seed int64
}

// Checks represents all available check types
var Checks = map[string]CheckType{
	"balances": {
		NewAction: balances.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			checkOpts := new(struct {
				DryRun             *bool          `yaml:"dry-run"`
				FileName           *string        `yaml:"file-name"`
				FileSize           *int64         `yaml:"file-size"`
				GasPrice           *string        `yaml:"gas-price"`
				PostageAmount      *int64         `yaml:"postage-amount"`
				PostageDepth       *uint64        `yaml:"postage-depth"`
				PostageLabel       *string        `yaml:"postage-label"`
				Seed               *int64         `yaml:"seed"`
				UploadNodeCount    *int           `yaml:"upload-node-count"`
				WaitBeforeDownload *time.Duration `yaml:"wait-before-download"`
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
				GasPrice               *string `yaml:"gas-price"`
				NodeGroup              *string `yaml:"node-group"`
				NumberOfChunksToRepair *int    `yaml:"number-of-chunks-to-repair"`
				PostageAmount          *int64  `yaml:"postage-amount"`
				PostageLabel           *string `yaml:"postage-label"`
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
				FileName        *string `yaml:"file-name"`
				FileSize        *int64  `yaml:"file-size"`
				FilesPerNode    *int    `yaml:"files-per-node"`
				Full            *bool   `yaml:"full"`
				GasPrice        *string `yaml:"gas-price"`
				PostageAmount   *int64  `yaml:"postage-amount"`
				PostageLabel    *string `yaml:"postage-label"`
				Seed            *int64  `yaml:"seed"`
				UploadNodeCount *int    `yaml:"upload-node-count"`
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
			checkOpts := new(struct {
				LightNodeNames *[]string `yaml:"group-1"`
				FullNodeNames  *[]string `yaml:"group-2"`
				BootNodeNames  *[]string `yaml:"boot-nodes"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := fullconnectivity.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"gc": {
		NewAction: gc.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			checkOpts := new(struct {
				CacheSize    *int    `yaml:"cache-size"`
				GasPrice     *string `yaml:"gas-price"`
				PostageLabel *string `yaml:"postage-label"`
				ReserveSize  *int    `yaml:"reserve-size"`
				Seed         *int64  `yaml:"seed"`
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
				FilesInCollection *int    `yaml:"files-in-collection"`
				GasPrice          *string `yaml:"gas-price"`
				MaxPathnameLength *int32  `yaml:"max-pathname-length"`
				PostageAmount     *int64  `yaml:"postage-amount"`
				PostageDepth      *uint64 `yaml:"postage-depth"`
				PostageLabel      *string `yaml:"postage-label"`
				Seed              *int64  `yaml:"seed"`
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
			opts := pingpong.NewDefaultOptions()
			return opts, nil
		},
	},
	"pss": {
		NewAction: pss.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			checkOpts := new(struct {
				Count          *int64         `yaml:"count"`
				AddressPrefix  *int           `yaml:"address-prefix"`
				GasPrice       *string        `yaml:"gas-price"`
				PostageAmount  *int64         `yaml:"postage-amount"`
				PostageDepth   *uint64        `yaml:"postage-depth"`
				PostageLabel   *string        `yaml:"postage-label"`
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
				ChunksPerNode              *int    `yaml:"chunks-per-node"`
				GasPrice                   *string `yaml:"gas-price"`
				PostageAmount              *int64  `yaml:"postage-amount"`
				PostageLabel               *string `yaml:"postage-label"`
				ReplicationFactorThreshold *int    `yaml:"replication-factor-threshold"`
				Seed                       *int64  `yaml:"seed"`
				UploadNodeCount            *int    `yaml:"upload-node-count"`
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
				ChunksPerNode     *int           `yaml:"chunks-per-node"`
				GasPrice          *string        `yaml:"gas-price"`
				Mode              *string        `yaml:"mode"`
				PostageAmount     *int64         `yaml:"postage-amount"`
				PostageDepth      *uint64        `yaml:"postage-depth"`
				PostageLabel      *string        `yaml:"postage-label"`
				Retries           *int           `yaml:"retries"`
				RetryDelay        *time.Duration `yaml:"retry-delay"`
				Seed              *int64         `yaml:"seed"`
				UploadNodeCount   *int           `yaml:"upload-node-count"`
				ExcludeNodeGroups *[]string      `yaml:"exclude-node-group"`
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
				ChunksPerNode   *int    `yaml:"chunks-per-node"`
				GasPrice        *string `yaml:"gas-price"`
				PostageAmount   *int64  `yaml:"postage-amount"`
				PostageDepth    *uint64 `yaml:"postage-depth"`
				PostageLabel    *string `yaml:"postage-label"`
				Seed            *int64  `yaml:"seed"`
				UploadNodeCount *int    `yaml:"upload-node-count"`
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
				GasPrice           *string        `yaml:"gas-price"`
				PostageAmount      *int64         `yaml:"postage-amount"`
				PostageDepth       *uint64        `yaml:"postage-depth"`
				PostageLabel       *string        `yaml:"postage-label"`
				Seed               *int64         `yaml:"seed"`
				Threshold          *int64         `yaml:"threshold"`
				UploadNodeCount    *int           `yaml:"upload-node-count"`
				WaitBeforeDownload *time.Duration `yaml:"wait-before-download"`
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
				ContentSize   *int64         `yaml:"content-size"`
				RndSeed       *int64         `yaml:"rnd-seed"`
				PostageAmount *int64         `yaml:"postage-amount"`
				PostageDepth  *uint64        `yaml:"postage-depth"`
				TxOnErrWait   *time.Duration `yaml:"tx-on-err-wait"`
				RxOnErrWait   *time.Duration `yaml:"rx-on-err-wait"`
				NodesSyncWait *time.Duration `yaml:"nodes-sync-wait"`
				Duration      *time.Duration `yaml:"duration"`
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
	"load": {
		NewAction: smoke.NewLoadCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			checkOpts := new(struct {
				ContentSize     *int64         `yaml:"content-size"`
				RndSeed         *int64         `yaml:"rnd-seed"`
				PostageAmount   *int64         `yaml:"postage-amount"`
				PostageDepth    *uint64        `yaml:"postage-depth"`
				GasPrice        *string        `yaml:"gas-price"`
				TxOnErrWait     *time.Duration `yaml:"tx-on-err-wait"`
				RxOnErrWait     *time.Duration `yaml:"rx-on-err-wait"`
				NodesSyncWait   *time.Duration `yaml:"nodes-sync-wait"`
				Duration        *time.Duration `yaml:"duration"`
				UploaderCount   *int           `yaml:"uploader-count"`
				UploadGroups    *[]string      `yaml:"upload-groups"`
				DownloaderCount *int           `yaml:"downloader-count"`
				DownloadGroups  *[]string      `yaml:"download-groups"`
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
				GasPrice       *string        `yaml:"gas-price"`
				PostageAmount  *int64         `yaml:"postage-amount"`
				PostageDepth   *uint64        `yaml:"postage-depth"`
				PostageLabel   *string        `yaml:"postage-label"`
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
	"content-availability": {
		NewAction: contentavailability.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			checkOpts := new(struct {
				ContentSize   *int64  `yaml:"content-size"`
				GasPrice      *string `yaml:"gas-price"`
				PostageAmount *int64  `yaml:"postage-amount"`
				PostageDepth  *uint64 `yaml:"postage-depth"`
				PostageLabel  *string `yaml:"postage-label"`
				Seed          *int64  `yaml:"seed"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := contentavailability.NewDefaultOptions()
			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}
			return opts, nil
		},
	},
	"postage": {
		NewAction: postage.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			checkOpts := new(struct {
				GasPrice           *string `yaml:"gas-price"`
				PostageAmount      *int64  `yaml:"postage-amount"`
				PostageTopupAmount *int64  `yaml:"postage-topup-amount"`
				PostageDepth       *uint64 `yaml:"postage-depth"`
				PostageNewDepth    *uint64 `yaml:"postage-new-depth"`
				PostageLabel       *string `yaml:"postage-label"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := postage.NewDefaultOptions()
			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}
			return opts, nil
		},
	},
	"authenticate": {
		NewAction: authenticated.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (interface{}, error) {
			checkOpts := new(struct {
				DryRun              *bool   `yaml:"dry-run"`
				Role                *string `yaml:"role"`
				AdminPassword       *string `yaml:"admin-password"`
				RestrictedGroupName *string `yaml:"restricted-group-name"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := authenticated.NewDefaultOptions()
			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}
			return opts, nil
		},
	},
}

// applyCheckConfig merges global and local options into default options
func applyCheckConfig(global CheckGlobalConfig, local, opts interface{}) (err error) {
	lv := reflect.ValueOf(local).Elem()
	lt := reflect.TypeOf(local).Elem()
	ov := reflect.Indirect(reflect.ValueOf(opts).Elem())
	ot := reflect.TypeOf(opts).Elem()

	for i := 0; i < lv.NumField(); i++ {
		fieldName := lt.Field(i).Name
		switch fieldName {
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
