package configmap

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethersphere/beekeeper/pkg/k8s/mocks"
)

func TestSet(t *testing.T) {
	testTable := []struct {
		name       string
		configName string
		options    Options
		errorMsg   error
	}{
		{
			name:       "update_config_map",
			configName: "update",
		},
		{
			name:       "create_config_map",
			configName: "create",
		},
		{
			name:       "create_fail",
			configName: "create_bad",
			errorMsg:   fmt.Errorf("creating configmap create_bad in namespace test: mock error: cannot create config map"),
		},
		{
			name:       "update_fail",
			configName: "update_bad",
			errorMsg:   fmt.Errorf("updating configmap update_bad in namespace test: mock error"),
		},
	}
	client := NewClient(&mocks.ClientsetMock{})

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			err := client.Set(context.Background(), test.configName, "test", test.options)
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}
			} else {
				if err == nil {
					t.Fatalf("error not happened, expected: %s", test.errorMsg.Error())
				}
				if err.Error() != test.errorMsg.Error() {
					t.Errorf("error expected: %s, got: %s", test.errorMsg.Error(), err.Error())
				}
			}
		})
	}
}

func TestDelete(t *testing.T) {
	testTable := []struct {
		name       string
		configName string
		errorMsg   error
	}{
		{
			name:       "delete_config_map",
			configName: "delete",
		},
		{
			name:       "delete_not_found",
			configName: "delete_not_found",
		},
		{
			name:       "delete_bad",
			configName: "delete_bad",
			errorMsg:   fmt.Errorf("deleting configmap delete_bad in namespace test: mock error: cannot delete config map"),
		},
	}
	client := NewClient(&mocks.ClientsetMock{})

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			err := client.Delete(context.Background(), test.configName, "test")
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}
			} else {
				if err == nil {
					t.Fatalf("error not happened, expected: %s", test.errorMsg.Error())
				}
				if err.Error() != test.errorMsg.Error() {
					t.Errorf("error expected: %s, got: %s", test.errorMsg.Error(), err.Error())
				}
			}
		})
	}
}
