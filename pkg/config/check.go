package config

import (
	"fmt"
	"math/big"
	"reflect"
	"time"

	beeRedundancy "github.com/ethersphere/bee/v2/pkg/file/redundancy"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/check/act"
	"github.com/ethersphere/beekeeper/pkg/check/balances"
	"github.com/ethersphere/beekeeper/pkg/check/cashout"
	"github.com/ethersphere/beekeeper/pkg/check/datadurability"
	"github.com/ethersphere/beekeeper/pkg/check/feed"
	"github.com/ethersphere/beekeeper/pkg/check/fileretrieval"
	"github.com/ethersphere/beekeeper/pkg/check/fullconnectivity"
	"github.com/ethersphere/beekeeper/pkg/check/gc"
	"github.com/ethersphere/beekeeper/pkg/check/gsoc"
	"github.com/ethersphere/beekeeper/pkg/check/kademlia"
	"github.com/ethersphere/beekeeper/pkg/check/load"
	"github.com/ethersphere/beekeeper/pkg/check/longavailability"
	"github.com/ethersphere/beekeeper/pkg/check/manifest"
	"github.com/ethersphere/beekeeper/pkg/check/networkavailability"
	"github.com/ethersphere/beekeeper/pkg/check/peercount"
	"github.com/ethersphere/beekeeper/pkg/check/pingpong"
	"github.com/ethersphere/beekeeper/pkg/check/postage"
	"github.com/ethersphere/beekeeper/pkg/check/pss"
	"github.com/ethersphere/beekeeper/pkg/check/pullsync"
	"github.com/ethersphere/beekeeper/pkg/check/pushsync"
	"github.com/ethersphere/beekeeper/pkg/check/redundancy"
	"github.com/ethersphere/beekeeper/pkg/check/retrieval"
	"github.com/ethersphere/beekeeper/pkg/check/settlements"
	"github.com/ethersphere/beekeeper/pkg/check/smoke"
	"github.com/ethersphere/beekeeper/pkg/check/soc"
	"github.com/ethersphere/beekeeper/pkg/check/stake"
	"github.com/ethersphere/beekeeper/pkg/check/withdraw"
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
	NewAction  func(logging.Logger) beekeeper.Action       // links check with beekeeper action
	NewOptions func(CheckGlobalConfig, Check) (any, error) // check options
}

// CheckGlobalConfig represents global configs for all checks
type CheckGlobalConfig struct {
	Seed    int64
	GethURL string
}

// Checks represents all available check types
var Checks = map[string]CheckType{
	"act": {
		NewAction: act.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			checkOpts := new(struct {
				FileName     *string        `yaml:"file-name"`
				FileSize     *int64         `yaml:"file-size"`
				PostageTTL   *time.Duration `yaml:"postage-ttl"`
				PostageDepth *int64         `yaml:"postage-depth"`
				PostageLabel *string        `yaml:"postage-label"`
				Seed         *int64         `yaml:"seed"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := act.NewOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}
			return opts, nil
		},
	},
	"balances": {
		NewAction: balances.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			checkOpts := new(struct {
				DryRun             *bool          `yaml:"dry-run"`
				FileName           *string        `yaml:"file-name"`
				FileSize           *int64         `yaml:"file-size"`
				GasPrice           *string        `yaml:"gas-price"`
				PostageTTL         *time.Duration `yaml:"postage-ttl"`
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
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
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
	"file-retrieval": {
		NewAction: fileretrieval.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			checkOpts := new(struct {
				FileName        *string        `yaml:"file-name"`
				FileSize        *int64         `yaml:"file-size"`
				FilesPerNode    *int           `yaml:"files-per-node"`
				Full            *bool          `yaml:"full"`
				GasPrice        *string        `yaml:"gas-price"`
				PostageTTL      *time.Duration `yaml:"postage-ttl"`
				PostageLabel    *string        `yaml:"postage-label"`
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
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
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
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
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
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
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
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			checkOpts := new(struct {
				FilesInCollection *int           `yaml:"files-in-collection"`
				GasPrice          *string        `yaml:"gas-price"`
				MaxPathnameLength *int32         `yaml:"max-pathname-length"`
				PostageTTL        *time.Duration `yaml:"postage-ttl"`
				PostageDepth      *uint64        `yaml:"postage-depth"`
				PostageLabel      *string        `yaml:"postage-label"`
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
	"manifest-v1": {
		NewAction: manifest.NewCheckV1,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			checkOpts := new(struct {
				FilesInCollection *int           `yaml:"files-in-collection"`
				GasPrice          *string        `yaml:"gas-price"`
				MaxPathnameLength *int32         `yaml:"max-pathname-length"`
				PostageTTL        *time.Duration `yaml:"postage-ttl"`
				PostageDepth      *uint64        `yaml:"postage-depth"`
				PostageLabel      *string        `yaml:"postage-label"`
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
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			return nil, nil
		},
	},
	"pingpong": {
		NewAction: pingpong.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			opts := pingpong.NewDefaultOptions()
			return opts, nil
		},
	},
	"pss": {
		NewAction: pss.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			checkOpts := new(struct {
				Count          *int64         `yaml:"count"`
				AddressPrefix  *int           `yaml:"address-prefix"`
				GasPrice       *string        `yaml:"gas-price"`
				PostageTTL     *time.Duration `yaml:"postage-ttl"`
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
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			checkOpts := new(struct {
				ChunksPerNode              *int           `yaml:"chunks-per-node"`
				GasPrice                   *string        `yaml:"gas-price"`
				PostageTTL                 *time.Duration `yaml:"postage-ttl"`
				PostageLabel               *string        `yaml:"postage-label"`
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
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			checkOpts := new(struct {
				ChunksPerNode     *int           `yaml:"chunks-per-node"`
				GasPrice          *string        `yaml:"gas-price"`
				Mode              *string        `yaml:"mode"`
				PostageTTL        *time.Duration `yaml:"postage-ttl"`
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
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			checkOpts := new(struct {
				ChunksPerNode   *int           `yaml:"chunks-per-node"`
				PostageTTL      *time.Duration `yaml:"postage-ttl"`
				PostageDepth    *uint64        `yaml:"postage-depth"`
				PostageLabel    *string        `yaml:"postage-label"`
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
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			checkOpts := new(struct {
				DryRun             *bool          `yaml:"dry-run"`
				ExpectSettlements  *bool          `yaml:"expect-settlements"`
				FileName           *string        `yaml:"file-name"`
				FileSize           *int64         `yaml:"file-size"`
				GasPrice           *string        `yaml:"gas-price"`
				PostageTTL         *time.Duration `yaml:"postage-ttl"`
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
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			checkOpts := new(struct {
				ContentSize   *int64         `yaml:"content-size"`
				FileSizes     *[]int64       `yaml:"file-sizes"`
				RndSeed       *int64         `yaml:"rnd-seed"`
				PostageTTL    *time.Duration `yaml:"postage-ttl"`
				PostageDepth  *uint64        `yaml:"postage-depth"`
				PostageLabel  *string        `yaml:"postage-label"`
				TxOnErrWait   *time.Duration `yaml:"tx-on-err-wait"`
				RxOnErrWait   *time.Duration `yaml:"rx-on-err-wait"`
				NodesSyncWait *time.Duration `yaml:"nodes-sync-wait"`
				Duration      *time.Duration `yaml:"duration"`
				RLevels       *[]uint8       `yaml:"r-levels"`
			})

			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}

			if checkOpts.FileSizes == nil && checkOpts.ContentSize != nil {
				checkOpts.FileSizes = &[]int64{*checkOpts.ContentSize}
			}

			opts := smoke.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"load": {
		NewAction: load.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			checkOpts := new(struct {
				ContentSize             *int64         `yaml:"content-size"`
				RndSeed                 *int64         `yaml:"rnd-seed"`
				PostageTTL              *time.Duration `yaml:"postage-ttl"`
				PostageDepth            *uint64        `yaml:"postage-depth"`
				PostageLabel            *string        `yaml:"postage-label"`
				GasPrice                *string        `yaml:"gas-price"`
				TxOnErrWait             *time.Duration `yaml:"tx-on-err-wait"`
				RxOnErrWait             *time.Duration `yaml:"rx-on-err-wait"`
				NodesSyncWait           *time.Duration `yaml:"nodes-sync-wait"`
				Duration                *time.Duration `yaml:"duration"`
				UploaderCount           *int           `yaml:"uploader-count"`
				UploadGroups            *[]string      `yaml:"upload-groups"`
				DownloaderCount         *int           `yaml:"downloader-count"`
				DownloadGroups          *[]string      `yaml:"download-groups"`
				MaxCommittedDepth       *uint8         `yaml:"max-committed-depth"`
				CommittedDepthCheckWait *time.Duration `yaml:"committed-depth-check-wait"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}

			opts := load.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"soc": {
		NewAction: soc.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			checkOpts := new(struct {
				GasPrice       *string        `yaml:"gas-price"`
				PostageTTL     *time.Duration `yaml:"postage-ttl"`
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
	"postage": {
		NewAction: postage.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
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
	"stake": {
		NewAction: stake.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			checkOpts := new(struct {
				Amount             *big.Int `yaml:"amount"`
				InsufficientAmount *big.Int `yaml:"insufficient-amount"`
				ContractAddr       *string  `yaml:"contract-addr"`
				CallerPrivateKey   *string  `yaml:"private-key"`
				GethURL            *string  `yaml:"geth-url"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := stake.NewDefaultOptions()
			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}
			return opts, nil
		},
	},
	"longavailability": {
		NewAction: longavailability.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			checkOpts := new(struct {
				RndSeed      *int64         `yaml:"rnd-seed"`
				RetryCount   *int64         `yaml:"retry-count"`
				RetryWait    *time.Duration `yaml:"retry-wait"`
				Refs         *[]string      `yaml:"refs"`
				NextIterWait *time.Duration `yaml:"next-iter-wait"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := longavailability.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"networkavailability": {
		NewAction: networkavailability.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			checkOpts := new(struct {
				RndSeed       *int64         `yaml:"rnd-seed"`
				PostageTTL    *time.Duration `yaml:"postage-ttl"`
				PostageDepth  *uint64        `yaml:"postage-depth"`
				PostageLabel  *string        `yaml:"postage-label"`
				SleepDuration *time.Duration `yaml:"sleep-duration"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := networkavailability.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"datadurability": {
		NewAction: datadurability.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			checkOpts := new(struct {
				Ref         *string `yaml:"ref"`
				Concurrency *int    `yaml:"concurrency"`
				MaxAttempts *int    `yaml:"max-attempts"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := datadurability.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"redundancy": {
		NewAction: redundancy.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			checkOpts := new(struct {
				DataSize     *int           `yaml:"data-size"`
				PostageDepth *int           `yaml:"postage-depth"`
				PostageLabel *string        `yaml:"postage-label"`
				PostageTTL   *time.Duration `yaml:"postage-ttl"`
				Seed         *int           `yaml:"seed"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := redundancy.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"withdraw": {
		NewAction: withdraw.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			checkOpts := new(struct {
				TargetAddr *string `yaml:"target-address"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := withdraw.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"gsoc": {
		NewAction: gsoc.NewCheck,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			checkOpts := new(struct {
				PostageTTL   *time.Duration `yaml:"postage-ttl"`
				PostageDepth *uint64        `yaml:"postage-depth"`
				PostageLabel *string        `yaml:"postage-label"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := gsoc.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"feed-v1": {
		NewAction: feed.NewCheckV1,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			checkOpts := new(struct {
				PostageTTL   *time.Duration `yaml:"postage-ttl"`
				PostageDepth *uint64        `yaml:"postage-depth"`
				PostageLabel *string        `yaml:"postage-label"`
				NUpdates     *int           `yaml:"n-updates"`
				RootRef      *string        `yaml:"root-ref"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := feed.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"feed": {
		NewAction: feed.NewCheckV2,
		NewOptions: func(checkGlobalConfig CheckGlobalConfig, check Check) (any, error) {
			checkOpts := new(struct {
				PostageTTL   *time.Duration `yaml:"postage-ttl"`
				PostageDepth *uint64        `yaml:"postage-depth"`
				PostageLabel *string        `yaml:"postage-label"`
				NUpdates     *int           `yaml:"n-updates"`
				RootRef      *string        `yaml:"root-ref"`
			})
			if err := check.Options.Decode(checkOpts); err != nil {
				return nil, fmt.Errorf("decoding check %s options: %w", check.Type, err)
			}
			opts := feed.NewDefaultOptions()

			if err := applyCheckConfig(checkGlobalConfig, checkOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
}

// applyCheckConfig merges global and local options into default options
func applyCheckConfig(global CheckGlobalConfig, local, opts any) (err error) {
	lv := reflect.ValueOf(local).Elem()
	lt := reflect.TypeOf(local).Elem()
	ov := reflect.Indirect(reflect.ValueOf(opts).Elem())
	ot := reflect.TypeOf(opts).Elem()

	for i := range lv.NumField() {
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
		case "GethURL":
			if lv.Field(i).IsNil() { // set globally
				if global.GethURL != "" {
					v := reflect.ValueOf(global.GethURL)
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
		case "RLevels":
			if !lv.Field(i).IsNil() {
				fieldValue := lv.FieldByName(fieldName).Elem()
				n := fieldValue.Len()
				levels := make([]beeRedundancy.Level, n)
				for j := 0; j < n; j++ {
					levels[j] = beeRedundancy.Level(uint8(fieldValue.Index(j).Uint()))
				}
				ft, ok := ot.FieldByName(fieldName)
				if ok {
					v := reflect.ValueOf(levels)
					if v.Type().AssignableTo(ft.Type) {
						ov.FieldByName(fieldName).Set(v)
					}
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

	return err
}
