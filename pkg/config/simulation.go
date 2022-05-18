package config

import (
	"fmt"
	"reflect"
	"time"

	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/ethersphere/beekeeper/pkg/simulate/pushsync"
	"github.com/ethersphere/beekeeper/pkg/simulate/retrieval"
	"github.com/ethersphere/beekeeper/pkg/simulate/upload"
	"gopkg.in/yaml.v3"
)

// Simulation represents simulation configuration
type Simulation struct {
	Options yaml.Node      `yaml:"options"`
	Timeout *time.Duration `yaml:"timeout"`
	Type    string         `yaml:"type"`
}

// SimulationType is used for linking beekeeper actions with simulation and it's proper options
type SimulationType struct {
	NewAction  func() beekeeper.Action
	NewOptions func(SimulationGlobalConfig, Simulation) (interface{}, error)
}

// SimulationGlobalConfig represents global configs for all simulations
type SimulationGlobalConfig struct {
	Seed int64
}

// Checks represents all available simulation types
var Simulations = map[string]SimulationType{
	"upload": {
		NewAction: upload.NewSimulation,
		NewOptions: func(simulationGlobalConfig SimulationGlobalConfig, simulation Simulation) (interface{}, error) {
			simulationOpts := new(struct {
				FileCount            *int64         `yaml:"file-count"`
				GasPrice             *string        `yaml:"gas-price"`
				MaxFileSize          *int64         `yaml:"max-file-size"`
				MinFileSize          *int64         `yaml:"min-file-size"`
				PostageAmount        *int64         `yaml:"postage-amount"`
				PostageDepth         *uint64        `yaml:"postage-depth"`
				PostageLabel         *string        `yaml:"postage-label"`
				Retries              *int           `yaml:"retries"`
				RetryDelay           *time.Duration `yaml:"retry-delay"`
				Seed                 *int64         `yaml:"seed"`
				Timeout              *time.Duration `yaml:"timeout"`
				UploadNodeName       *string        `yaml:"upload-node-name"`
				UploadNodePercentage *int           `yaml:"upload-node-percentage"`
				SyncUpload           *bool          `yaml:"sync-upload"`
			})
			if err := simulation.Options.Decode(simulationOpts); err != nil {
				return nil, fmt.Errorf("decoding simulation %s options: %w", simulation.Type, err)
			}
			opts := upload.NewDefaultOptions()

			if err := applySimulationConfig(simulationGlobalConfig, simulationOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"retrieval": {
		NewAction: retrieval.NewSimulation,
		NewOptions: func(simulationGlobalConfig SimulationGlobalConfig, simulation Simulation) (interface{}, error) {
			simulationOpts := new(struct {
				ChunksPerNode   *int           `yaml:"chunks-per-node"`
				GasPrice        *string        `yaml:"gas-price"`
				PostageAmount   *int64         `yaml:"postage-amount"`
				PostageDepth    *uint64        `yaml:"postage-depth"`
				PostageLabel    *string        `yaml:"postage-label"`
				Seed            *int64         `yaml:"seed"`
				UploadNodeCount *int           `yaml:"upload-node-count"`
				UploadDelay     *time.Duration `yaml:"upload-delay"`
			})
			if err := simulation.Options.Decode(simulationOpts); err != nil {
				return nil, fmt.Errorf("decoding simulation %s options: %w", simulation.Type, err)
			}
			opts := retrieval.NewDefaultOptions()

			if err := applySimulationConfig(simulationGlobalConfig, simulationOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
	"pushsync": {
		NewAction: pushsync.NewSimulation,
		NewOptions: func(simulationGlobalConfig SimulationGlobalConfig, simulation Simulation) (interface{}, error) {
			simulationOpts := new(struct {
				GasPrice         *string  `yaml:"gas-price"`
				PostageAmount    *int64   `yaml:"postage-amount"`
				PostageDepth     *uint64  `yaml:"postage-depth"`
				Seed             *int64   `yaml:"seed"`
				ProxyApiEndpoint *string  `yaml:"proxy-api-endpoint"`
				ChunkCount       *int64   `yaml:"chunk-count"`
				StartPercentage  *float64 `yaml:"start-percentage"`
				EndPercentage    *float64 `yaml:"end-percentage"`
				StepPercentage   *float64 `yaml:"step-percentage"`
			})
			if err := simulation.Options.Decode(simulationOpts); err != nil {
				return nil, fmt.Errorf("decoding simulation %s options: %w", simulation.Type, err)
			}
			opts := pushsync.NewDefaultOptions()

			if err := applySimulationConfig(simulationGlobalConfig, simulationOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
}

// applySimulationConfig merges given, global and default simulation options
func applySimulationConfig(global SimulationGlobalConfig, local, opts interface{}) (err error) {
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
