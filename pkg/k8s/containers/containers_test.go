package containers

import (
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestToK8S(t *testing.T) {
	var trueBoolPointer bool = true

	testTable := []struct {
		name      string
		container Container
		expected  v1.Container
	}{
		{
			name:      "default",
			container: Container{},
			expected: v1.Container{
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{},
					Limits:   v1.ResourceList{},
				},
				SecurityContext: &v1.SecurityContext{
					Privileged:     new(bool),
					SELinuxOptions: &v1.SELinuxOptions{},
					WindowsOptions: &v1.WindowsSecurityContextOptions{
						GMSACredentialSpecName: func() *string {
							name := ""
							return &name
						}(),
						GMSACredentialSpec: func() *string {
							spec := ""
							return &spec
						}(),
						RunAsUserName: func() *string {
							run := ""
							return &run
						}(),
					},
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
		{
			name: "init_all",
			container: Container{
				Name:    "container",
				Args:    []string{"arg1", "arg2"},
				Command: []string{"cmd1", "cmd2"},
				Env: []EnvVar{
					{
						Name:  "dev",
						Value: "default",
						ValueFrom: ValueFrom{
							Field: Field{
								APIVersion: "v1",
								Path:       "/path",
							},
							ResourceField: ResourceField{
								ContainerName: "containerName",
								Resource:      "resource",
								Divisor:       "1Gi",
							},
							ConfigMap: ConfigMapKey{
								ConfigMapName: "configMapName",
								Key:           "key",
								Optional:      true,
							},
							Secret: SecretKey{
								SecretName: "secretName",
								Key:        "key",
								Optional:   true,
							},
						},
					},
				},
				EnvFrom: []EnvFrom{
					{
						Prefix: "pre",
						ConfigMap: ConfigMapRef{
							Name:     "configMapName",
							Optional: true,
						},
						Secret: SecretRef{
							Name:     "secretName",
							Optional: true,
						},
					},
				},
				Image:           "image",
				ImagePullPolicy: "imagePullPolicy",
				Lifecycle: Lifecycle{
					PostStart: &Handler{
						Exec: &ExecHandler{
							Command: []string{"cmd_start1", "cmd_start2"},
						},
						HTTPGet: &HTTPGetHandler{
							Host:   "host_start",
							Path:   "path",
							Port:   "10000",
							Scheme: "scheme",
							HTTPHeaders: []HTTPHeader{
								{
									Name:  "headerName",
									Value: "headerValue",
								},
							},
						},
						TCPSocket: &TCPSocketHandler{
							Host: "tcpHost_start",
							Port: "10001",
						},
					},
					PreStop: &Handler{
						Exec: &ExecHandler{
							Command: []string{"cmd_stop1", "cmd_stop2"},
						},
						HTTPGet: &HTTPGetHandler{
							Host:   "host_stop",
							Path:   "path",
							Port:   "10002",
							Scheme: "scheme",
							HTTPHeaders: []HTTPHeader{
								{
									Name:  "headerName",
									Value: "headerValue",
								},
							},
						},
						TCPSocket: &TCPSocketHandler{
							Host: "tcpHost_stop",
							Port: "10003",
						},
					},
				},
				LivenessProbe: Probe{
					Exec: &ExecProbe{
						FailureThreshold: 1,
						Handler: ExecHandler{
							Command: []string{"cmd_probe_1"},
						},
						InitialDelaySeconds: 2,
						PeriodSeconds:       3,
						SuccessThreshold:    4,
						TimeoutSeconds:      5,
					},
				},
				Ports: []Port{
					{
						Name:          "port",
						ContainerPort: 12000,
						HostIP:        "hostIp",
						HostPort:      12001,
						Protocol:      "http",
					},
				},
				ReadinessProbe: Probe{
					Exec: &ExecProbe{
						FailureThreshold: 16,
						Handler: ExecHandler{
							Command: []string{"cmd_ready_1"},
						},
						InitialDelaySeconds: 17,
						PeriodSeconds:       18,
						SuccessThreshold:    19,
						TimeoutSeconds:      20,
					},
				},
				Resources: Resources{
					Limit: Limit{
						CPU:              "100",
						Memory:           "101",
						Storage:          "102",
						EphemeralStorage: "103",
					},
					Request: Request{
						CPU:              "200",
						Memory:           "201",
						Storage:          "202",
						EphemeralStorage: "203",
					},
				},
				SecurityContext: SecurityContext{
					AllowPrivilegeEscalation: true,
					Capabilities: Capabilities{
						Add:  []string{"add"},
						Drop: []string{"drop"},
					},
					Privileged:             true,
					ProcMount:              "ProcMount",
					ReadOnlyRootFilesystem: true,
					RunAsGroup:             1,
					RunAsNonRoot:           true,
					RunAsUser:              2,
					SELinuxOptions: SELinuxOptions{
						User:  "user",
						Role:  "role",
						Type:  "type",
						Level: "level",
					},
					WindowsOptions: WindowsOptions{
						GMSACredentialSpecName: "name",
						GMSACredentialSpec:     "spec",
						RunAsUserName:          "run",
					},
				},
				StartupProbe: Probe{
					Exec: &ExecProbe{
						FailureThreshold: 31,
						Handler: ExecHandler{
							Command: []string{"cmd_startup_1"},
						},
						InitialDelaySeconds: 32,
						PeriodSeconds:       33,
						SuccessThreshold:    34,
						TimeoutSeconds:      35,
					},
				},
				Stdin:                    true,
				StdinOnce:                true,
				TerminationMessagePath:   "TerminationMessagePath",
				TerminationMessagePolicy: "TerminationMessagePolicy",
				TTY:                      true,
				VolumeDevices: []VolumeDevice{
					{
						Name:       "VolumeName",
						DevicePath: "VolumeDevicePath",
					},
				},
				VolumeMounts: []VolumeMount{
					{
						Name:      "VolumeMountName",
						MountPath: "VolumeMountPath",
						SubPath:   "VolumeMountSubPath",
						ReadOnly:  true,
					},
				},
				WorkingDir: "WorkingDir",
			},
			expected: v1.Container{
				Name:       "container",
				Image:      "image",
				Command:    []string{"cmd1", "cmd2"},
				Args:       []string{"arg1", "arg2"},
				WorkingDir: "WorkingDir",
				Ports: []v1.ContainerPort{
					{
						Name:          "port",
						ContainerPort: 12000,
						HostIP:        "hostIp",
						HostPort:      12001,
						Protocol:      "http",
					},
				},
				EnvFrom: []v1.EnvFromSource{
					{
						Prefix: "pre",
						ConfigMapRef: &v1.ConfigMapEnvSource{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "configMapName",
							},
							Optional: &trueBoolPointer,
						},
						SecretRef: &v1.SecretEnvSource{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "secretName",
							},
							Optional: &trueBoolPointer,
						},
					},
				},
				Env: []v1.EnvVar{
					{
						Name:  "dev",
						Value: "default",
						ValueFrom: &v1.EnvVarSource{
							FieldRef: &v1.ObjectFieldSelector{
								APIVersion: "v1",
								FieldPath:  "/path",
							},
							ResourceFieldRef: &v1.ResourceFieldSelector{
								ContainerName: "containerName",
								Resource:      "resource",
								Divisor:       resource.MustParse("1Gi"),
							},
							ConfigMapKeyRef: &v1.ConfigMapKeySelector{
								LocalObjectReference: v1.LocalObjectReference{
									Name: "configMapName",
								},
								Key:      "key",
								Optional: &trueBoolPointer,
							},
							SecretKeyRef: &v1.SecretKeySelector{
								LocalObjectReference: v1.LocalObjectReference{
									Name: "secretName",
								},
								Key:      "key",
								Optional: &trueBoolPointer,
							},
						},
					},
				},
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU:              resource.MustParse("200"),
						v1.ResourceMemory:           resource.MustParse("201"),
						v1.ResourceStorage:          resource.MustParse("202"),
						v1.ResourceEphemeralStorage: resource.MustParse("203"),
					},
					Limits: v1.ResourceList{
						v1.ResourceCPU:              resource.MustParse("100"),
						v1.ResourceMemory:           resource.MustParse("101"),
						v1.ResourceStorage:          resource.MustParse("102"),
						v1.ResourceEphemeralStorage: resource.MustParse("103"),
					},
				},
				VolumeMounts: []v1.VolumeMount{
					{
						Name:      "VolumeMountName",
						ReadOnly:  true,
						MountPath: "VolumeMountPath",
						SubPath:   "VolumeMountSubPath",
						// MountPropagation: &"", //TODO not used
						// SubPathExpr:      "", //TODO not used
					},
				},
				VolumeDevices: []v1.VolumeDevice{
					{
						Name:       "VolumeName",
						DevicePath: "VolumeDevicePath",
					},
				},
				LivenessProbe: &v1.Probe{
					Handler: v1.Handler{
						Exec: &v1.ExecAction{
							Command: []string{"cmd_probe_1"},
						},
					},
					InitialDelaySeconds: 2,
					TimeoutSeconds:      5,
					PeriodSeconds:       3,
					SuccessThreshold:    4,
					FailureThreshold:    1,
				},
				ReadinessProbe: &v1.Probe{
					Handler: v1.Handler{
						Exec: &v1.ExecAction{
							Command: []string{"cmd_ready_1"},
						},
					},
					InitialDelaySeconds: 17,
					TimeoutSeconds:      20,
					PeriodSeconds:       18,
					SuccessThreshold:    19,
					FailureThreshold:    16,
				},
				StartupProbe: &v1.Probe{
					Handler: v1.Handler{
						Exec: &v1.ExecAction{
							Command: []string{"cmd_startup_1"},
						},
					},
					InitialDelaySeconds: 32,
					TimeoutSeconds:      35,
					PeriodSeconds:       33,
					SuccessThreshold:    34,
					FailureThreshold:    31,
				},
				Lifecycle: &v1.Lifecycle{
					PostStart: &v1.Handler{
						Exec: &v1.ExecAction{
							Command: []string{"cmd_start1", "cmd_start2"},
						},
					},
					PreStop: &v1.Handler{
						Exec: &v1.ExecAction{
							Command: []string{"cmd_stop1", "cmd_stop2"},
						},
					},
				},
				TerminationMessagePath:   "TerminationMessagePath",
				TerminationMessagePolicy: "TerminationMessagePolicy",
				ImagePullPolicy:          "imagePullPolicy",
				SecurityContext: &v1.SecurityContext{
					Capabilities: &v1.Capabilities{
						Add:  []v1.Capability{"add"},
						Drop: []v1.Capability{"drop"},
					},
					Privileged: func() *bool {
						var pointerBool bool = true
						return &pointerBool
					}(),
					SELinuxOptions: &v1.SELinuxOptions{
						User:  "user",
						Role:  "role",
						Type:  "type",
						Level: "level",
					},
					WindowsOptions: &v1.WindowsSecurityContextOptions{
						GMSACredentialSpecName: func() *string {
							name := "name"
							return &name
						}(),
						GMSACredentialSpec: func() *string {
							spec := "spec"
							return &spec
						}(),
						RunAsUserName: func() *string {
							run := "run"
							return &run
						}(),
					},
					RunAsUser: func() *int64 {
						var pointerInt64 int64 = 2
						return &pointerInt64
					}(),
					RunAsGroup: func() *int64 {
						var pointerInt64 int64 = 1
						return &pointerInt64
					}(),
					RunAsNonRoot: func() *bool {
						var pointerBool bool = true
						return &pointerBool
					}(),
					ReadOnlyRootFilesystem: func() *bool {
						var pointerBool bool = true
						return &pointerBool
					}(),
					AllowPrivilegeEscalation: func() *bool {
						var pointerBool bool = true
						return &pointerBool
					}(),
					ProcMount: func() *v1.ProcMountType {
						p := v1.ProcMountType("ProcMount")
						return &p
					}(),
				},
				Stdin:     true,
				StdinOnce: true,
				TTY:       true,
			},
		},
		{
			name: "init_http_get",
			container: Container{
				Name: "container",
				Lifecycle: Lifecycle{
					PostStart: &Handler{
						HTTPGet: &HTTPGetHandler{
							Host:   "host_start",
							Path:   "path",
							Port:   "10000",
							Scheme: "scheme",
							HTTPHeaders: []HTTPHeader{
								{
									Name:  "headerName",
									Value: "headerValue",
								},
							},
						},
					},
					PreStop: &Handler{
						HTTPGet: &HTTPGetHandler{
							Host:   "host_stop",
							Path:   "path",
							Port:   "10002",
							Scheme: "scheme",
							HTTPHeaders: []HTTPHeader{
								{
									Name:  "headerName",
									Value: "headerValue",
								},
							},
						},
					},
				},
				LivenessProbe: Probe{
					HTTPGet: &HTTPGetProbe{
						FailureThreshold: 1,
						Handler: HTTPGetHandler{
							Host:   "http_host_lp",
							Path:   "path",
							Port:   "10000",
							Scheme: "scheme",
							HTTPHeaders: []HTTPHeader{
								{
									Name:  "headerName",
									Value: "headerValue",
								},
							},
						},
						InitialDelaySeconds: 2,
						PeriodSeconds:       3,
						SuccessThreshold:    4,
						TimeoutSeconds:      5,
					},
				},
				ReadinessProbe: Probe{
					HTTPGet: &HTTPGetProbe{
						FailureThreshold: 6,
						Handler: HTTPGetHandler{
							Host:   "http_host_rp",
							Path:   "path",
							Port:   "10000",
							Scheme: "scheme",
							HTTPHeaders: []HTTPHeader{
								{
									Name:  "headerName",
									Value: "headerValue",
								},
							},
						},
						InitialDelaySeconds: 7,
						PeriodSeconds:       8,
						SuccessThreshold:    9,
						TimeoutSeconds:      10,
					},
				},
				StartupProbe: Probe{
					HTTPGet: &HTTPGetProbe{
						FailureThreshold: 11,
						Handler: HTTPGetHandler{
							Host:   "http_host_sp",
							Path:   "path",
							Port:   "10000",
							Scheme: "scheme",
							HTTPHeaders: []HTTPHeader{
								{
									Name:  "headerName",
									Value: "headerValue",
								},
							},
						},
						InitialDelaySeconds: 12,
						PeriodSeconds:       13,
						SuccessThreshold:    14,
						TimeoutSeconds:      15,
					},
				},
			},
			expected: v1.Container{
				Name: "container",
				LivenessProbe: &v1.Probe{
					Handler: v1.Handler{
						HTTPGet: &v1.HTTPGetAction{
							Path: "path",
							Port: intstr.IntOrString{
								Type:   1,
								IntVal: 0,
								StrVal: "10000",
							},
							Host:   "http_host_lp",
							Scheme: "scheme",
							HTTPHeaders: []v1.HTTPHeader{
								{
									Name:  "headerName",
									Value: "headerValue",
								},
							},
						},
					},
					InitialDelaySeconds: 2,
					TimeoutSeconds:      5,
					PeriodSeconds:       3,
					SuccessThreshold:    4,
					FailureThreshold:    1,
				},
				ReadinessProbe: &v1.Probe{
					Handler: v1.Handler{
						HTTPGet: &v1.HTTPGetAction{
							Path: "path",
							Port: intstr.IntOrString{
								Type:   1,
								IntVal: 0,
								StrVal: "10000",
							},
							Host:   "http_host_rp",
							Scheme: "scheme",
							HTTPHeaders: []v1.HTTPHeader{
								{
									Name:  "headerName",
									Value: "headerValue",
								},
							},
						},
					},
					InitialDelaySeconds: 7,
					TimeoutSeconds:      10,
					PeriodSeconds:       8,
					SuccessThreshold:    9,
					FailureThreshold:    6,
				},
				StartupProbe: &v1.Probe{
					Handler: v1.Handler{
						HTTPGet: &v1.HTTPGetAction{
							Path: "path",
							Port: intstr.IntOrString{
								Type:   1,
								IntVal: 0,
								StrVal: "10000",
							},
							Host:   "http_host_sp",
							Scheme: "scheme",
							HTTPHeaders: []v1.HTTPHeader{
								{
									Name:  "headerName",
									Value: "headerValue",
								},
							},
						},
					},
					InitialDelaySeconds: 12,
					TimeoutSeconds:      15,
					PeriodSeconds:       13,
					SuccessThreshold:    14,
					FailureThreshold:    11,
				},
				Lifecycle: &v1.Lifecycle{
					PostStart: &v1.Handler{
						HTTPGet: &v1.HTTPGetAction{
							Host: "host_start",
							Path: "path",
							Port: intstr.IntOrString{
								Type:   1,
								IntVal: 0,
								StrVal: "10000",
							},
							Scheme: "scheme",
							HTTPHeaders: []v1.HTTPHeader{
								{
									Name:  "headerName",
									Value: "headerValue",
								},
							},
						},
					},
					PreStop: &v1.Handler{
						HTTPGet: &v1.HTTPGetAction{
							Host: "host_stop",
							Path: "path",
							Port: intstr.IntOrString{
								Type:   1,
								IntVal: 0,
								StrVal: "10002",
							},
							Scheme: "scheme",
							HTTPHeaders: []v1.HTTPHeader{
								{
									Name:  "headerName",
									Value: "headerValue",
								},
							},
						},
					},
				},
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{},
					Limits:   v1.ResourceList{},
				},
				SecurityContext: &v1.SecurityContext{
					Privileged:     new(bool),
					SELinuxOptions: &v1.SELinuxOptions{},
					WindowsOptions: &v1.WindowsSecurityContextOptions{
						GMSACredentialSpecName: func() *string {
							name := ""
							return &name
						}(),
						GMSACredentialSpec: func() *string {
							spec := ""
							return &spec
						}(),
						RunAsUserName: func() *string {
							run := ""
							return &run
						}(),
					},
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
		{
			name: "init_tcp",
			container: Container{
				Name: "container",
				Lifecycle: Lifecycle{
					PostStart: &Handler{
						TCPSocket: &TCPSocketHandler{
							Host: "tcp_post_start",
							Port: "10000",
						},
					},
					PreStop: &Handler{
						TCPSocket: &TCPSocketHandler{
							Host: "tcp_pre_stop",
							Port: "10000",
						},
					},
				},
				LivenessProbe: Probe{
					TCPSocket: &TCPSocketProbe{
						FailureThreshold: 1,
						Handler: TCPSocketHandler{
							Host: "tcp_lp",
							Port: "10000",
						},
						InitialDelaySeconds: 2,
						PeriodSeconds:       3,
						SuccessThreshold:    4,
						TimeoutSeconds:      5,
					},
				},
				ReadinessProbe: Probe{
					TCPSocket: &TCPSocketProbe{
						FailureThreshold: 1,
						Handler: TCPSocketHandler{
							Host: "tcp_rp",
							Port: "10000",
						},
						InitialDelaySeconds: 2,
						PeriodSeconds:       3,
						SuccessThreshold:    4,
						TimeoutSeconds:      5,
					},
				},
				StartupProbe: Probe{
					TCPSocket: &TCPSocketProbe{
						FailureThreshold: 1,
						Handler: TCPSocketHandler{
							Host: "tcp_sp",
							Port: "10000",
						},
						InitialDelaySeconds: 2,
						PeriodSeconds:       3,
						SuccessThreshold:    4,
						TimeoutSeconds:      5,
					},
				},
			},
			expected: v1.Container{
				Name: "container",
				LivenessProbe: &v1.Probe{
					Handler: v1.Handler{
						TCPSocket: &v1.TCPSocketAction{
							Port: intstr.IntOrString{
								Type:   1,
								IntVal: 0,
								StrVal: "10000",
							},
							Host: "tcp_lp",
						},
					},
					InitialDelaySeconds: 2,
					TimeoutSeconds:      5,
					PeriodSeconds:       3,
					SuccessThreshold:    4,
					FailureThreshold:    1,
				},
				ReadinessProbe: &v1.Probe{
					Handler: v1.Handler{
						TCPSocket: &v1.TCPSocketAction{
							Port: intstr.IntOrString{
								Type:   1,
								IntVal: 0,
								StrVal: "10000",
							},
							Host: "tcp_rp",
						},
					},
					InitialDelaySeconds: 2,
					TimeoutSeconds:      5,
					PeriodSeconds:       3,
					SuccessThreshold:    4,
					FailureThreshold:    1,
				},
				StartupProbe: &v1.Probe{
					Handler: v1.Handler{
						TCPSocket: &v1.TCPSocketAction{
							Port: intstr.IntOrString{
								Type:   1,
								IntVal: 0,
								StrVal: "10000",
							},
							Host: "tcp_sp",
						},
					},
					InitialDelaySeconds: 2,
					TimeoutSeconds:      5,
					PeriodSeconds:       3,
					SuccessThreshold:    4,
					FailureThreshold:    1,
				},
				Lifecycle: &v1.Lifecycle{
					PostStart: &v1.Handler{
						TCPSocket: &v1.TCPSocketAction{
							Port: intstr.IntOrString{
								Type:   1,
								IntVal: 0,
								StrVal: "10000",
							},
							Host: "tcp_post_start",
						},
					},
					PreStop: &v1.Handler{
						TCPSocket: &v1.TCPSocketAction{
							Port: intstr.IntOrString{
								Type:   1,
								IntVal: 0,
								StrVal: "10000",
							},
							Host: "tcp_pre_stop",
						},
					},
				},
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{},
					Limits:   v1.ResourceList{},
				},
				SecurityContext: &v1.SecurityContext{
					Privileged:     new(bool),
					SELinuxOptions: &v1.SELinuxOptions{},
					WindowsOptions: &v1.WindowsSecurityContextOptions{
						GMSACredentialSpecName: func() *string {
							name := ""
							return &name
						}(),
						GMSACredentialSpec: func() *string {
							spec := ""
							return &spec
						}(),
						RunAsUserName: func() *string {
							run := ""
							return &run
						}(),
					},
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
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			container := test.container.ToK8S()

			if !reflect.DeepEqual(test.expected.Args, container.Args) {
				t.Errorf("response Args expected: %#v, got: %#v", test.expected.Args, container.Args)
			}

			if !reflect.DeepEqual(test.expected.Command, container.Command) {
				t.Errorf("response Command expected: %#v, got: %#v", test.expected.Command, container.Command)
			}

			if !reflect.DeepEqual(test.expected.Env, container.Env) {
				t.Errorf("response Env expected: %#v, got: %#v", test.expected.Env, container.Env)
			}

			if !reflect.DeepEqual(test.expected.EnvFrom, container.EnvFrom) {
				t.Errorf("response EnvFrom expected: %#v, got: %#v", test.expected.EnvFrom, container.EnvFrom)
			}

			if !reflect.DeepEqual(test.expected.Image, container.Image) {
				t.Errorf("response Image expected: %#v, got: %#v", test.expected.Image, container.Image)
			}

			if !reflect.DeepEqual(test.expected.ImagePullPolicy, container.ImagePullPolicy) {
				t.Errorf("response ImagePullPolicy expected: %#v, got: %#v", test.expected.ImagePullPolicy, container.ImagePullPolicy)
			}

			if !reflect.DeepEqual(test.expected.Lifecycle, container.Lifecycle) {
				t.Errorf("response Lifecycle expected: %#v, got: %#v", test.expected.Lifecycle, container.Lifecycle)
			}

			if !reflect.DeepEqual(test.expected.LivenessProbe, container.LivenessProbe) {
				t.Errorf("response LivenessProbe expected: %#v, got: %#v", test.expected.LivenessProbe, container.LivenessProbe)
			}

			if !reflect.DeepEqual(test.expected.Name, container.Name) {
				t.Errorf("response Name expected: %#v, got: %#v", test.expected.Name, container.Name)
			}

			if !reflect.DeepEqual(test.expected.Ports, container.Ports) {
				t.Errorf("response Ports expected: %#v, got: %#v", test.expected.Ports, container.Ports)
			}

			if !reflect.DeepEqual(test.expected.ReadinessProbe, container.ReadinessProbe) {
				t.Errorf("response ReadinessProbe expected: %#v, got: %#v", test.expected.ReadinessProbe, container.ReadinessProbe)
			}

			if !reflect.DeepEqual(test.expected.Resources, container.Resources) {
				t.Errorf("response Resources expected: %#v, got: %#v", test.expected.Resources, container.Resources)
			}

			if !reflect.DeepEqual(test.expected.SecurityContext, container.SecurityContext) {
				t.Errorf("response SecurityContext expected: %#v, got: %#v", test.expected.SecurityContext, container.SecurityContext)
			}

			if !reflect.DeepEqual(test.expected.StartupProbe, container.StartupProbe) {
				t.Errorf("response StartupProbe expected: %#v, got: %#v", test.expected.StartupProbe, container.StartupProbe)
			}

			if !reflect.DeepEqual(test.expected.Stdin, container.Stdin) {
				t.Errorf("response Stdin expected: %#v, got: %#v", test.expected.Stdin, container.Stdin)
			}

			if !reflect.DeepEqual(test.expected.StdinOnce, container.StdinOnce) {
				t.Errorf("response StdinOnce expected: %#v, got: %#v", test.expected.StdinOnce, container.StdinOnce)
			}

			if !reflect.DeepEqual(test.expected.TTY, container.TTY) {
				t.Errorf("response TTY expected: %#v, got: %#v", test.expected.TTY, container.TTY)
			}

			if !reflect.DeepEqual(test.expected.TerminationMessagePath, container.TerminationMessagePath) {
				t.Errorf("response TerminationMessagePath expected: %#v, got: %#v", test.expected.TerminationMessagePath, container.TerminationMessagePath)
			}

			if !reflect.DeepEqual(test.expected.TerminationMessagePolicy, container.TerminationMessagePolicy) {
				t.Errorf("response TerminationMessagePolicy expected: %#v, got: %#v", test.expected.TerminationMessagePolicy, container.TerminationMessagePolicy)
			}

			if !reflect.DeepEqual(test.expected.VolumeDevices, container.VolumeDevices) {
				t.Errorf("response VolumeDevices expected: %#v, got: %#v", test.expected.VolumeDevices, container.VolumeDevices)
			}

			if !reflect.DeepEqual(test.expected.VolumeMounts, container.VolumeMounts) {
				t.Errorf("response VolumeMounts expected: %#v, got: %#v", test.expected.VolumeMounts, container.VolumeMounts)
			}

			if !reflect.DeepEqual(test.expected.WorkingDir, container.WorkingDir) {
				t.Errorf("response WorkingDir expected: %#v, got: %#v", test.expected.WorkingDir, container.WorkingDir)
			}

			if !reflect.DeepEqual(test.expected, container) {
				t.Error("response not as expected")
			}
		})
	}
}
