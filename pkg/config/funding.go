package config

import (
	"reflect"

	orchestration "github.com/ethersphere/beekeeper/pkg/orchestration/k8s"
)

// Funding represents funding deposits for every node in the cluster
type Funding struct {
	Bzz  *float64 `yaml:"bzz"`
	Eth  *float64 `yaml:"eth"`
	GBzz *float64 `yaml:"gbzz"`
}

// Export exports Funding to orchestration.FundingOptions
func (f *Funding) Export() (o orchestration.FundingOptions) {
	localVal := reflect.ValueOf(f).Elem()
	localType := reflect.TypeOf(f).Elem()
	remoteVal := reflect.ValueOf(&o).Elem()

	for i := 0; i < localVal.NumField(); i++ {
		localField := localVal.Field(i)
		if localField.IsValid() && !localField.IsNil() {
			localFieldVal := localVal.Field(i).Elem()
			localFieldName := localType.Field(i).Name

			remoteFieldVal := remoteVal.FieldByName(localFieldName)
			if remoteFieldVal.IsValid() && remoteFieldVal.Type() == localFieldVal.Type() {
				remoteFieldVal.Set(localFieldVal)
			}
		}
	}

	return remoteVal.Interface().(orchestration.FundingOptions)
}
