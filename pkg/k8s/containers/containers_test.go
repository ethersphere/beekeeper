package containers

import (
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestToK8S_Containers(t *testing.T) {
	testTable := []struct {
		name               string
		containers         Containers
		expectedContainers []v1.Container
	}{
		{
			name:       "default",
			containers: Containers{{}},
			expectedContainers: []v1.Container{
				newExpectedDefaultContainer(),
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

func TestToK8S(t *testing.T) {
	var trueBoolPointer bool = true

	testTable := []struct {
		name      string
		container Container
		expected  v1.Container
	}{
		{
			name: "init_simple",
			container: Container{
				Name:                     "container",
				Args:                     []string{"arg1", "arg2"},
				Command:                  []string{"cmd1", "cmd2"},
				Image:                    "image",
				ImagePullPolicy:          "imagePullPolicy",
				Stdin:                    true,
				StdinOnce:                true,
				TerminationMessagePath:   "TerminationMessagePath",
				TerminationMessagePolicy: "TerminationMessagePolicy",
				TTY:                      true,
				WorkingDir:               "WorkingDir",
			},
			expected: func() v1.Container {
				container := newExpectedDefaultContainer()
				container.Name = "container"
				container.Image = "image"
				container.Command = []string{"cmd1", "cmd2"}
				container.Args = []string{"arg1", "arg2"}
				container.WorkingDir = "WorkingDir"
				container.TerminationMessagePath = "TerminationMessagePath"
				container.TerminationMessagePolicy = "TerminationMessagePolicy"
				container.ImagePullPolicy = "imagePullPolicy"
				container.Stdin = true
				container.StdinOnce = true
				container.TTY = true
				return container
			}(),
		},
		{
			name: "env",
			container: Container{
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
			},
			expected: func() v1.Container {
				container := newExpectedDefaultContainer()
				container.Env = []v1.EnvVar{
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
				}
				return container
			}(),
		},
		{
			name: "env_from",
			container: Container{
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
			},
			expected: func() v1.Container {
				container := newExpectedDefaultContainer()
				container.EnvFrom = []v1.EnvFromSource{
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
				}
				return container
			}(),
		},
		{
			name: "lifecycle_exec",
			container: Container{
				Lifecycle: Lifecycle{
					PostStart: &Handler{
						Exec: &ExecHandler{
							Command: []string{"cmd_start1", "cmd_start2"},
						},
						HTTPGet:   &HTTPGetHandler{},
						TCPSocket: &TCPSocketHandler{},
					},
					PreStop: &Handler{
						Exec: &ExecHandler{
							Command: []string{"cmd_stop1", "cmd_stop2"},
						},
						HTTPGet:   &HTTPGetHandler{},
						TCPSocket: &TCPSocketHandler{},
					},
				},
			},
			expected: func() v1.Container {
				container := newExpectedDefaultContainer()
				container.Lifecycle = &v1.Lifecycle{
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
				}
				return container
			}(),
		},
		{
			name: "lifecycle_http",
			container: Container{
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
						TCPSocket: &TCPSocketHandler{},
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
						TCPSocket: &TCPSocketHandler{},
					},
				},
			},
			expected: func() v1.Container {
				container := newExpectedDefaultContainer()
				container.Lifecycle = &v1.Lifecycle{
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
				}
				return container
			}(),
		},
		{
			name: "lifecycle_tcp",
			container: Container{
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
			},
			expected: func() v1.Container {
				container := newExpectedDefaultContainer()
				container.Lifecycle = &v1.Lifecycle{
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
				}
				return container
			}(),
		},
		{
			name: "lifecycle_no_handlers",
			container: Container{
				Lifecycle: Lifecycle{
					PostStart: &Handler{},
					PreStop:   &Handler{},
				},
			},
			expected: func() v1.Container {
				container := newExpectedDefaultContainer()
				container.Lifecycle = &v1.Lifecycle{
					PostStart: &v1.Handler{},
					PreStop:   &v1.Handler{},
				}
				return container
			}(),
		},
		{
			name: "liveness_probe_exec",
			container: Container{
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
					HTTPGet:   &HTTPGetProbe{},
					TCPSocket: &TCPSocketProbe{},
				},
			},
			expected: func() v1.Container {
				container := newExpectedDefaultContainer()
				container.LivenessProbe = &v1.Probe{
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
				}
				return container
			}(),
		},
		{
			name: "liveness_probe_http",
			container: Container{
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
					TCPSocket: &TCPSocketProbe{},
				},
			},
			expected: func() v1.Container {
				container := newExpectedDefaultContainer()
				container.LivenessProbe = &v1.Probe{
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
				}
				return container
			}(),
		},
		{
			name: "liveness_probe_tcp",
			container: Container{
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
			},
			expected: func() v1.Container {
				container := newExpectedDefaultContainer()
				container.LivenessProbe = &v1.Probe{
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
				}
				return container
			}(),
		},
		{
			name: "ports",
			container: Container{
				Ports: []Port{
					{
						Name:          "port",
						ContainerPort: 12000,
						HostIP:        "hostIp",
						HostPort:      12001,
						Protocol:      "http",
					},
				},
			},
			expected: func() v1.Container {
				container := newExpectedDefaultContainer()
				container.Ports = []v1.ContainerPort{
					{
						Name:          "port",
						ContainerPort: 12000,
						HostIP:        "hostIp",
						HostPort:      12001,
						Protocol:      "http",
					},
				}
				return container
			}(),
		},
		{
			name: "readiness_probe_exec",
			container: Container{
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
					HTTPGet:   &HTTPGetProbe{},
					TCPSocket: &TCPSocketProbe{},
				},
			},
			expected: func() v1.Container {
				container := newExpectedDefaultContainer()
				container.ReadinessProbe = &v1.Probe{
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
				}
				return container
			}(),
		},
		{
			name: "readiness_probe_http",
			container: Container{
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
					TCPSocket: &TCPSocketProbe{},
				},
			},
			expected: func() v1.Container {
				container := newExpectedDefaultContainer()
				container.ReadinessProbe = &v1.Probe{
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
				}
				return container
			}(),
		},
		{
			name: "readiness_probe_tcp",
			container: Container{
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
			},
			expected: func() v1.Container {
				container := newExpectedDefaultContainer()
				container.ReadinessProbe = &v1.Probe{
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
				}
				return container
			}(),
		},
		{
			name: "resources",
			container: Container{
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
			},
			expected: func() v1.Container {
				container := newExpectedDefaultContainer()
				container.Resources = v1.ResourceRequirements{
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
				}
				return container
			}(),
		},
		{
			name: "security_context",
			container: Container{
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
			},
			expected: func() v1.Container {
				container := newExpectedDefaultContainer()
				container.SecurityContext = &v1.SecurityContext{
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
				}
				return container
			}(),
		},
		{
			name: "startup_probe_exec",
			container: Container{
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
					HTTPGet:   &HTTPGetProbe{},
					TCPSocket: &TCPSocketProbe{},
				},
			},
			expected: func() v1.Container {
				container := newExpectedDefaultContainer()
				container.StartupProbe = &v1.Probe{
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
				}
				return container
			}(),
		},
		{
			name: "startup_probe_http",
			container: Container{
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
					TCPSocket: &TCPSocketProbe{},
				},
			},
			expected: func() v1.Container {
				container := newExpectedDefaultContainer()
				container.StartupProbe = &v1.Probe{
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
				}
				return container
			}(),
		},
		{
			name: "startup_probe_tcp",
			container: Container{
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
			expected: func() v1.Container {
				container := newExpectedDefaultContainer()
				container.StartupProbe = &v1.Probe{
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
				}
				return container
			}(),
		},
		{
			name: "volume_devices",
			container: Container{
				VolumeDevices: []VolumeDevice{
					{
						Name:       "VolumeName",
						DevicePath: "VolumeDevicePath",
					},
				},
			},
			expected: func() v1.Container {
				container := newExpectedDefaultContainer()
				container.VolumeDevices = []v1.VolumeDevice{
					{
						Name:       "VolumeName",
						DevicePath: "VolumeDevicePath",
					},
				}
				return container
			}(),
		},
		{
			name: "volume_mounts",
			container: Container{
				VolumeMounts: []VolumeMount{
					{
						Name:      "VolumeMountName",
						MountPath: "VolumeMountPath",
						SubPath:   "VolumeMountSubPath",
						ReadOnly:  true,
					},
				},
			},
			expected: func() v1.Container {
				container := newExpectedDefaultContainer()
				container.VolumeMounts = []v1.VolumeMount{
					{
						Name:      "VolumeMountName",
						ReadOnly:  true,
						MountPath: "VolumeMountPath",
						SubPath:   "VolumeMountSubPath",
					},
				}
				return container
			}(),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			container := test.container.ToK8S()

			if !reflect.DeepEqual(test.expected, container) {
				t.Errorf("response expected: %#v, got: %#v", test.expected, container)
			}
		})
	}
}

func newExpectedDefaultContainer() v1.Container {
	return v1.Container{
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
	}
}
