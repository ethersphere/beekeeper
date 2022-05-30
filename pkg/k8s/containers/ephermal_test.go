package containers

import (
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
)

func Test_EphemeralContainers_ToK8S(t *testing.T) {
	testTable := []struct {
		name               string
		containers         EphemeralContainers
		expectedContainers []v1.EphemeralContainer
	}{
		{
			name: "default",
			containers: EphemeralContainers{
				{
					TargetContainerName: "test",
				},
			},
			expectedContainers: []v1.EphemeralContainer{
				{
					TargetContainerName: "test",
					EphemeralContainerCommon: v1.EphemeralContainerCommon{
						Resources: v1.ResourceRequirements{
							Requests: v1.ResourceList{},
							Limits:   v1.ResourceList{},
						},
						SecurityContext: &v1.SecurityContext{
							Privileged:               new(bool),
							SELinuxOptions:           &v1.SELinuxOptions{},
							RunAsUser:                new(int64),
							RunAsNonRoot:             new(bool),
							ReadOnlyRootFilesystem:   new(bool),
							AllowPrivilegeEscalation: new(bool),
							RunAsGroup:               new(int64),
							ProcMount: func() *v1.ProcMountType {
								procMountType := v1.ProcMountType("")
								return &procMountType
							}(),
						},
					},
				},
			},
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			containers := test.containers.ToK8S()

			if !reflect.DeepEqual(test.expectedContainers, containers) {
				t.Errorf("response expected: %#v, got: %#v", test.expectedContainers, containers)
			}
		})
	}
}
