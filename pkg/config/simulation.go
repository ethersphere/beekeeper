package config

import (
	"fmt"
	"reflect"
	"time"

	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/ethersphere/beekeeper/pkg/simulate/upload"
	"github.com/prometheus/client_golang/prometheus/push"
)

type SimulationGlobalConfig struct {
	MetricsEnabled bool
	MetricsPusher  *push.Pusher
	Seed           int64
}

// TODO: consider SimulationClass, SimulationKind, SimulationType, etc.
type Simulation struct {
	NewAction  func() beekeeper.Action
	NewOptions func(SimulationConfig, SimulationGlobalConfig) (interface{}, error)
}

var Simulations = map[string]Simulation{
	"upload": {
		NewAction: upload.NewSimulation,
		NewOptions: func(simulationConfig SimulationConfig, simulationGlobalConfig SimulationGlobalConfig) (interface{}, error) {
			simulationOpts := new(struct {
				FileSize             *int64         `yaml:"file-size"`
				Retries              *int           `yaml:"retries"`
				RetryDelay           *time.Duration `yaml:"retry-delay"`
				Seed                 *int64         `yaml:"seed"`
				Timeout              *time.Duration `yaml:"timeout"`
				UploadNodePercentage *int           `yaml:"upload-node-percentage"`
			})
			if err := simulationConfig.Options.Decode(simulationOpts); err != nil {
				return nil, fmt.Errorf("decoding simulation %s options: %w", simulationConfig.Name, err)
			}
			opts := upload.NewDefaultOptions()

			if err := applySimulationConfig(simulationGlobalConfig, simulationOpts, &opts); err != nil {
				return nil, fmt.Errorf("applying options: %w", err)
			}

			return opts, nil
		},
	},
}

func applySimulationConfig(global SimulationGlobalConfig, local, opts interface{}) (err error) {
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
