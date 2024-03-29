package pod_test

import (
	"reflect"
	"testing"

	"github.com/ethersphere/beekeeper/pkg/k8s/pod"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestToK8S(t *testing.T) {
	testTable := []struct {
		name     string
		pts      pod.PodTemplateSpec
		expected v1.PodTemplateSpec
	}{
		{
			name:     "default",
			pts:      pod.PodTemplateSpec{},
			expected: newDefaultPodTemplateSpec(),
		},
		{
			name: "image_pull_secrets",
			pts: pod.PodTemplateSpec{
				Spec: pod.PodSpec{
					ImagePullSecrets: []string{"test_1", "test_2"},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodTemplateSpec()
				newPodSpec.Spec.ImagePullSecrets = []v1.LocalObjectReference{{Name: "test_1"}, {Name: "test_2"}}
				return newPodSpec
			}(),
		},
		{
			name: "preemption_policy",
			pts: pod.PodTemplateSpec{
				Spec: pod.PodSpec{
					PreemptionPolicy: "test",
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodTemplateSpec()
				newPodSpec.Spec.PreemptionPolicy = func() *v1.PreemptionPolicy {
					pp := v1.PreemptionPolicy("test")
					return &pp
				}()
				return newPodSpec
			}(),
		},
		{
			name: "node_affinity",
			pts: pod.PodTemplateSpec{
				Spec: pod.PodSpec{
					Affinity: pod.Affinity{
						NodeAffinity: &pod.NodeAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: pod.PreferredSchedulingTerms{
								{
									Preference: pod.NodeSelectorTerm{
										MatchExpressions: pod.NodeSelectorRequirements{
											{
												Key:      "key_1",
												Operator: "operator_1",
												Values:   []string{"value_1"},
											},
										},
										MatchFields: pod.NodeSelectorRequirements{
											{
												Key:      "key_2",
												Operator: "operator_2",
												Values:   []string{"value_2"},
											},
										},
									},
									Weight: 1,
								},
							},
							RequiredDuringSchedulingIgnoredDuringExecution: pod.NodeSelector{
								NodeSelectorTerms: pod.NodeSelectorTerms{
									{
										MatchExpressions: pod.NodeSelectorRequirements{
											{
												Key:      "key_3",
												Operator: "operator_3",
												Values:   []string{"value_3"},
											},
										},
										MatchFields: pod.NodeSelectorRequirements{
											{
												Key:      "key_4",
												Operator: "operator_4",
												Values:   []string{"value_4"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodTemplateSpec()
				newPodSpec.Spec.Affinity = &v1.Affinity{
					NodeAffinity: &v1.NodeAffinity{
						PreferredDuringSchedulingIgnoredDuringExecution: []v1.PreferredSchedulingTerm{
							{
								Weight: 1,
								Preference: v1.NodeSelectorTerm{
									MatchExpressions: []v1.NodeSelectorRequirement{{
										Key:      "key_1",
										Operator: "operator_1",
										Values:   []string{"value_1"},
									}},
									MatchFields: []v1.NodeSelectorRequirement{{
										Key:      "key_2",
										Operator: "operator_2",
										Values:   []string{"value_2"},
									}},
								},
							},
						},
						RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
							NodeSelectorTerms: []v1.NodeSelectorTerm{{
								MatchExpressions: []v1.NodeSelectorRequirement{{
									Key:      "key_3",
									Operator: "operator_3",
									Values:   []string{"value_3"},
								}},
								MatchFields: []v1.NodeSelectorRequirement{{
									Key:      "key_4",
									Operator: "operator_4",
									Values:   []string{"value_4"},
								}},
							}},
						},
					},
				}
				return newPodSpec
			}(),
		},
		{
			name: "pod_affinity",
			pts: pod.PodTemplateSpec{
				Spec: pod.PodSpec{
					Affinity: pod.Affinity{
						PodAffinity: &pod.PodAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: pod.WeightedPodAffinityTerms{
								{
									PodAffinityTerm: pod.PodAffinityTerm{
										LabelSelector: map[string]string{"label_1": "label_value_1"},
										Namespaces:    []string{"namespaces_1"},
										TopologyKey:   "topology_key_1",
									},
									Weight: 1,
								},
							},
							RequiredDuringSchedulingIgnoredDuringExecution: pod.PodAffinityTerms{
								{
									LabelSelector: map[string]string{"label_2": "label_value_2"},
									Namespaces:    []string{"namespaces_2"},
									TopologyKey:   "topology_key_2",
								},
							},
						},
					},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodTemplateSpec()
				newPodSpec.Spec.Affinity = &v1.Affinity{
					PodAffinity: &v1.PodAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
							{
								LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"label_2": "label_value_2"}},
								Namespaces:    []string{"namespaces_2"},
								TopologyKey:   "topology_key_2",
							},
						},
						PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
							{
								Weight: 1,
								PodAffinityTerm: v1.PodAffinityTerm{
									LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"label_1": "label_value_1"}},
									Namespaces:    []string{"namespaces_1"},
									TopologyKey:   "topology_key_1",
								},
							},
						},
					},
				}
				return newPodSpec
			}(),
		},
		{
			name: "pod_anti_affinity",
			pts: pod.PodTemplateSpec{
				Spec: pod.PodSpec{
					Affinity: pod.Affinity{
						PodAntiAffinity: &pod.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: pod.PodAffinityTerms{
								{
									LabelSelector: map[string]string{"label_3": "label_value_3"},
									Namespaces:    []string{"namespaces_3"},
									TopologyKey:   "topology_key_3",
								},
							},
							PreferredDuringSchedulingIgnoredDuringExecution: pod.WeightedPodAffinityTerms{
								{
									PodAffinityTerm: pod.PodAffinityTerm{
										LabelSelector: map[string]string{"label_4": "label_value_4"},
										Namespaces:    []string{"namespaces_4"},
										TopologyKey:   "topology_key_4",
									},
									Weight: 4,
								},
							},
						},
					},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodTemplateSpec()
				newPodSpec.Spec.Affinity = &v1.Affinity{
					PodAntiAffinity: &v1.PodAntiAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
							{
								LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"label_3": "label_value_3"}},
								Namespaces:    []string{"namespaces_3"},
								TopologyKey:   "topology_key_3",
							},
						},
						PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
							{
								Weight: 4,
								PodAffinityTerm: v1.PodAffinityTerm{
									LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"label_4": "label_value_4"}},
									Namespaces:    []string{"namespaces_4"},
									TopologyKey:   "topology_key_4",
								},
							},
						},
					},
				}
				return newPodSpec
			}(),
		},
		{
			name: "dns_config",
			pts: pod.PodTemplateSpec{
				Spec: pod.PodSpec{
					DNSConfig: pod.PodDNSConfig{
						Nameservers: []string{"server_1"},
						Searches:    []string{"search_1"},
						Options: []pod.PodDNSConfigOption{
							{
								Name:  "option_1",
								Value: "value_1",
							},
						},
					},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodTemplateSpec()
				newPodSpec.Spec.DNSConfig = &v1.PodDNSConfig{
					Nameservers: []string{"server_1"},
					Searches:    []string{"search_1"},
					Options: []v1.PodDNSConfigOption{
						{
							Name: "option_1",
							Value: func() *string {
								value := "value_1"
								return &value
							}(),
						},
					},
				}
				return newPodSpec
			}(),
		},
		{
			name: "host_aliases",
			pts: pod.PodTemplateSpec{
				Spec: pod.PodSpec{
					HostAliases: pod.HostAliases{
						{
							IP:        "8.8.8.8",
							Hostnames: []string{"host"},
						},
					},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodTemplateSpec()
				newPodSpec.Spec.HostAliases = []v1.HostAlias{{
					IP:        "8.8.8.8",
					Hostnames: []string{"host"},
				}}
				return newPodSpec
			}(),
		},
		{
			name: "pod_readiness_gate",
			pts: pod.PodTemplateSpec{
				Spec: pod.PodSpec{
					ReadinessGates: pod.PodReadinessGates{{
						ConditionType: "condition",
					}},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodTemplateSpec()
				newPodSpec.Spec.ReadinessGates = []v1.PodReadinessGate{{
					ConditionType: "condition",
				}}
				return newPodSpec
			}(),
		},
		{
			name: "tolerations",
			pts: pod.PodTemplateSpec{
				Spec: pod.PodSpec{
					Tolerations: pod.Tolerations{{
						Key:               "key",
						Operator:          "operator",
						Value:             "value",
						Effect:            "effect",
						TolerationSeconds: 1,
					}},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodTemplateSpec()
				newPodSpec.Spec.Tolerations = []v1.Toleration{{
					Key:      "key",
					Operator: "operator",
					Value:    "value",
					Effect:   "effect",
					TolerationSeconds: func() *int64 {
						var seconds int64 = 1
						return &seconds
					}(),
				}}
				return newPodSpec
			}(),
		},
		{
			name: "topology_spread_constraints",
			pts: pod.PodTemplateSpec{
				Spec: pod.PodSpec{
					TopologySpreadConstraints: pod.TopologySpreadConstraints{{
						MaxSkew:           1,
						TopologyKey:       "topology_key",
						WhenUnsatisfiable: "when_unsatisfiable",
						LabelSelector:     map[string]string{"label": "value"},
					}},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodTemplateSpec()
				newPodSpec.Spec.TopologySpreadConstraints = []v1.TopologySpreadConstraint{{
					MaxSkew:           1,
					TopologyKey:       "topology_key",
					WhenUnsatisfiable: "when_unsatisfiable",
					LabelSelector: &metav1.LabelSelector{
						MatchLabels:      map[string]string{"label": "value"},
						MatchExpressions: nil,
					},
				}}
				return newPodSpec
			}(),
		},
		{
			name: "volumes_empty_dir",
			pts: pod.PodTemplateSpec{
				Spec: pod.PodSpec{
					Volumes: pod.Volumes{{
						EmptyDir: &pod.EmptyDirVolume{
							Name:      "name",
							Medium:    "Memory",
							SizeLimit: "1",
						},
					}},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodTemplateSpec()
				newPodSpec.Spec.Volumes = []v1.Volume{{
					Name: "name",
					VolumeSource: v1.VolumeSource{
						EmptyDir: &v1.EmptyDirVolumeSource{
							Medium: v1.StorageMedium("Memory"),
							SizeLimit: func() *resource.Quantity {
								r := resource.MustParse("1")
								return &r
							}(),
						},
					},
				}}
				return newPodSpec
			}(),
		},
		{
			name: "volumes_empty_dir_no_size_limit",
			pts: pod.PodTemplateSpec{
				Spec: pod.PodSpec{
					Volumes: pod.Volumes{{
						EmptyDir: &pod.EmptyDirVolume{
							Name:   "name",
							Medium: "Memory",
						},
					}},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodTemplateSpec()
				newPodSpec.Spec.Volumes = []v1.Volume{{
					Name: "name",
					VolumeSource: v1.VolumeSource{
						EmptyDir: &v1.EmptyDirVolumeSource{
							Medium:    v1.StorageMedium("Memory"),
							SizeLimit: nil,
						},
					},
				}}
				return newPodSpec
			}(),
		},
		{
			name: "volumes_config_map",
			pts: pod.PodTemplateSpec{
				Spec: pod.PodSpec{
					Volumes: pod.Volumes{{
						ConfigMap: &pod.ConfigMapVolume{
							Name:          "name",
							ConfigMapName: "cm_name",
							DefaultMode:   1,
							Items: []pod.Item{{
								Key:   "key",
								Value: "value",
							}},
							Optional: true,
						},
					}},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodTemplateSpec()
				newPodSpec.Spec.Volumes = []v1.Volume{{
					Name: "name",
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: v1.LocalObjectReference{Name: "cm_name"},
							DefaultMode: func() *int32 {
								var mode int32 = 1
								return &mode
							}(),
							Items: []v1.KeyToPath{{
								Key:  "key",
								Path: "value",
								Mode: nil,
							}},
							Optional: func() *bool {
								var optional bool = true
								return &optional
							}(),
						},
					},
				}}
				return newPodSpec
			}(),
		},
		{
			name: "volumes_secret",
			pts: pod.PodTemplateSpec{
				Spec: pod.PodSpec{
					Volumes: pod.Volumes{{
						Secret: &pod.SecretVolume{
							Name:        "name",
							SecretName:  "secret_name",
							DefaultMode: 1,
							Items: []pod.Item{{
								Key:   "key",
								Value: "value",
							}},
							Optional: true,
						},
					}},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodTemplateSpec()
				newPodSpec.Spec.Volumes = []v1.Volume{{
					Name: "name",
					VolumeSource: v1.VolumeSource{
						Secret: &v1.SecretVolumeSource{
							SecretName: "secret_name",
							DefaultMode: func() *int32 {
								var mode int32 = 1
								return &mode
							}(),
							Items: []v1.KeyToPath{{
								Key:  "key",
								Path: "value",
								Mode: nil,
							}},
							Optional: func() *bool {
								var optional bool = true
								return &optional
							}(),
						},
					},
				}}
				return newPodSpec
			}(),
		},
		{
			name: "volumes_not_defined",
			pts: pod.PodTemplateSpec{
				Spec: pod.PodSpec{
					Volumes: pod.Volumes{{}},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodTemplateSpec()
				newPodSpec.Spec.Volumes = []v1.Volume{{}}
				return newPodSpec
			}(),
		},
		{
			name: "security_sysctl",
			pts: pod.PodTemplateSpec{
				Spec: pod.PodSpec{
					PodSecurityContext: pod.PodSecurityContext{
						Sysctls: pod.Sysctls{{
							Name:  "name",
							Value: "value",
						}},
					},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodTemplateSpec()
				newPodSpec.Spec.SecurityContext.Sysctls = []v1.Sysctl{{
					Name:  "name",
					Value: "value",
				}}
				return newPodSpec
			}(),
		},
		{
			name: "security_windows_options",
			pts: pod.PodTemplateSpec{
				Spec: pod.PodSpec{
					PodSecurityContext: pod.PodSecurityContext{
						WindowsOptions: pod.WindowsOptions{
							GMSACredentialSpecName: "spec_name",
							GMSACredentialSpec:     "spec",
							RunAsUserName:          "user",
						},
					},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodTemplateSpec()
				newPodSpec.Spec.SecurityContext.WindowsOptions = &v1.WindowsSecurityContextOptions{
					GMSACredentialSpecName: func() *string {
						name := "spec_name"
						return &name
					}(),
					GMSACredentialSpec: func() *string {
						spec := "spec"
						return &spec
					}(),
					RunAsUserName: func() *string {
						run := "user"
						return &run
					}(),
				}
				return newPodSpec
			}(),
		},
		{
			name: "security_fs_group_change_policy",
			pts: pod.PodTemplateSpec{
				Spec: pod.PodSpec{
					PodSecurityContext: pod.PodSecurityContext{
						FSGroupChangePolicy: "test",
					},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodTemplateSpec()
				newPodSpec.Spec.SecurityContext.FSGroupChangePolicy = func() *v1.PodFSGroupChangePolicy {
					f := v1.PodFSGroupChangePolicy("test")
					return &f
				}()
				return newPodSpec
			}(),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			pts := test.pts.ToK8S()

			if !reflect.DeepEqual(test.expected, pts) {
				t.Fatalf("response expected: %#v, got: %#v", test.expected, pts)
			}
		})
	}
}

func newDefaultPodTemplateSpec() v1.PodTemplateSpec {
	return v1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: v1.PodSpec{
			TerminationGracePeriodSeconds: new(int64),
			AutomountServiceAccountToken:  new(bool),
			ShareProcessNamespace:         new(bool),
			SecurityContext: &v1.PodSecurityContext{
				SELinuxOptions: &v1.SELinuxOptions{},
				RunAsUser:      new(int64),
				RunAsGroup:     new(int64),
				RunAsNonRoot:   new(bool),
				FSGroup:        new(int64),
			},
			Affinity:           &v1.Affinity{},
			Priority:           new(int32),
			DNSConfig:          &v1.PodDNSConfig{},
			EnableServiceLinks: new(bool),
		},
	}
}
