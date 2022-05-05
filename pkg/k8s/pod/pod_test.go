package pod

import (
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestToK8S(t *testing.T) {
	testTable := []struct {
		name     string
		pts      PodTemplateSpec
		expected v1.PodTemplateSpec
	}{
		{
			name:     "default",
			pts:      PodTemplateSpec{},
			expected: newDefaultPodSpec(),
		},
		{
			name: "image_pull_secrets",
			pts: PodTemplateSpec{
				Spec: PodSpec{
					ImagePullSecrets: []string{"test_1", "test_2"},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodSpec()
				newPodSpec.Spec.ImagePullSecrets = []v1.LocalObjectReference{{Name: "test_1"}, {Name: "test_2"}}
				return newPodSpec
			}(),
		},
		{
			name: "preemption_policy",
			pts: PodTemplateSpec{
				Spec: PodSpec{
					PreemptionPolicy: "test",
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodSpec()
				newPodSpec.Spec.PreemptionPolicy = func() *v1.PreemptionPolicy {
					pp := v1.PreemptionPolicy("test")
					return &pp
				}()
				return newPodSpec
			}(),
		},
		{
			name: "node_affinity",
			pts: PodTemplateSpec{
				Spec: PodSpec{
					Affinity: Affinity{
						NodeAffinity: &NodeAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: PreferredSchedulingTerms{
								{
									Preference: NodeSelectorTerm{
										MatchExpressions: NodeSelectorRequirements{
											{
												Key:      "key_1",
												Operator: "operator_1",
												Values:   []string{"value_1"},
											},
										},
										MatchFields: NodeSelectorRequirements{
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
							RequiredDuringSchedulingIgnoredDuringExecution: NodeSelector{
								NodeSelectorTerms: NodeSelectorTerms{
									{
										MatchExpressions: NodeSelectorRequirements{
											{
												Key:      "key_3",
												Operator: "operator_3",
												Values:   []string{"value_3"},
											},
										},
										MatchFields: NodeSelectorRequirements{
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
				newPodSpec := newDefaultPodSpec()
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
			pts: PodTemplateSpec{
				Spec: PodSpec{
					Affinity: Affinity{
						PodAffinity: &PodAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: WeightedPodAffinityTerms{
								{
									PodAffinityTerm: PodAffinityTerm{
										LabelSelector: map[string]string{"label_1": "label_value_1"},
										Namespaces:    []string{"namespaces_1"},
										TopologyKey:   "topology_key_1",
									},
									Weight: 1,
								},
							},
							RequiredDuringSchedulingIgnoredDuringExecution: PodAffinityTerms{
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
				newPodSpec := newDefaultPodSpec()
				newPodSpec.Spec.Affinity = &v1.Affinity{
					PodAffinity: &v1.PodAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
							{
								LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"label_2": "label_value_2"}},
								Namespaces:    []string{"namespaces_2"},
								TopologyKey:   "topology_key_2",
								// NamespaceSelector: &metav1.LabelSelector{}, //TODO not used?
							},
						},
						PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
							{
								Weight: 1,
								PodAffinityTerm: v1.PodAffinityTerm{
									LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"label_1": "label_value_1"}},
									Namespaces:    []string{"namespaces_1"},
									TopologyKey:   "topology_key_1",
									// NamespaceSelector: &metav1.LabelSelector{}, //TODO not used?
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
			pts: PodTemplateSpec{
				Spec: PodSpec{
					Affinity: Affinity{
						PodAntiAffinity: &PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: PodAffinityTerms{
								{
									LabelSelector: map[string]string{"label_3": "label_value_3"},
									Namespaces:    []string{"namespaces_3"},
									TopologyKey:   "topology_key_3",
								},
							},
							PreferredDuringSchedulingIgnoredDuringExecution: WeightedPodAffinityTerms{
								{
									PodAffinityTerm: PodAffinityTerm{
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
				newPodSpec := newDefaultPodSpec()
				newPodSpec.Spec.Affinity = &v1.Affinity{
					PodAntiAffinity: &v1.PodAntiAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
							{
								LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"label_3": "label_value_3"}},
								Namespaces:    []string{"namespaces_3"},
								TopologyKey:   "topology_key_3",
								// NamespaceSelector: &metav1.LabelSelector{}, //TODO not used?
							},
						},
						PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
							{
								Weight: 4,
								PodAffinityTerm: v1.PodAffinityTerm{
									LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"label_4": "label_value_4"}},
									Namespaces:    []string{"namespaces_4"},
									TopologyKey:   "topology_key_4",
									// NamespaceSelector: &metav1.LabelSelector{}, //TODO not used?
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
			pts: PodTemplateSpec{
				Spec: PodSpec{
					DNSConfig: PodDNSConfig{
						Nameservers: []string{"server_1"},
						Searches:    []string{"search_1"},
						Options: []PodDNSConfigOption{
							{
								Name:  "option_1",
								Value: "value_1",
							},
						},
					},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodSpec()
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
			pts: PodTemplateSpec{
				Spec: PodSpec{
					HostAliases: HostAliases{
						{
							IP:        "8.8.8.8",
							Hostnames: []string{"host"},
						},
					},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodSpec()
				newPodSpec.Spec.HostAliases = []v1.HostAlias{{
					IP:        "8.8.8.8",
					Hostnames: []string{"host"},
				}}
				return newPodSpec
			}(),
		},
		{
			name: "pod_readiness_gate",
			pts: PodTemplateSpec{
				Spec: PodSpec{
					ReadinessGates: PodReadinessGates{{
						ConditionType: "condition",
					}},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodSpec()
				newPodSpec.Spec.ReadinessGates = []v1.PodReadinessGate{{
					ConditionType: "condition",
				}}
				return newPodSpec
			}(),
		},
		{
			name: "tolerations",
			pts: PodTemplateSpec{
				Spec: PodSpec{
					Tolerations: Tolerations{{
						Key:               "key",
						Operator:          "operator",
						Value:             "value",
						Effect:            "effect",
						TolerationSeconds: 1,
					}},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodSpec()
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
			pts: PodTemplateSpec{
				Spec: PodSpec{
					TopologySpreadConstraints: TopologySpreadConstraints{{
						MaxSkew:           1,
						TopologyKey:       "topology_key",
						WhenUnsatisfiable: "when_unsatisfiable",
						LabelSelector:     map[string]string{"label": "value"},
					}},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodSpec()
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
			pts: PodTemplateSpec{
				Spec: PodSpec{
					Volumes: Volumes{{
						EmptyDir: &EmptyDirVolume{
							Name:      "name",
							Medium:    "Memory",
							SizeLimit: "1",
						},
					}},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodSpec()
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
			pts: PodTemplateSpec{
				Spec: PodSpec{
					Volumes: Volumes{{
						EmptyDir: &EmptyDirVolume{
							Name:   "name",
							Medium: "Memory",
						},
					}},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodSpec()
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
			pts: PodTemplateSpec{
				Spec: PodSpec{
					Volumes: Volumes{{
						ConfigMap: &ConfigMapVolume{
							Name:          "name",
							ConfigMapName: "cm_name",
							DefaultMode:   1,
							Items: []Item{{
								Key:   "key",
								Value: "value",
							}},
							Optional: true,
						},
					}},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodSpec()
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
			pts: PodTemplateSpec{
				Spec: PodSpec{
					Volumes: Volumes{{
						Secret: &SecretVolume{
							Name:        "name",
							SecretName:  "secret_name",
							DefaultMode: 1,
							Items: []Item{{
								Key:   "key",
								Value: "value",
							}},
							Optional: true,
						},
					}},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodSpec()
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
			pts: PodTemplateSpec{
				Spec: PodSpec{
					Volumes: Volumes{{}},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodSpec()
				newPodSpec.Spec.Volumes = []v1.Volume{{}}
				return newPodSpec
			}(),
		},
		{
			name: "security_sysctl",
			pts: PodTemplateSpec{
				Spec: PodSpec{
					PodSecurityContext: PodSecurityContext{
						Sysctls: Sysctls{{
							Name:  "name",
							Value: "value",
						}},
					},
				},
			},
			expected: func() v1.PodTemplateSpec {
				newPodSpec := newDefaultPodSpec()
				newPodSpec.Spec.SecurityContext.Sysctls = []v1.Sysctl{{
					Name:  "name",
					Value: "value",
				}}
				return newPodSpec
			}(),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			pts := test.pts.ToK8S()

			if !reflect.DeepEqual(test.expected.ObjectMeta, pts.ObjectMeta) {
				t.Errorf("response ObjectMeta expected: %#v, got: %#v", test.expected.ObjectMeta, pts.ObjectMeta)
			}

			if !reflect.DeepEqual(test.expected.Spec.ActiveDeadlineSeconds, pts.Spec.ActiveDeadlineSeconds) {
				t.Errorf("response Spec.ActiveDeadlineSeconds expected: %#v, got: %#v", test.expected.Spec.ActiveDeadlineSeconds, pts.Spec.ActiveDeadlineSeconds)
			}

			if !reflect.DeepEqual(test.expected.Spec.Affinity, pts.Spec.Affinity) {
				t.Errorf("response Spec.Affinity expected: %#v, got: %#v", test.expected.Spec.Affinity, pts.Spec.Affinity)
			}

			if !reflect.DeepEqual(test.expected.Spec.AutomountServiceAccountToken, pts.Spec.AutomountServiceAccountToken) {
				t.Errorf("response Spec.AutomountServiceAccountToken expected: %#v, got: %#v", test.expected.Spec.AutomountServiceAccountToken, pts.Spec.AutomountServiceAccountToken)
			}

			if !reflect.DeepEqual(test.expected.Spec.Containers, pts.Spec.Containers) {
				t.Errorf("response Spec.Containers expected: %#v, got: %#v", test.expected.Spec.Containers, pts.Spec.Containers)
			}

			if !reflect.DeepEqual(test.expected.Spec.DNSConfig, pts.Spec.DNSConfig) {
				t.Errorf("response Spec.DNSConfig expected: %#v, got: %#v", test.expected.Spec.DNSConfig, pts.Spec.DNSConfig)
			}

			if !reflect.DeepEqual(test.expected.Spec.DNSPolicy, pts.Spec.DNSPolicy) {
				t.Errorf("response Spec.DNSPolicy expected: %#v, got: %#v", test.expected.Spec.DNSPolicy, pts.Spec.DNSPolicy)
			}

			if !reflect.DeepEqual(test.expected.Spec.EnableServiceLinks, pts.Spec.EnableServiceLinks) {
				t.Errorf("response Spec.EnableServiceLinks expected: %#v, got: %#v", test.expected.Spec.EnableServiceLinks, pts.Spec.EnableServiceLinks)
			}

			if !reflect.DeepEqual(test.expected.Spec.EphemeralContainers, pts.Spec.EphemeralContainers) {
				t.Errorf("response Spec.EphemeralContainers expected: %#v, got: %#v", test.expected.Spec.EphemeralContainers, pts.Spec.EphemeralContainers)
			}

			if !reflect.DeepEqual(test.expected.Spec.HostAliases, pts.Spec.HostAliases) {
				t.Errorf("response Spec.HostAliases expected: %#v, got: %#v", test.expected.Spec.HostAliases, pts.Spec.HostAliases)
			}

			if !reflect.DeepEqual(test.expected.Spec.HostIPC, pts.Spec.HostIPC) {
				t.Errorf("response Spec.HostIPC expected: %#v, got: %#v", test.expected.Spec.HostIPC, pts.Spec.HostIPC)
			}

			if !reflect.DeepEqual(test.expected.Spec.HostNetwork, pts.Spec.HostNetwork) {
				t.Errorf("response Spec.HostNetwork expected: %#v, got: %#v", test.expected.Spec.HostNetwork, pts.Spec.HostNetwork)
			}

			if !reflect.DeepEqual(test.expected.Spec.HostPID, pts.Spec.HostPID) {
				t.Errorf("response Spec.HostPID expected: %#v, got: %#v", test.expected.Spec.HostPID, pts.Spec.HostPID)
			}

			if !reflect.DeepEqual(test.expected.Spec.Hostname, pts.Spec.Hostname) {
				t.Errorf("response Spec.Hostname expected: %#v, got: %#v", test.expected.Spec.Hostname, pts.Spec.Hostname)
			}

			if !reflect.DeepEqual(test.expected.Spec.ImagePullSecrets, pts.Spec.ImagePullSecrets) {
				t.Errorf("response Spec.ImagePullSecrets expected: %#v, got: %#v", test.expected.Spec.ImagePullSecrets, pts.Spec.ImagePullSecrets)
			}

			if !reflect.DeepEqual(test.expected.Spec.InitContainers, pts.Spec.InitContainers) {
				t.Errorf("response Spec.InitContainers expected: %#v, got: %#v", test.expected.Spec.InitContainers, pts.Spec.InitContainers)
			}

			if !reflect.DeepEqual(test.expected.Spec.NodeName, pts.Spec.NodeName) {
				t.Errorf("response Spec.NodeName expected: %#v, got: %#v", test.expected.Spec.NodeName, pts.Spec.NodeName)
			}

			if !reflect.DeepEqual(test.expected.Spec.NodeSelector, pts.Spec.NodeSelector) {
				t.Errorf("response Spec.NodeSelector expected: %#v, got: %#v", test.expected.Spec.NodeSelector, pts.Spec.NodeSelector)
			}

			if !reflect.DeepEqual(test.expected.Spec.Overhead, pts.Spec.Overhead) {
				t.Errorf("response Spec.Overhead expected: %#v, got: %#v", test.expected.Spec.Overhead, pts.Spec.Overhead)
			}

			if !reflect.DeepEqual(test.expected.Spec.PreemptionPolicy, pts.Spec.PreemptionPolicy) {
				t.Errorf("response Spec.PreemptionPolicy expected: %#v, got: %#v", test.expected.Spec.PreemptionPolicy, pts.Spec.PreemptionPolicy)
			}

			if !reflect.DeepEqual(test.expected.Spec.Priority, pts.Spec.Priority) {
				t.Errorf("response Spec.Priority expected: %#v, got: %#v", test.expected.Spec.Priority, pts.Spec.Priority)
			}

			if !reflect.DeepEqual(test.expected.Spec.PriorityClassName, pts.Spec.PriorityClassName) {
				t.Errorf("response Spec.PriorityClassName expected: %#v, got: %#v", test.expected.Spec.PriorityClassName, pts.Spec.PriorityClassName)
			}

			if !reflect.DeepEqual(test.expected.Spec.ReadinessGates, pts.Spec.ReadinessGates) {
				t.Errorf("response Spec.ReadinessGates expected: %#v, got: %#v", test.expected.Spec.ReadinessGates, pts.Spec.ReadinessGates)
			}

			if !reflect.DeepEqual(test.expected.Spec.RestartPolicy, pts.Spec.RestartPolicy) {
				t.Errorf("response Spec.RestartPolicy expected: %#v, got: %#v", test.expected.Spec.RestartPolicy, pts.Spec.RestartPolicy)
			}

			if !reflect.DeepEqual(test.expected.Spec.RuntimeClassName, pts.Spec.RuntimeClassName) {
				t.Errorf("response Spec.RuntimeClassName expected: %#v, got: %#v", test.expected.Spec.RuntimeClassName, pts.Spec.RuntimeClassName)
			}

			if !reflect.DeepEqual(test.expected.Spec.SchedulerName, pts.Spec.SchedulerName) {
				t.Errorf("response Spec.SchedulerName expected: %#v, got: %#v", test.expected.Spec.SchedulerName, pts.Spec.SchedulerName)
			}

			if !reflect.DeepEqual(test.expected.Spec.SecurityContext, pts.Spec.SecurityContext) {
				t.Errorf("response Spec.SecurityContext expected: %#v, got: %#v", test.expected.Spec.SecurityContext, pts.Spec.SecurityContext)
			}

			if !reflect.DeepEqual(test.expected.Spec.ServiceAccountName, pts.Spec.ServiceAccountName) {
				t.Errorf("response Spec.ServiceAccountName expected: %#v, got: %#v", test.expected.Spec.ServiceAccountName, pts.Spec.ServiceAccountName)
			}

			if !reflect.DeepEqual(test.expected.Spec.SetHostnameAsFQDN, pts.Spec.SetHostnameAsFQDN) {
				t.Errorf("response Spec.SetHostnameAsFQDN expected: %#v, got: %#v", test.expected.Spec.SetHostnameAsFQDN, pts.Spec.SetHostnameAsFQDN)
			}

			if !reflect.DeepEqual(test.expected.Spec.ShareProcessNamespace, pts.Spec.ShareProcessNamespace) {
				t.Errorf("response Spec.ShareProcessNamespace expected: %#v, got: %#v", test.expected.Spec.ShareProcessNamespace, pts.Spec.ShareProcessNamespace)
			}

			if !reflect.DeepEqual(test.expected.Spec.Subdomain, pts.Spec.Subdomain) {
				t.Errorf("response Spec.Subdomain expected: %#v, got: %#v", test.expected.Spec.Subdomain, pts.Spec.Subdomain)
			}

			if !reflect.DeepEqual(test.expected.Spec.TerminationGracePeriodSeconds, pts.Spec.TerminationGracePeriodSeconds) {
				t.Errorf("response Spec.TerminationGracePeriodSeconds expected: %#v, got: %#v", test.expected.Spec.TerminationGracePeriodSeconds, pts.Spec.TerminationGracePeriodSeconds)
			}

			if !reflect.DeepEqual(test.expected.Spec.Tolerations, pts.Spec.Tolerations) {
				t.Errorf("response Spec.Tolerations expected: %#v, got: %#v", test.expected.Spec.Tolerations, pts.Spec.Tolerations)
			}

			if !reflect.DeepEqual(test.expected.Spec.TopologySpreadConstraints, pts.Spec.TopologySpreadConstraints) {
				t.Errorf("response Spec.TopologySpreadConstraints expected: %#v, got: %#v", test.expected.Spec.TopologySpreadConstraints, pts.Spec.TopologySpreadConstraints)
			}

			if !reflect.DeepEqual(test.expected.Spec.Volumes, pts.Spec.Volumes) {
				t.Errorf("response Spec.Volumes expected: %#v, got: %#v", test.expected.Spec.Volumes, pts.Spec.Volumes)
			}

			if !reflect.DeepEqual(test.expected.Spec, pts.Spec) {
				t.Fatalf("response Spec expected: %#v, got: %#v", test.expected.Spec, pts.Spec)
			}

			if !reflect.DeepEqual(test.expected, pts) {
				t.Fatalf("response expected: %#v, got: %#v", test.expected, pts)
			}
		})
	}
}

func newDefaultPodSpec() v1.PodTemplateSpec {
	return v1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: v1.PodSpec{
			TerminationGracePeriodSeconds: new(int64),
			AutomountServiceAccountToken:  new(bool),
			ShareProcessNamespace:         new(bool),
			SecurityContext: &v1.PodSecurityContext{
				SELinuxOptions: &v1.SELinuxOptions{},
				WindowsOptions: &v1.WindowsSecurityContextOptions{GMSACredentialSpecName: func() *string {
					name := ""
					return &name
				}(), GMSACredentialSpec: func() *string {
					spec := ""
					return &spec
				}(), RunAsUserName: func() *string {
					run := ""
					return &run
				}()},
				RunAsUser:    new(int64),
				RunAsGroup:   new(int64),
				RunAsNonRoot: new(bool),
				FSGroup:      new(int64),
				FSGroupChangePolicy: func() *v1.PodFSGroupChangePolicy {
					f := v1.PodFSGroupChangePolicy("")
					return &f
				}(),
			},
			Affinity:           &v1.Affinity{},
			Priority:           new(int32),
			DNSConfig:          &v1.PodDNSConfig{},
			EnableServiceLinks: new(bool),
		},
	}
}
