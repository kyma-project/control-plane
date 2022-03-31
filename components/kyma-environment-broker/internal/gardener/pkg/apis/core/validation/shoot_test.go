// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validation_test

import (
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/utils/pointer"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/gardener/pkg/apis/core"
	v1beta1constants "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/gardener/pkg/apis/core/v1beta1/constants"
	. "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/gardener/pkg/apis/core/validation"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/gardener/pkg/features"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/gardener/pkg/utils/test"
	. "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/gardener/pkg/utils/test/matchers"
)

var _ = Describe("Shoot Validation Tests", func() {
	Describe("#ValidateShoot, #ValidateShootUpdate", func() {
		var (
			shoot *core.Shoot

			domain          = "my-cluster.example.com"
			dnsProviderType = "some-provider"
			secretName      = "some-secret"
			purpose         = core.ShootPurposeEvaluation
			addon           = core.Addon{
				Enabled: true,
			}

			maxSurge         = intstr.FromInt(1)
			maxUnavailable   = intstr.FromInt(0)
			systemComponents = &core.WorkerSystemComponents{
				Allow: true,
			}
			worker = core.Worker{
				Name: "worker-name",
				Machine: core.Machine{
					Type: "large",
					Image: &core.ShootMachineImage{
						Name:    "image-name",
						Version: "1.0.0",
					},
				},
				Minimum:          1,
				Maximum:          1,
				MaxSurge:         &maxSurge,
				MaxUnavailable:   &maxUnavailable,
				SystemComponents: systemComponents,
			}
			invalidWorker = core.Worker{
				Name: "",
				Machine: core.Machine{
					Type: "",
				},
				Minimum:          -1,
				Maximum:          -2,
				MaxSurge:         &maxSurge,
				MaxUnavailable:   &maxUnavailable,
				SystemComponents: systemComponents,
			}
			invalidWorkerName = core.Worker{
				Name: "not_compliant",
				Machine: core.Machine{
					Type: "large",
					Image: &core.ShootMachineImage{
						Name:    "image-name",
						Version: "1.0.0",
					},
				},
				Minimum:          1,
				Maximum:          1,
				MaxSurge:         &maxSurge,
				MaxUnavailable:   &maxUnavailable,
				SystemComponents: systemComponents,
			}
			invalidWorkerTooLongName = core.Worker{
				Name: "worker-name-is-too-long",
				Machine: core.Machine{
					Type: "large",
					Image: &core.ShootMachineImage{
						Name:    "image-name",
						Version: "1.0.0",
					},
				},
				Minimum:          1,
				Maximum:          1,
				MaxSurge:         &maxSurge,
				MaxUnavailable:   &maxUnavailable,
				SystemComponents: systemComponents,
			}
			workerAutoScalingMinZero = core.Worker{
				Name: "cpu-worker",
				Machine: core.Machine{
					Type: "large",
					Image: &core.ShootMachineImage{
						Name:    "image-name",
						Version: "1.0.0",
					},
				},
				Minimum:          0,
				Maximum:          2,
				MaxSurge:         &maxSurge,
				MaxUnavailable:   &maxUnavailable,
				SystemComponents: systemComponents,
			}
			workerAutoScalingMinMaxZero = core.Worker{
				Name: "cpu-worker",
				Machine: core.Machine{
					Type: "large",
					Image: &core.ShootMachineImage{
						Name:    "image-name",
						Version: "1.0.0",
					},
				},
				Minimum:          0,
				Maximum:          0,
				MaxSurge:         &maxSurge,
				MaxUnavailable:   &maxUnavailable,
				SystemComponents: systemComponents,
			}
		)

		BeforeEach(func() {
			shoot = &core.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "shoot",
					Namespace: "my-namespace",
				},
				Spec: core.ShootSpec{
					Addons: &core.Addons{
						KubernetesDashboard: &core.KubernetesDashboard{
							Addon: addon,
						},
						NginxIngress: &core.NginxIngress{
							Addon: addon,
						},
					},
					CloudProfileName:  "aws-profile",
					Region:            "eu-west-1",
					SecretBindingName: "my-secret",
					Purpose:           &purpose,
					DNS: &core.DNS{
						Providers: []core.DNSProvider{
							{
								Type:    &dnsProviderType,
								Primary: pointer.Bool(true),
							},
						},
						Domain: &domain,
					},
					Kubernetes: core.Kubernetes{
						Version: "1.20.2",
						KubeAPIServer: &core.KubeAPIServerConfig{
							OIDCConfig: &core.OIDCConfig{
								CABundle:       pointer.String("-----BEGIN CERTIFICATE-----\nMIICRzCCAfGgAwIBAgIJALMb7ecMIk3MMA0GCSqGSIb3DQEBCwUAMH4xCzAJBgNV\nBAYTAkdCMQ8wDQYDVQQIDAZMb25kb24xDzANBgNVBAcMBkxvbmRvbjEYMBYGA1UE\nCgwPR2xvYmFsIFNlY3VyaXR5MRYwFAYDVQQLDA1JVCBEZXBhcnRtZW50MRswGQYD\nVQQDDBJ0ZXN0LWNlcnRpZmljYXRlLTAwIBcNMTcwNDI2MjMyNjUyWhgPMjExNzA0\nMDIyMzI2NTJaMH4xCzAJBgNVBAYTAkdCMQ8wDQYDVQQIDAZMb25kb24xDzANBgNV\nBAcMBkxvbmRvbjEYMBYGA1UECgwPR2xvYmFsIFNlY3VyaXR5MRYwFAYDVQQLDA1J\nVCBEZXBhcnRtZW50MRswGQYDVQQDDBJ0ZXN0LWNlcnRpZmljYXRlLTAwXDANBgkq\nhkiG9w0BAQEFAANLADBIAkEAtBMa7NWpv3BVlKTCPGO/LEsguKqWHBtKzweMY2CV\ntAL1rQm913huhxF9w+ai76KQ3MHK5IVnLJjYYA5MzP2H5QIDAQABo1AwTjAdBgNV\nHQ4EFgQU22iy8aWkNSxv0nBxFxerfsvnZVMwHwYDVR0jBBgwFoAU22iy8aWkNSxv\n0nBxFxerfsvnZVMwDAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQsFAANBAEOefGbV\nNcHxklaW06w6OBYJPwpIhCVozC1qdxGX1dg8VkEKzjOzjgqVD30m59OFmSlBmHsl\nnkVA6wyOSDYBf3o=\n-----END CERTIFICATE-----"),
								ClientID:       pointer.String("client-id"),
								GroupsClaim:    pointer.String("groups-claim"),
								GroupsPrefix:   pointer.String("groups-prefix"),
								IssuerURL:      pointer.String("https://some-endpoint.com"),
								UsernameClaim:  pointer.String("user-claim"),
								UsernamePrefix: pointer.String("user-prefix"),
								RequiredClaims: map[string]string{"foo": "bar"},
							},
							AdmissionPlugins: []core.AdmissionPlugin{
								{
									Name: "PodNodeSelector",
									Config: &runtime.RawExtension{
										Raw: []byte(`podNodeSelectorPluginConfig:
  clusterDefaultNodeSelector: <node-selectors-labels>
  namespace1: <node-selectors-labels>
	namespace2: <node-selectors-labels>`),
									},
								},
							},
							AuditConfig: &core.AuditConfig{
								AuditPolicy: &core.AuditPolicy{
									ConfigMapRef: &corev1.ObjectReference{
										Name: "audit-policy-config",
									},
								},
							},
							EnableBasicAuthentication: pointer.Bool(false),
						},
						KubeControllerManager: &core.KubeControllerManagerConfig{
							NodeCIDRMaskSize: pointer.Int32(22),
							HorizontalPodAutoscalerConfig: &core.HorizontalPodAutoscalerConfig{
								SyncPeriod: makeDurationPointer(30 * time.Second),
								Tolerance:  pointer.Float64(0.1),
							},
						},
					},
					Networking: core.Networking{
						Type: "some-network-plugin",
					},
					Provider: core.Provider{
						Type:    "aws",
						Workers: []core.Worker{worker},
					},
					Maintenance: &core.Maintenance{
						AutoUpdate: &core.MaintenanceAutoUpdate{
							KubernetesVersion: true,
						},
						TimeWindow: &core.MaintenanceTimeWindow{
							Begin: "220000+0100",
							End:   "230000+0100",
						},
					},
					Monitoring: &core.Monitoring{
						Alerting: &core.Alerting{},
					},
					Tolerations: []core.Toleration{
						{Key: "foo"},
					},
				},
			}
		})

		DescribeTable("Shoot metadata",
			func(objectMeta metav1.ObjectMeta, matcher gomegatypes.GomegaMatcher) {
				shoot.ObjectMeta = objectMeta

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(matcher)
			},

			Entry("should forbid Shoot with empty metadata",
				metav1.ObjectMeta{},
				ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("metadata.name"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("metadata.namespace"),
					})),
				),
			),
			Entry("should forbid Shoot with empty name",
				metav1.ObjectMeta{Name: "", Namespace: "my-namespace"},
				ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("metadata.name"),
				}))),
			),
			Entry("should forbid Shoot with '.' in the name (not a DNS-1123 label compliant name)",
				metav1.ObjectMeta{Name: "shoot.test", Namespace: "my-namespace"},
				ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("metadata.name"),
				}))),
			),
			Entry("should forbid Shoot with '_' in the name (not a DNS-1123 subdomain)",
				metav1.ObjectMeta{Name: "shoot_test", Namespace: "my-namespace"},
				ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("metadata.name"),
				}))),
			),
			Entry("should forbid Shoot with name containing two consecutive hyphens",
				metav1.ObjectMeta{Name: "sho--ot", Namespace: "my-namespace"},
				ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("metadata.name"),
				}))),
			),
		)

		It("should forbid empty Shoot resources", func() {
			shoot := &core.Shoot{
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       core.ShootSpec{},
			}

			errorList := ValidateShoot(shoot)

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("metadata.name"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("metadata.namespace"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.kubernetes.version"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.networking.type"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.provider.type"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("spec.provider.workers"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("spec.provider.workers"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.cloudProfileName"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.region"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.secretBindingName"),
				})),
			))
		})

		Context("exposure class", func() {
			It("should pass as exposure class is not changed", func() {
				shoot.Spec.ExposureClassName = pointer.String("exposure-class-1")
				newShoot := prepareShootForUpdate(shoot)

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(errorList).To(HaveLen(0))
			})

			It("should forbid to change the exposure class", func() {
				shoot.Spec.ExposureClassName = pointer.String("exposure-class-1")
				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.ExposureClassName = pointer.String("exposure-class-2")

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(errorList).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.exposureClassName"),
					})),
				))
			})
		})

		DescribeTable("purpose validation",
			func(purpose core.ShootPurpose, namespace string, matcher gomegatypes.GomegaMatcher) {
				shootCopy := shoot.DeepCopy()
				shootCopy.Namespace = namespace
				shootCopy.Spec.Purpose = &purpose
				errorList := ValidateShoot(shootCopy)
				Expect(errorList).To(matcher)
			},

			Entry("evaluation purpose", core.ShootPurposeEvaluation, "dev", BeEmpty()),
			Entry("testing purpose", core.ShootPurposeTesting, "dev", BeEmpty()),
			Entry("development purpose", core.ShootPurposeDevelopment, "dev", BeEmpty()),
			Entry("production purpose", core.ShootPurposeProduction, "dev", BeEmpty()),
			Entry("infrastructure purpose in garden namespace", core.ShootPurposeInfrastructure, "garden", BeEmpty()),
			Entry("infrastructure purpose in other namespace", core.ShootPurposeInfrastructure, "dev", ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeNotSupported),
				"Field": Equal("spec.purpose"),
			})))),
			Entry("unknown purpose", core.ShootPurpose("foo"), "dev", ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeNotSupported),
				"Field": Equal("spec.purpose"),
			})))),
		)

		DescribeTable("addons validation",
			func(purpose core.ShootPurpose, version string, allowed bool) {
				shootCopy := shoot.DeepCopy()
				shootCopy.Spec.Purpose = &purpose
				shootCopy.Spec.Kubernetes.Version = version
				shootCopy.Spec.Kubernetes.KubeAPIServer.EnableBasicAuthentication = pointer.Bool(false)

				errorList := ValidateShoot(shootCopy)

				if allowed {
					Expect(errorList).To(BeEmpty())
				} else {
					Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeForbidden),
						"Field": Equal("spec.addons"),
					}))))
				}
			},
			Entry("should allow addons on evaluation shoots with version >= 1.22", core.ShootPurposeEvaluation, "1.22.0", true),
			Entry("should forbid addons on testing shoots with version >= 1.22", core.ShootPurposeTesting, "1.22.0", false),
			Entry("should forbid addons on development shoots with version >= 1.22", core.ShootPurposeDevelopment, "1.22.0", false),
			Entry("should forbid addons on production shoots with version >= 1.22", core.ShootPurposeProduction, "1.22.0", false),
			Entry("should allow addons on evaluation shoots with a pre-release version >= 1.22", core.ShootPurposeEvaluation, "1.22.0-alpha.1", true),
			Entry("should forbid addons on production shoots with a pre-release version >= 1.22", core.ShootPurposeProduction, "1.22.0-alpha.1", false),
			Entry("should forbid addons on development shoots with a pre-release version >= 1.22", core.ShootPurposeDevelopment, "1.22.0-alpha.1", false),
			Entry("should forbid addons on production shoots with a pre-release version >= 1.22", core.ShootPurposeProduction, "1.22.0-alpha.1", false),
			Entry("should allow addons on evaluation shoots with version < 1.22", core.ShootPurposeEvaluation, "1.21.10", true),
			Entry("should allow addons on testing shoots with version < 1.22", core.ShootPurposeTesting, "1.21.10", true),
			Entry("should allow addons on development shoots with version < 1.22", core.ShootPurposeDevelopment, "1.21.10", true),
			Entry("should allow addons on production shoots with version < 1.22", core.ShootPurposeProduction, "1.21.10", true),
		)

		It("should forbid unsupported addon configuration", func() {
			shoot.Spec.Addons.KubernetesDashboard.AuthenticationMode = pointer.String("does-not-exist")

			errorList := ValidateShoot(shoot)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeNotSupported),
				"Field": Equal("spec.addons.kubernetesDashboard.authenticationMode"),
			}))))
		})

		It("should allow external traffic policies 'Cluster' for nginx-ingress", func() {
			v := corev1.ServiceExternalTrafficPolicyTypeCluster
			shoot.Spec.Addons.NginxIngress.ExternalTrafficPolicy = &v
			errorList := ValidateShoot(shoot)
			Expect(errorList).To(BeEmpty())
		})

		It("should allow external traffic policies 'Local' for nginx-ingress", func() {
			v := corev1.ServiceExternalTrafficPolicyTypeLocal
			shoot.Spec.Addons.NginxIngress.ExternalTrafficPolicy = &v
			errorList := ValidateShoot(shoot)
			Expect(errorList).To(BeEmpty())
		})

		It("should forbid unsupported external traffic policies for nginx-ingress", func() {
			v := corev1.ServiceExternalTrafficPolicyType("something-else")
			shoot.Spec.Addons.NginxIngress.ExternalTrafficPolicy = &v

			errorList := ValidateShoot(shoot)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeNotSupported),
				"Field": Equal("spec.addons.nginxIngress.externalTrafficPolicy"),
			}))))
		})

		It("should forbid enabling the nginx-ingress addon for shooted seeds if it was disabled", func() {
			newShoot := prepareShootForUpdate(shoot)

			metav1.SetMetaDataAnnotation(&shoot.ObjectMeta, v1beta1constants.AnnotationShootUseAsSeed, "true")
			shoot.Spec.Addons.NginxIngress.Enabled = false
			metav1.SetMetaDataAnnotation(&newShoot.ObjectMeta, v1beta1constants.AnnotationShootUseAsSeed, "true")
			newShoot.Spec.Addons.NginxIngress.Enabled = true

			errorList := ValidateShootUpdate(newShoot, shoot)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeForbidden),
				"Field": Equal("spec.addons.nginxIngress.enabled"),
			}))))
		})

		It("should allow enabling the nginx-ingress addon for shoots if it was disabled", func() {
			newShoot := prepareShootForUpdate(shoot)
			shoot.Spec.Addons.NginxIngress.Enabled = false
			newShoot.Spec.Addons.NginxIngress.Enabled = true

			errorList := ValidateShootUpdate(newShoot, shoot)

			Expect(errorList).To(BeEmpty())
		})

		It("should forbid using basic auth mode for kubernetes dashboard when it's disabled in kube-apiserver config", func() {
			shoot.Spec.Addons.KubernetesDashboard.AuthenticationMode = pointer.String(core.KubernetesDashboardAuthModeBasic)
			shoot.Spec.Kubernetes.KubeAPIServer.EnableBasicAuthentication = pointer.Bool(false)

			errorList := ValidateShoot(shoot)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.addons.kubernetesDashboard.authenticationMode"),
			}))))
		})

		It("should allow using basic auth mode for kubernetes dashboard when it's enabled in kube-apiserver config", func() {
			shoot.Spec.Kubernetes.Version = "1.18.9"
			shoot.Spec.Addons.KubernetesDashboard.AuthenticationMode = pointer.String(core.KubernetesDashboardAuthModeBasic)
			shoot.Spec.Kubernetes.KubeAPIServer.EnableBasicAuthentication = pointer.Bool(true)

			errorList := ValidateShoot(shoot)

			Expect(errorList).To(BeEmpty())
		})

		It("should forbid unsupported specification (provider independent)", func() {
			shoot.Spec.CloudProfileName = ""
			shoot.Spec.Region = ""
			shoot.Spec.SecretBindingName = ""
			shoot.Spec.SeedName = pointer.String("")
			shoot.Spec.SeedSelector = &core.SeedSelector{
				LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{"foo": "no/slash/allowed"}},
			}
			shoot.Spec.Provider.Type = ""

			errorList := ValidateShoot(shoot)

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.cloudProfileName"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.region"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.secretBindingName"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.seedName"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.seedSelector.matchLabels"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.provider.type"),
				})),
			))
		})

		It("should forbid adding invalid/duplicate emails", func() {
			shoot.Spec.Monitoring.Alerting.EmailReceivers = []string{
				"z",
				"foo@bar.baz",
				"foo@bar.baz",
			}

			errorList := ValidateShoot(shoot)

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("spec.monitoring.alerting.emailReceivers[0]"),
					"Detail": Equal("must provide a valid email"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeDuplicate),
					"Field": Equal("spec.monitoring.alerting.emailReceivers[2]"),
				})),
			))
		})

		It("should forbid invalid tolerations", func() {
			shoot.Spec.Tolerations = []core.Toleration{
				{},
				{Key: "foo"},
				{Key: "foo"},
				{Key: "bar", Value: pointer.String("baz")},
				{Key: "bar", Value: pointer.String("baz")},
			}

			errorList := ValidateShoot(shoot)

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.tolerations[0].key"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeDuplicate),
					"Field": Equal("spec.tolerations[2]"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeDuplicate),
					"Field": Equal("spec.tolerations[4]"),
				})),
			))
		})

		It("should forbid updating some cloud keys", func() {
			newShoot := prepareShootForUpdate(shoot)
			shoot.Spec.CloudProfileName = "another-profile"
			shoot.Spec.Region = "another-region"
			shoot.Spec.SecretBindingName = "another-reference"
			shoot.Spec.Provider.Type = "another-provider"

			errorList := ValidateShootUpdate(newShoot, shoot)

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.cloudProfileName"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.region"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.secretBindingName"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.provider.type"),
				})),
			))
		})

		It("should forbid updating the seed if it has been set previously and the SeedChange feature gate is not enabled", func() {
			defer test.WithFeatureGate(utilfeature.DefaultFeatureGate, features.SeedChange, false)()

			newShoot := prepareShootForUpdate(shoot)
			newShoot.Spec.SeedName = pointer.String("another-seed")
			shoot.Spec.SeedName = pointer.String("first-seed")

			errorList := ValidateShootUpdate(newShoot, shoot)

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.seedName"),
				}))),
			)
		})

		It("should allow updating the seed if it has been set previously and the SeedChange feature gate is enabled", func() {
			defer test.WithFeatureGate(utilfeature.DefaultFeatureGate, features.SeedChange, true)()

			newShoot := prepareShootForUpdate(shoot)
			newShoot.Spec.SeedName = pointer.String("another-seed")
			shoot.Spec.SeedName = pointer.String("first-seed")

			errorList := ValidateShootUpdate(newShoot, shoot)

			Expect(errorList).To(BeEmpty())
		})

		It("should forbid passing an extension w/o type information", func() {
			extension := core.Extension{}
			shoot.Spec.Extensions = append(shoot.Spec.Extensions, extension)

			errorList := ValidateShoot(shoot)

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.extensions[0].type"),
				}))))
		})

		It("should allow passing an extension w/ type information", func() {
			extension := core.Extension{
				Type: "arbitrary",
			}
			shoot.Spec.Extensions = append(shoot.Spec.Extensions, extension)

			errorList := ValidateShoot(shoot)

			Expect(errorList).To(BeEmpty())
		})

		It("should forbid resources w/o names or w/ invalid references", func() {
			ref := core.NamedResourceReference{}
			shoot.Spec.Resources = append(shoot.Spec.Resources, ref)

			errorList := ValidateShoot(shoot)

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.resources[0].name"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.resources[0].resourceRef.kind"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.resources[0].resourceRef.name"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.resources[0].resourceRef.apiVersion"),
				})),
			))
		})

		It("should forbid resources with non-unique names", func() {
			ref := core.NamedResourceReference{
				Name: "test",
				ResourceRef: autoscalingv1.CrossVersionObjectReference{
					Kind:       "Secret",
					Name:       "test-secret",
					APIVersion: "v1",
				},
			}
			shoot.Spec.Resources = append(shoot.Spec.Resources, ref, ref)

			errorList := ValidateShoot(shoot)

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeDuplicate),
					"Field": Equal("spec.resources[1].name"),
				})),
			))
		})

		It("should allow resources w/ names and valid references", func() {
			ref := core.NamedResourceReference{
				Name: "test",
				ResourceRef: autoscalingv1.CrossVersionObjectReference{
					Kind:       "Secret",
					Name:       "test-secret",
					APIVersion: "v1",
				},
			}
			shoot.Spec.Resources = append(shoot.Spec.Resources, ref)

			errorList := ValidateShoot(shoot)

			Expect(errorList).To(BeEmpty())
		})

		It("should allow updating the seed if it has not been set previously", func() {
			newShoot := prepareShootForUpdate(shoot)
			newShoot.Spec.SeedName = pointer.String("another-seed")
			shoot.Spec.SeedName = nil

			errorList := ValidateShootUpdate(newShoot, shoot)

			Expect(errorList).To(HaveLen(0))
		})

		Context("Provider validation", func() {
			BeforeEach(func() {
				provider := core.Provider{
					Type:    "foo",
					Workers: []core.Worker{worker},
				}

				shoot.Spec.Provider = provider
			})

			It("should not return any errors", func() {
				errorList := ValidateShoot(shoot)

				Expect(errorList).To(BeEmpty())
			})

			It("should invalid k8s networks", func() {
				invalidCIDR := "invalid-cidr"

				shoot.Spec.Networking.Nodes = &invalidCIDR
				shoot.Spec.Networking.Services = &invalidCIDR
				shoot.Spec.Networking.Pods = &invalidCIDR

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("spec.networking.nodes"),
					"Detail": ContainSubstring("invalid CIDR address"),
				}, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("spec.networking.pods"),
					"Detail": ContainSubstring("invalid CIDR address"),
				}, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("spec.networking.services"),
					"Detail": ContainSubstring("invalid CIDR address"),
				}))
			})

			It("should forbid non canonical CIDRs", func() {
				nodeCIDR := "10.250.0.3/16"
				podCIDR := "100.96.0.4/11"
				serviceCIDR := "100.64.0.5/13"

				shoot.Spec.Networking.Nodes = &nodeCIDR
				shoot.Spec.Networking.Services = &serviceCIDR
				shoot.Spec.Networking.Pods = &podCIDR

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("spec.networking.nodes"),
					"Detail": Equal("must be valid canonical CIDR"),
				}, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("spec.networking.pods"),
					"Detail": Equal("must be valid canonical CIDR"),
				}, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("spec.networking.services"),
					"Detail": Equal("must be valid canonical CIDR"),
				}))
			})

			It("should forbid an empty worker list", func() {
				shoot.Spec.Provider.Workers = []core.Worker{}

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeForbidden),
						"Field": Equal("spec.provider.workers"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeForbidden),
						"Field": Equal("spec.provider.workers"),
					})),
				))
			})

			It("should enforce unique worker names", func() {
				shoot.Spec.Provider.Workers = []core.Worker{worker, worker}

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeDuplicate),
					"Field": Equal("spec.provider.workers[1].name"),
				}))))
			})

			It("should forbid invalid worker configuration", func() {
				w := invalidWorker.DeepCopy()
				shoot.Spec.Provider.Workers = []core.Worker{*w}

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.provider.workers[0].name"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("spec.provider.workers[0].machine.type"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.provider.workers[0].minimum"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.provider.workers[0].maximum"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeForbidden),
						"Field": Equal("spec.provider.workers[0].maximum"),
					})),
				))
			})

			It("should allow workers min = 0 if max > 0", func() {
				shoot.Spec.Provider.Workers = []core.Worker{workerAutoScalingMinZero, worker}

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(BeEmpty())
			})

			It("should allow workers having min=max=0 if at least one pool is active", func() {
				shoot.Spec.Provider.Workers = []core.Worker{worker, workerAutoScalingMinMaxZero}

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(BeEmpty())
			})

			It("should forbid too long worker names", func() {
				shoot.Spec.Provider.Workers[0] = invalidWorkerTooLongName

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeTooLong),
					"Field": Equal("spec.provider.workers[0].name"),
				}))))
			})

			It("should forbid worker pools with names that are not DNS-1123 label compliant", func() {
				shoot.Spec.Provider.Workers = []core.Worker{invalidWorkerName}

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.provider.workers[0].name"),
				}))))
			})

			Context("NodeCIDRMask validation", func() {
				var (
					defaultMaxPod           int32 = 110
					maxPod                  int32 = 260
					defaultNodeCIDRMaskSize int32 = 24
					testWorker              core.Worker
				)

				BeforeEach(func() {
					shoot.Spec.Kubernetes.KubeControllerManager.NodeCIDRMaskSize = &defaultNodeCIDRMaskSize
					shoot.Spec.Kubernetes.Kubelet = &core.KubeletConfig{MaxPods: &defaultMaxPod}
					testWorker = *worker.DeepCopy()
					testWorker.Name = "testworker"
				})

				It("should not return any errors", func() {
					worker.Kubernetes = &core.WorkerKubernetes{
						Kubelet: &core.KubeletConfig{
							MaxPods: &defaultMaxPod,
						},
					}

					errorList := ValidateShoot(shoot)

					Expect(errorList).To(HaveLen(0))
				})

				Context("Non-default max pod settings", func() {
					Context("one worker pool", func() {
						It("should deny NodeCIDR with too few ips", func() {
							testWorker.Kubernetes = &core.WorkerKubernetes{
								Kubelet: &core.KubeletConfig{
									MaxPods: &maxPod,
								},
							}
							shoot.Spec.Provider.Workers = append(shoot.Spec.Provider.Workers, testWorker)

							errorList := ValidateShoot(shoot)

							Expect(errorList).To(ConsistOfFields(Fields{
								"Type":   Equal(field.ErrorTypeInvalid),
								"Field":  Equal("spec.kubernetes.kubeControllerManager.nodeCIDRMaskSize"),
								"Detail": ContainSubstring("kubelet or kube-controller configuration incorrect"),
							}))
						})
					})

					Context("multiple worker pools", func() {
						It("should deny NodeCIDR with too few ips", func() {
							testWorker.Kubernetes = &core.WorkerKubernetes{
								Kubelet: &core.KubeletConfig{
									MaxPods: &maxPod,
								},
							}

							secondTestWorker := *testWorker.DeepCopy()
							secondTestWorker.Name = "testworker2"
							secondTestWorker.Kubernetes = &core.WorkerKubernetes{
								Kubelet: &core.KubeletConfig{
									MaxPods: &maxPod,
								},
							}

							shoot.Spec.Provider.Workers = append(shoot.Spec.Provider.Workers, testWorker, secondTestWorker)

							errorList := ValidateShoot(shoot)

							Expect(errorList).To(ConsistOfFields(Fields{
								"Type":   Equal(field.ErrorTypeInvalid),
								"Field":  Equal("spec.kubernetes.kubeControllerManager.nodeCIDRMaskSize"),
								"Detail": ContainSubstring("kubelet or kube-controller configuration incorrect"),
							}))
						})
					})

					Context("Global default max pod", func() {
						It("should deny NodeCIDR with too few ips", func() {
							shoot.Spec.Kubernetes.Kubelet = &core.KubeletConfig{MaxPods: &maxPod}

							errorList := ValidateShoot(shoot)

							Expect(errorList).To(ConsistOfFields(Fields{
								"Type":   Equal(field.ErrorTypeInvalid),
								"Field":  Equal("spec.kubernetes.kubeControllerManager.nodeCIDRMaskSize"),
								"Detail": ContainSubstring("kubelet or kube-controller configuration incorrect"),
							}))
						})
					})
				})
			})

			It("should allow adding a worker pool", func() {
				newShoot := prepareShootForUpdate(shoot)

				worker := *shoot.Spec.Provider.Workers[0].DeepCopy()
				worker.Name = "second-worker"

				newShoot.Spec.Provider.Workers = append(newShoot.Spec.Provider.Workers, worker)

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(errorList).To(HaveLen(0))
			})

			It("should allow removing a worker pool", func() {
				newShoot := prepareShootForUpdate(shoot)

				worker := *shoot.Spec.Provider.Workers[0].DeepCopy()
				worker.Name = "second-worker"

				shoot.Spec.Provider.Workers = append(shoot.Spec.Provider.Workers, worker)

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(errorList).To(HaveLen(0))
			})

			It("should allow swapping worker pools", func() {
				newShoot := prepareShootForUpdate(shoot)

				worker := *shoot.Spec.Provider.Workers[0].DeepCopy()
				worker.Name = "second-worker"

				newShoot.Spec.Provider.Workers = append(newShoot.Spec.Provider.Workers, worker)
				shoot.Spec.Provider.Workers = append(shoot.Spec.Provider.Workers, worker)

				newShoot.Spec.Provider.Workers = []core.Worker{newShoot.Spec.Provider.Workers[1], newShoot.Spec.Provider.Workers[0]}

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(errorList).To(HaveLen(0))
			})

			Context("Worker nodes max count validation", func() {
				var (
					worker1 = worker.DeepCopy()
					worker2 = worker.DeepCopy()
				)
				worker1.Name = "worker1"
				worker2.Name = "worker2"

				It("should allow valid total number of worker nodes", func() {
					shoot.Spec.Kubernetes.KubeControllerManager.NodeCIDRMaskSize = pointer.Int32(24)
					shoot.Spec.Networking.Pods = pointer.String("100.96.0.0/20")
					worker1.Maximum = 4
					worker2.Maximum = 4

					shoot.Spec.Provider.Workers = []core.Worker{
						*worker1,
						*worker2,
					}

					errorList := ValidateTotalNodeCountWithPodCIDR(shoot)

					Expect(errorList).To(BeEmpty())
				})

				It("should allow valid total number of worker nodes", func() {
					shoot.Spec.Kubernetes.KubeControllerManager.NodeCIDRMaskSize = pointer.Int32(24)
					shoot.Spec.Networking.Pods = pointer.String("100.96.0.0/16")
					worker1.Maximum = 128
					worker2.Maximum = 128

					shoot.Spec.Provider.Workers = []core.Worker{
						*worker1,
						*worker2,
					}

					errorList := ValidateTotalNodeCountWithPodCIDR(shoot)

					Expect(errorList).To(BeEmpty())
				})

				It("should not allow invalid total number of worker nodes", func() {
					shoot.Spec.Kubernetes.KubeControllerManager.NodeCIDRMaskSize = pointer.Int32(24)
					shoot.Spec.Networking.Pods = pointer.String("100.96.0.0/20")
					worker1.Maximum = 16
					worker2.Maximum = 16

					shoot.Spec.Provider.Workers = []core.Worker{
						*worker1,
						*worker2,
					}

					errorList := ValidateTotalNodeCountWithPodCIDR(shoot)

					Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.provider.workers"),
					}))))
				})

				It("should not allow ivalid total number of worker nodes", func() {
					shoot.Spec.Kubernetes.KubeControllerManager.NodeCIDRMaskSize = pointer.Int32(24)
					shoot.Spec.Networking.Pods = pointer.String("100.96.0.0/16")
					worker1.Maximum = 128
					worker2.Maximum = 129

					shoot.Spec.Provider.Workers = []core.Worker{
						*worker1,
						*worker2,
					}

					errorList := ValidateTotalNodeCountWithPodCIDR(shoot)

					Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.provider.workers"),
					}))))
				})
			})
		})

		Context("dns section", func() {
			It("should forbid specifying a provider without a domain", func() {
				shoot.Spec.DNS.Domain = nil

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.dns.domain"),
				}))))
			})

			It("should allow specifying the 'unmanaged' provider without a domain", func() {
				shoot.Spec.DNS.Domain = nil
				shoot.Spec.DNS.Providers = []core.DNSProvider{
					{
						Type:    pointer.String(core.DNSUnmanaged),
						Primary: pointer.Bool(true),
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(HaveLen(0))
			})

			It("should forbid specifying invalid domain", func() {
				shoot.Spec.DNS.Providers = nil
				shoot.Spec.DNS.Domain = pointer.String("foo/bar.baz")

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.dns.domain"),
				}))))
			})

			It("should forbid specifying a secret name when provider equals 'unmanaged'", func() {
				shoot.Spec.DNS.Providers = []core.DNSProvider{
					{
						Type:       pointer.String(core.DNSUnmanaged),
						SecretName: pointer.String(""),
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.dns.providers[0].secretName"),
				}))))
			})

			It("should require a provider if a secret name is given", func() {
				shoot.Spec.DNS.Providers = []core.DNSProvider{
					{
						SecretName: pointer.String(""),
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.dns.providers[0].type"),
				}))))
			})

			It("should allow assigning the dns domain (dns nil)", func() {
				oldShoot := prepareShootForUpdate(shoot)
				oldShoot.Spec.DNS = nil
				newShoot := prepareShootForUpdate(oldShoot)
				newShoot.Spec.DNS = &core.DNS{
					Domain: pointer.String("some-domain.com"),
				}

				errorList := ValidateShootUpdate(newShoot, oldShoot)

				Expect(errorList).To(BeEmpty())
			})

			It("should allow assigning the dns domain (dns non-nil)", func() {
				oldShoot := prepareShootForUpdate(shoot)
				oldShoot.Spec.DNS = &core.DNS{}
				newShoot := prepareShootForUpdate(oldShoot)
				newShoot.Spec.DNS.Domain = pointer.String("some-domain.com")

				errorList := ValidateShootUpdate(newShoot, oldShoot)

				Expect(errorList).To(BeEmpty())
			})

			It("should forbid removing the dns section", func() {
				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.DNS = nil

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.dns"),
				}))))
			})

			It("should forbid updating the dns domain", func() {
				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.DNS.Domain = pointer.String("another-domain.com")

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.dns.domain"),
				}))))
			})

			It("should allow updating the dns providers if seed is assigned", func() {
				oldShoot := shoot.DeepCopy()
				oldShoot.Spec.SeedName = nil
				oldShoot.Spec.DNS.Providers[0].Type = pointer.String("some-dns-provider")

				newShoot := prepareShootForUpdate(oldShoot)
				newShoot.Spec.SeedName = pointer.String("seed")
				newShoot.Spec.DNS.Providers = nil

				errorList := ValidateShootUpdate(newShoot, oldShoot)

				Expect(errorList).To(BeEmpty())
			})

			It("should forbid updating the primary dns provider type", func() {
				newShoot := prepareShootForUpdate(shoot)
				shoot.Spec.DNS.Providers[0].Type = pointer.String("some-other-provider")

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("spec.dns.providers"),
				}))))
			})

			It("should forbid to unset the primary DNS provider type", func() {
				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.DNS.Providers[0].Type = nil

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("spec.dns.providers"),
				}))))
			})

			It("should forbid to remove the primary DNS provider", func() {
				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.DNS.Providers[0].Primary = pointer.Bool(false)

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("spec.dns.providers"),
				}))))
			})

			It("should forbid adding another primary provider", func() {
				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.DNS.Providers = append(newShoot.Spec.DNS.Providers, core.DNSProvider{
					Primary: pointer.Bool(true),
				})

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("spec.dns.providers[1].primary"),
				}))))
			})

			It("should having the a provider with invalid secretName", func() {
				invalidSecretName := "foo/bar"

				shoot.Spec.DNS.Providers = []core.DNSProvider{
					{
						SecretName: &secretName,
						Type:       &dnsProviderType,
					},
					{
						SecretName: &invalidSecretName,
						Type:       &dnsProviderType,
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.dns.providers[1]"),
				}))))
			})

			It("should having the same provider multiple times", func() {
				shoot.Spec.DNS.Providers = []core.DNSProvider{
					{
						SecretName: &secretName,
						Type:       &dnsProviderType,
					},
					{
						SecretName: &secretName,
						Type:       &dnsProviderType,
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.dns.providers[1]"),
				}))))
			})

			It("should allow updating the dns secret name", func() {
				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.DNS.Providers[0].SecretName = pointer.String("my-dns-secret")

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(errorList).To(HaveLen(0))
			})

			It("should forbid having more than one primary provider", func() {
				shoot.Spec.DNS.Providers = append(shoot.Spec.DNS.Providers, core.DNSProvider{
					Primary: pointer.Bool(true),
				})

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("spec.dns.providers[1].primary"),
				}))))
			})
		})

		Context("OIDC validation", func() {
			It("should forbid unsupported OIDC configuration", func() {
				shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.CABundle = pointer.String("")
				shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.ClientID = pointer.String("")
				shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.GroupsClaim = pointer.String("")
				shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.GroupsPrefix = pointer.String("")
				shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.IssuerURL = pointer.String("")
				shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.UsernameClaim = pointer.String("")
				shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.UsernamePrefix = pointer.String("")
				shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.RequiredClaims = map[string]string{}
				shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.SigningAlgs = []string{"foo"}

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeAPIServer.oidcConfig.issuerURL"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeAPIServer.oidcConfig.clientID"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeAPIServer.oidcConfig.caBundle"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeAPIServer.oidcConfig.groupsClaim"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeAPIServer.oidcConfig.groupsPrefix"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeNotSupported),
					"Field":  Equal("spec.kubernetes.kubeAPIServer.oidcConfig.signingAlgs[0]"),
					"Detail": Equal("supported values: \"ES256\", \"ES384\", \"ES512\", \"PS256\", \"PS384\", \"PS512\", \"RS256\", \"RS384\", \"RS512\", \"none\""),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeAPIServer.oidcConfig.usernameClaim"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeAPIServer.oidcConfig.usernamePrefix"),
				}))))
			})

			DescribeTable("should forbid issuerURL to be empty string or nil, if clientID exists ", func(errorListSize int, issuerURL *string) {
				shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.ClientID = pointer.String("someClientID")
				shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.IssuerURL = issuerURL

				errorList := ValidateShoot(shoot)
				Expect(errorList).To(HaveLen(errorListSize))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeAPIServer.oidcConfig.issuerURL"),
				}))
			},
				Entry("should add error if clientID is set but issuerURL is nil ", 1, nil),
				Entry("should add error if clientID is set but issuerURL is empty string", 2, pointer.String("")),
			)

			It("should forbid issuerURL which is not HTTPS schema", func() {
				shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.IssuerURL = pointer.String("http://issuer.com")
				shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.ClientID = pointer.String("someClientID")

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(HaveLen(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeAPIServer.oidcConfig.issuerURL"),
				}))
			})

			It("should not fail if both clientID and issuerURL are set", func() {
				shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.IssuerURL = pointer.String("https://issuer.com")
				shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.ClientID = pointer.String("someClientID")

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(BeEmpty())
			})

			DescribeTable("should forbid clientID to be empty string or nil, if issuerURL exists ", func(errorListSize int, clientID *string) {
				shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.IssuerURL = pointer.String("https://issuer.com")
				shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.ClientID = clientID

				errorList := ValidateShoot(shoot)
				Expect(errorList).To(HaveLen(errorListSize))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeAPIServer.oidcConfig.clientID"),
				}))
			},
				Entry("should add error if issuerURL is set but clientID is nil", 1, nil),
				Entry("should add error if issuerURL is set but clientID is empty string ", 2, pointer.String("")),
			)
		})

		Context("basic authentication", func() {
			It("should allow basic authentication when kubernetes <= 1.18", func() {
				shoot.Spec.Kubernetes.Version = "1.18.1"
				shoot.Spec.Kubernetes.KubeAPIServer.EnableBasicAuthentication = pointer.Bool(true)

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(HaveLen(0))
			})

			It("should forbid basic authentication when kubernetes >= 1.19", func() {
				shoot.Spec.Kubernetes.Version = "1.19.1"
				shoot.Spec.Kubernetes.KubeAPIServer.EnableBasicAuthentication = pointer.Bool(true)

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("spec.kubernetes.kubeAPIServer.enableBasicAuthentication"),
				}))))
			})

			It("should allow disabling basic authentication when kubernetes >= 1.19", func() {
				shoot.Spec.Kubernetes.Version = "1.19.1"
				shoot.Spec.Kubernetes.KubeAPIServer.EnableBasicAuthentication = pointer.Bool(false)

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(HaveLen(0))
			})
		})

		Context("admission plugin validation", func() {
			It("should allow not specifying admission plugins", func() {
				shoot.Spec.Kubernetes.KubeAPIServer.AdmissionPlugins = nil

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(HaveLen(0))
			})

			It("should forbid specifying admission plugins without a name", func() {
				shoot.Spec.Kubernetes.KubeAPIServer.AdmissionPlugins = []core.AdmissionPlugin{
					{
						Name: "",
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(HaveLen(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.kubernetes.kubeAPIServer.admissionPlugins[0].name"),
				}))
			})

			It("should forbid specifying the SecurityContextDeny admission plugin", func() {
				shoot.Spec.Kubernetes.KubeAPIServer.AdmissionPlugins = []core.AdmissionPlugin{
					{
						Name: "SecurityContextDeny",
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("spec.kubernetes.kubeAPIServer.admissionPlugins[0].name"),
				}))))
			})
		})

		Context("WatchCacheSizes validation", func() {
			var negativeSize int32 = -1

			DescribeTable("watch cache size validation",
				func(sizes *core.WatchCacheSizes, matcher gomegatypes.GomegaMatcher) {
					Expect(ValidateWatchCacheSizes(sizes, nil)).To(matcher)
				},

				Entry("valid (unset)", nil, BeEmpty()),
				Entry("valid (fields unset)", &core.WatchCacheSizes{}, BeEmpty()),
				Entry("valid (default=0)", &core.WatchCacheSizes{
					Default: pointer.Int32(0),
				}, BeEmpty()),
				Entry("valid (default>0)", &core.WatchCacheSizes{
					Default: pointer.Int32(42),
				}, BeEmpty()),
				Entry("invalid (default<0)", &core.WatchCacheSizes{
					Default: pointer.Int32(negativeSize),
				}, ConsistOf(
					field.Invalid(field.NewPath("default"), int64(negativeSize), apivalidation.IsNegativeErrorMsg),
				)),

				// APIGroup unset (core group)
				Entry("valid (core/secrets=0)", &core.WatchCacheSizes{
					Resources: []core.ResourceWatchCacheSize{{
						Resource:  "secrets",
						CacheSize: 0,
					}},
				}, BeEmpty()),
				Entry("valid (core/secrets=>0)", &core.WatchCacheSizes{
					Resources: []core.ResourceWatchCacheSize{{
						Resource:  "secrets",
						CacheSize: 42,
					}},
				}, BeEmpty()),
				Entry("invalid (core/secrets=<0)", &core.WatchCacheSizes{
					Resources: []core.ResourceWatchCacheSize{{
						Resource:  "secrets",
						CacheSize: negativeSize,
					}},
				}, ConsistOf(
					field.Invalid(field.NewPath("resources[0].size"), int64(negativeSize), apivalidation.IsNegativeErrorMsg),
				)),
				Entry("invalid (core/resource empty)", &core.WatchCacheSizes{
					Resources: []core.ResourceWatchCacheSize{{
						Resource:  "",
						CacheSize: 42,
					}},
				}, ConsistOf(
					field.Required(field.NewPath("resources[0].resource"), "must not be empty"),
				)),

				// APIGroup set
				Entry("valid (apps/deployments=0)", &core.WatchCacheSizes{
					Resources: []core.ResourceWatchCacheSize{{
						APIGroup:  pointer.String("apps"),
						Resource:  "deployments",
						CacheSize: 0,
					}},
				}, BeEmpty()),
				Entry("valid (apps/deployments=>0)", &core.WatchCacheSizes{
					Resources: []core.ResourceWatchCacheSize{{
						APIGroup:  pointer.String("apps"),
						Resource:  "deployments",
						CacheSize: 42,
					}},
				}, BeEmpty()),
				Entry("invalid (apps/deployments=<0)", &core.WatchCacheSizes{
					Resources: []core.ResourceWatchCacheSize{{
						APIGroup:  pointer.String("apps"),
						Resource:  "deployments",
						CacheSize: negativeSize,
					}},
				}, ConsistOf(
					field.Invalid(field.NewPath("resources[0].size"), int64(negativeSize), apivalidation.IsNegativeErrorMsg),
				)),
				Entry("invalid (apps/resource empty)", &core.WatchCacheSizes{
					Resources: []core.ResourceWatchCacheSize{{
						Resource:  "",
						CacheSize: 42,
					}},
				}, ConsistOf(
					field.Required(field.NewPath("resources[0].resource"), "must not be empty"),
				)),
			)
		})

		Context("requests", func() {
			It("should not allow too high values for max inflight requests fields", func() {
				shoot.Spec.Kubernetes.KubeAPIServer.Requests = &core.KubeAPIServerRequests{
					MaxNonMutatingInflight: pointer.Int32(123123123),
					MaxMutatingInflight:    pointer.Int32(412412412),
				}

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeAPIServer.requests.maxNonMutatingInflight"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeAPIServer.requests.maxMutatingInflight"),
				}))))
			})

			It("should not allow negative values for max inflight requests fields", func() {
				shoot.Spec.Kubernetes.KubeAPIServer.Requests = &core.KubeAPIServerRequests{
					MaxNonMutatingInflight: pointer.Int32(-1),
					MaxMutatingInflight:    pointer.Int32(-1),
				}

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeAPIServer.requests.maxNonMutatingInflight"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeAPIServer.requests.maxMutatingInflight"),
				}))))
			})
		})

		Context("service account config", func() {
			It("should not allow too specify a negative max token duration", func() {
				shoot.Spec.Kubernetes.KubeAPIServer.ServiceAccountConfig = &core.ServiceAccountConfig{
					MaxTokenExpiration: &metav1.Duration{Duration: -1},
				}

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeAPIServer.serviceAccountConfig.maxTokenExpiration"),
				}))))
			})

			It("should not allow too specify the 'extend' flag if kubernetes is lower than 1.19", func() {
				shoot.Spec.Kubernetes.Version = "1.18.9"
				shoot.Spec.Kubernetes.KubeAPIServer.ServiceAccountConfig = &core.ServiceAccountConfig{
					ExtendTokenExpiration: pointer.Bool(true),
				}

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("spec.kubernetes.kubeAPIServer.serviceAccountConfig.extendTokenExpiration"),
				}))))
			})

			It("should not allow too specify multiple issuers if kubernetes is lower than 1.22", func() {
				shoot.Spec.Kubernetes.Version = "1.21.9"
				shoot.Spec.Kubernetes.KubeAPIServer.ServiceAccountConfig = &core.ServiceAccountConfig{
					Issuer:          pointer.String("issuer"),
					AcceptedIssuers: []string{"issuer1"},
				}

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("spec.kubernetes.kubeAPIServer.serviceAccountConfig.acceptedIssuers"),
				}))))
			})
		})

		It("should not allow too specify a negative event ttl duration", func() {
			shoot.Spec.Kubernetes.KubeAPIServer.EventTTL = &metav1.Duration{Duration: -1}

			errorList := ValidateShoot(shoot)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.kubernetes.kubeAPIServer.eventTTL"),
			}))))
		})

		It("should not allow too specify an event ttl duration longer than 7d", func() {
			shoot.Spec.Kubernetes.KubeAPIServer.EventTTL = &metav1.Duration{Duration: time.Hour * 24 * 8}

			errorList := ValidateShoot(shoot)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.kubernetes.kubeAPIServer.eventTTL"),
			}))))
		})

		Context("KubeControllerManager validation", func() {
			It("should forbid unsupported HPA configuration", func() {
				shoot.Spec.Kubernetes.KubeControllerManager.HorizontalPodAutoscalerConfig.DownscaleStabilization = makeDurationPointer(-1 * time.Second)
				shoot.Spec.Kubernetes.KubeControllerManager.HorizontalPodAutoscalerConfig.InitialReadinessDelay = makeDurationPointer(-1 * time.Second)
				shoot.Spec.Kubernetes.KubeControllerManager.HorizontalPodAutoscalerConfig.CPUInitializationPeriod = makeDurationPointer(-1 * time.Second)

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeControllerManager.horizontalPodAutoscaler.downscaleStabilization"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeControllerManager.horizontalPodAutoscaler.initialReadinessDelay"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeControllerManager.horizontalPodAutoscaler.cpuInitializationPeriod"),
				}))))
			})

			It("should succeed when using valid configuration parameters", func() {
				shoot.Spec.Kubernetes.KubeControllerManager.HorizontalPodAutoscalerConfig.DownscaleStabilization = makeDurationPointer(5 * time.Minute)
				shoot.Spec.Kubernetes.KubeControllerManager.HorizontalPodAutoscalerConfig.InitialReadinessDelay = makeDurationPointer(30 * time.Second)
				shoot.Spec.Kubernetes.KubeControllerManager.HorizontalPodAutoscalerConfig.CPUInitializationPeriod = makeDurationPointer(5 * time.Minute)

				errorList := ValidateShoot(shoot)
				Expect(errorList).To(HaveLen(0))
			})

			It("should fail updating immutable fields", func() {
				shoot.Spec.Kubernetes.KubeControllerManager.NodeCIDRMaskSize = pointer.Int32(24)

				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.Kubernetes.KubeControllerManager.NodeCIDRMaskSize = pointer.Int32(22)

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("spec.kubernetes.kubeControllerManager.nodeCIDRMaskSize"),
					"Detail": ContainSubstring(`field is immutable`),
				}))
			})

			It("should succeed not changing immutable fields", func() {
				shoot.Spec.Kubernetes.KubeControllerManager.NodeCIDRMaskSize = pointer.Int32(24)

				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.Kubernetes.KubeControllerManager.NodeCIDRMaskSize = pointer.Int32(24)

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(errorList).To(BeEmpty())
			})

			It("should fail when nodeCIDRMaskSize is out of upper boundary", func() {
				shoot.Spec.Kubernetes.KubeControllerManager.NodeCIDRMaskSize = pointer.Int32(32)

				errorList := ValidateShoot(shoot)
				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeControllerManager.nodeCIDRMaskSize"),
				}))))
			})

			It("should fail when nodeCIDRMaskSize is out of lower boundary", func() {
				shoot.Spec.Kubernetes.KubeControllerManager.NodeCIDRMaskSize = pointer.Int32(0)

				errorList := ValidateShoot(shoot)
				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeControllerManager.nodeCIDRMaskSize"),
				}))))
			})

			It("should succeed when nodeCIDRMaskSize is within boundaries", func() {
				shoot.Spec.Kubernetes.KubeControllerManager.NodeCIDRMaskSize = pointer.Int32(22)

				errorList := ValidateShoot(shoot)
				Expect(errorList).To(BeEmpty())
			})

			It("should prevent setting a negative pod eviction timeout", func() {
				shoot.Spec.Kubernetes.KubeControllerManager.PodEvictionTimeout = &metav1.Duration{Duration: -1}

				errorList := ValidateShoot(shoot)
				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeControllerManager.podEvictionTimeout"),
				}))))
			})

			It("should prevent setting the pod eviction timeout to 0", func() {
				shoot.Spec.Kubernetes.KubeControllerManager.PodEvictionTimeout = &metav1.Duration{}

				errorList := ValidateShoot(shoot)
				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeControllerManager.podEvictionTimeout"),
				}))))
			})

			It("should prevent setting a negative node monitor grace period", func() {
				shoot.Spec.Kubernetes.KubeControllerManager.NodeMonitorGracePeriod = &metav1.Duration{Duration: -1}

				errorList := ValidateShoot(shoot)
				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeControllerManager.nodeMonitorGracePeriod"),
				}))))
			})

			It("should prevent setting the node monitor grace period to 0", func() {
				shoot.Spec.Kubernetes.KubeControllerManager.NodeMonitorGracePeriod = &metav1.Duration{}

				errorList := ValidateShoot(shoot)
				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.kubeControllerManager.nodeMonitorGracePeriod"),
				}))))
			})
		})

		Context("KubeProxy validation", func() {
			BeforeEach(func() {
				shoot.Spec.Kubernetes.KubeProxy = &core.KubeProxyConfig{}
			})

			It("should succeed when using IPTables mode", func() {
				mode := core.ProxyModeIPTables
				shoot.Spec.Kubernetes.KubeProxy.Mode = &mode
				errorList := ValidateShoot(shoot)

				Expect(errorList).To(BeEmpty())
			})

			It("should succeed when using IPVS mode", func() {
				mode := core.ProxyModeIPVS
				shoot.Spec.Kubernetes.KubeProxy.Mode = &mode
				errorList := ValidateShoot(shoot)

				Expect(errorList).To(BeEmpty())
			})

			It("should fail when using nil proxy mode", func() {
				shoot.Spec.Kubernetes.KubeProxy.Mode = nil
				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.kubernetes.kubeProxy.mode"),
				}))))
			})

			It("should fail when using unknown proxy mode", func() {
				m := core.ProxyMode("fooMode")
				shoot.Spec.Kubernetes.KubeProxy.Mode = &m
				errorList := ValidateShoot(shoot)
				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeNotSupported),
					"Field": Equal("spec.kubernetes.kubeProxy.mode"),
				}))))
			})

			It("should be successful when proxy mode is changed", func() {
				mode := core.ProxyMode("IPVS")
				kubernetesConfig := core.KubernetesConfig{}
				config := core.KubeProxyConfig{
					KubernetesConfig: kubernetesConfig,
					Mode:             &mode,
				}
				shoot.Spec.Kubernetes.KubeProxy = &config
				shoot.Spec.Kubernetes.Version = "1.20.1"
				oldMode := core.ProxyMode("IPTables")
				oldConfig := core.KubeProxyConfig{
					KubernetesConfig: kubernetesConfig,
					Mode:             &oldMode,
				}
				shoot.Spec.Kubernetes.KubeProxy.Mode = &mode
				oldShoot := shoot.DeepCopy()
				oldShoot.Spec.Kubernetes.KubeProxy = &oldConfig

				errorList := ValidateShootSpecUpdate(&shoot.Spec, &oldShoot.Spec, metav1.ObjectMeta{}, field.NewPath("spec"))
				Expect(errorList).To(BeEmpty())
			})

			It("should not fail when kube-proxy is switched off", func() {
				kubernetesConfig := core.KubernetesConfig{}
				disabled := false
				config := core.KubeProxyConfig{
					KubernetesConfig: kubernetesConfig,
					Enabled:          &disabled,
				}
				shoot.Spec.Kubernetes.KubeProxy = &config
				enabled := true
				oldConfig := core.KubeProxyConfig{
					KubernetesConfig: kubernetesConfig,
					Enabled:          &enabled,
				}
				oldShoot := shoot.DeepCopy()
				oldShoot.Spec.Kubernetes.KubeProxy = &oldConfig

				errorList := ValidateShootSpecUpdate(&shoot.Spec, &oldShoot.Spec, metav1.ObjectMeta{}, field.NewPath("spec"))

				Expect(errorList).To(BeEmpty())
			})
		})

		var (
			negativeDuration            = metav1.Duration{Duration: -time.Second}
			negativeInteger       int32 = -100
			positiveInteger       int32 = 100
			expanderLeastWaste          = core.ClusterAutoscalerExpanderLeastWaste
			expanderMostPods            = core.ClusterAutoscalerExpanderMostPods
			expanderPriority            = core.ClusterAutoscalerExpanderPriority
			expanderRandom              = core.ClusterAutoscalerExpanderRandom
			ignoreTaintsUnique          = []string{"taint-1", "taint-2"}
			ignoreTaintsDuplicate       = []string{"taint-1", "taint-1"}
			ignoreTaintsInvalid         = []string{"taint 1", "taint-1"}
			version                     = "1.20"
		)

		Context("ClusterAutoscaler validation", func() {
			DescribeTable("cluster autoscaler values",
				func(clusterAutoscaler core.ClusterAutoscaler, supportedVersionForIgnoreTaints string, matcher gomegatypes.GomegaMatcher) {
					Expect(ValidateClusterAutoscaler(clusterAutoscaler, supportedVersionForIgnoreTaints, nil)).To(matcher)
				},
				Entry("valid", core.ClusterAutoscaler{}, version, BeEmpty()),
				Entry("valid with threshold", core.ClusterAutoscaler{
					ScaleDownUtilizationThreshold: pointer.Float64(0.5),
				}, version, BeEmpty()),
				Entry("invalid negative threshold", core.ClusterAutoscaler{
					ScaleDownUtilizationThreshold: pointer.Float64(-0.5),
				}, version, ConsistOf(field.Invalid(field.NewPath("scaleDownUtilizationThreshold"), -0.5, "can not be negative"))),
				Entry("invalid > 1 threshold", core.ClusterAutoscaler{
					ScaleDownUtilizationThreshold: pointer.Float64(1.5),
				}, version, ConsistOf(field.Invalid(field.NewPath("scaleDownUtilizationThreshold"), 1.5, "can not be greater than 1.0"))),
				Entry("valid with maxNodeProvisionTime", core.ClusterAutoscaler{
					MaxNodeProvisionTime: &metav1.Duration{Duration: time.Minute},
				}, version, BeEmpty()),
				Entry("invalid with negative maxNodeProvisionTime", core.ClusterAutoscaler{
					MaxNodeProvisionTime: &negativeDuration,
				}, version, ConsistOf(field.Invalid(field.NewPath("maxNodeProvisionTime"), negativeDuration, "can not be negative"))),
				Entry("valid with maxGracefulTerminationSeconds", core.ClusterAutoscaler{
					MaxGracefulTerminationSeconds: &positiveInteger,
				}, version, BeEmpty()),
				Entry("invalid with negative maxGracefulTerminationSeconds", core.ClusterAutoscaler{
					MaxGracefulTerminationSeconds: &negativeInteger,
				}, version, ConsistOf(field.Invalid(field.NewPath("maxGracefulTerminationSeconds"), negativeInteger, "can not be negative"))),
				Entry("valid with expander least waste", core.ClusterAutoscaler{
					Expander: &expanderLeastWaste,
				}, version, BeEmpty()),
				Entry("valid with expander most pods", core.ClusterAutoscaler{
					Expander: &expanderMostPods,
				}, version, BeEmpty()),
				Entry("valid with expander priority", core.ClusterAutoscaler{
					Expander: &expanderPriority,
				}, version, BeEmpty()),
				Entry("valid with expander random", core.ClusterAutoscaler{
					Expander: &expanderRandom,
				}, version, BeEmpty()),
				Entry("valid with ignore taint", core.ClusterAutoscaler{
					IgnoreTaints: ignoreTaintsUnique,
				}, version, BeEmpty()),
				Entry("duplicate ignore taint", core.ClusterAutoscaler{
					IgnoreTaints: ignoreTaintsDuplicate,
				}, version, ConsistOf(field.Duplicate(field.NewPath("ignoreTaints").Index(1), ignoreTaintsDuplicate[1]))),
				Entry("invalid with ignore taint",
					core.ClusterAutoscaler{
						IgnoreTaints: ignoreTaintsInvalid,
					},
					version,
					ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("ignoreTaints[0]"),
					}))),
				),
			)
		})

		Context("VerticalPodAutoscaler validation", func() {
			DescribeTable("verticalPod autoscaler values",
				func(verticalPodAutoscaler core.VerticalPodAutoscaler, matcher gomegatypes.GomegaMatcher) {
					Expect(ValidateVerticalPodAutoscaler(verticalPodAutoscaler, nil)).To(matcher)
				},
				Entry("valid", core.VerticalPodAutoscaler{}, BeEmpty()),
				Entry("invalid negative durations", core.VerticalPodAutoscaler{
					EvictAfterOOMThreshold: &negativeDuration,
					UpdaterInterval:        &negativeDuration,
					RecommenderInterval:    &negativeDuration,
				}, ConsistOf(
					field.Invalid(field.NewPath("evictAfterOOMThreshold"), negativeDuration, "can not be negative"),
					field.Invalid(field.NewPath("updaterInterval"), negativeDuration, "can not be negative"),
					field.Invalid(field.NewPath("recommenderInterval"), negativeDuration, "can not be negative"),
				)),
			)
		})

		Context("AuditConfig validation", func() {
			It("should forbid empty name", func() {
				shoot.Spec.Kubernetes.KubeAPIServer.AuditConfig.AuditPolicy.ConfigMapRef.Name = ""
				errorList := ValidateShoot(shoot)

				Expect(errorList).ToNot(BeEmpty())
				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.kubernetes.kubeAPIServer.auditConfig.auditPolicy.configMapRef.name"),
				}))))
			})

			It("should allow nil AuditConfig", func() {
				shoot.Spec.Kubernetes.KubeAPIServer.AuditConfig = nil
				errorList := ValidateShoot(shoot)

				Expect(errorList).To(BeEmpty())
			})
		})

		Context("FeatureGates validation", func() {
			It("should forbid invalid feature gates", func() {
				featureGates := map[string]bool{
					"AnyVolumeDataSource":      true,
					"CustomResourceValidation": true,
					"Foo":                      true,
				}
				shoot.Spec.Kubernetes.Version = "1.18.14"
				shoot.Spec.Kubernetes.KubeAPIServer.FeatureGates = featureGates
				shoot.Spec.Kubernetes.KubeControllerManager.FeatureGates = featureGates
				shoot.Spec.Kubernetes.KubeScheduler = &core.KubeSchedulerConfig{
					KubernetesConfig: core.KubernetesConfig{
						FeatureGates: featureGates,
					},
				}
				proxyMode := core.ProxyModeIPTables
				shoot.Spec.Kubernetes.KubeProxy = &core.KubeProxyConfig{
					KubernetesConfig: core.KubernetesConfig{
						FeatureGates: featureGates,
					},
					Mode: &proxyMode,
				}
				shoot.Spec.Kubernetes.Kubelet = &core.KubeletConfig{
					KubernetesConfig: core.KubernetesConfig{
						FeatureGates: featureGates,
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(errorList).ToNot(BeEmpty())
				Expect(errorList).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeForbidden),
						"Field": Equal("spec.kubernetes.kubeAPIServer.featureGates.CustomResourceValidation"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.kubernetes.kubeAPIServer.featureGates.Foo"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeForbidden),
						"Field": Equal("spec.kubernetes.kubeControllerManager.featureGates.CustomResourceValidation"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.kubernetes.kubeControllerManager.featureGates.Foo"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeForbidden),
						"Field": Equal("spec.kubernetes.kubeScheduler.featureGates.CustomResourceValidation"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.kubernetes.kubeScheduler.featureGates.Foo"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeForbidden),
						"Field": Equal("spec.kubernetes.kubeProxy.featureGates.CustomResourceValidation"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.kubernetes.kubeProxy.featureGates.Foo"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeForbidden),
						"Field": Equal("spec.kubernetes.kubelet.featureGates.CustomResourceValidation"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.kubernetes.kubelet.featureGates.Foo"),
					})),
				))
			})
		})

		It("should require a kubernetes version", func() {
			shoot.Spec.Kubernetes.Version = ""

			errorList := ValidateShoot(shoot)

			Expect(errorList).To(HaveLen(1))
			Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("spec.kubernetes.version"),
			}))
		})

		It("should forbid removing the kubernetes version", func() {
			newShoot := prepareShootForUpdate(shoot)
			newShoot.Spec.Kubernetes.Version = ""

			Expect(ValidateShootUpdate(newShoot, shoot)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("spec.kubernetes.version"),
					"Detail": Equal("cannot validate kubernetes version upgrade because it is unset"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("spec.provider.workers[0].kubernetes.version"),
					"Detail": Equal("cannot validate kubernetes version upgrade because it is unset"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("spec.kubernetes.version"),
					"Detail": Equal("kubernetes version must not be empty"),
				})),
			))
		})

		It("should forbid kubernetes version downgrades", func() {
			newShoot := prepareShootForUpdate(shoot)
			newShoot.Spec.Kubernetes.Version = "1.7.2"

			Expect(ValidateShootUpdate(newShoot, shoot)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeForbidden),
					"Field":  Equal("spec.kubernetes.version"),
					"Detail": Equal("kubernetes version downgrade is not supported"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeForbidden),
					"Field":  Equal("spec.provider.workers[0].kubernetes.version"),
					"Detail": Equal("kubernetes version downgrade is not supported"),
				})),
			))
		})

		It("should forbid kubernetes version upgrades skipping a minor version", func() {
			newShoot := prepareShootForUpdate(shoot)
			newShoot.Spec.Kubernetes.Version = "1.22.1"

			Expect(ValidateShootUpdate(newShoot, shoot)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeForbidden),
					"Field":  Equal("spec.kubernetes.version"),
					"Detail": Equal("kubernetes version upgrade cannot skip a minor version"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeForbidden),
					"Field":  Equal("spec.provider.workers[0].kubernetes.version"),
					"Detail": Equal("kubernetes version upgrade cannot skip a minor version"),
				})),
			))
		})

		Context("worker pool kubernetes version", func() {
			It("should forbid specifying a worker pool kubernetes version since the WorkerPoolKubernetesVersion feature gate is disabled", func() {
				defer test.WithFeatureGate(utilfeature.DefaultFeatureGate, features.WorkerPoolKubernetesVersion, false)()

				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.Provider.Workers[0].Kubernetes = &core.WorkerKubernetes{Version: &shoot.Spec.Kubernetes.Version}

				Expect(ValidateShootUpdate(newShoot, shoot)).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeForbidden),
					"Field":  Equal("spec.provider.workers[0].kubernetes.version"),
					"Detail": Equal("worker pool kubernetes version may only be set if WorkerPoolKubernetesVersion feature gate is enabled"),
				}))))
			})

			It("should forbid worker pool kubernetes version higher than control plane", func() {
				defer test.WithFeatureGate(utilfeature.DefaultFeatureGate, features.WorkerPoolKubernetesVersion, true)()

				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.Provider.Workers[0].Kubernetes = &core.WorkerKubernetes{Version: pointer.String("1.21.0")}

				Expect(ValidateShootUpdate(newShoot, shoot)).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeForbidden),
					"Field":  Equal("spec.provider.workers[0].kubernetes.version"),
					"Detail": Equal("worker group kubernetes version must not be higher than control plane version"),
				}))))
			})

			It("should work to set worker pool kubernetes version equal to control plane version", func() {
				defer test.WithFeatureGate(utilfeature.DefaultFeatureGate, features.WorkerPoolKubernetesVersion, true)()

				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.Provider.Workers[0].Kubernetes = &core.WorkerKubernetes{Version: pointer.String("1.20.2")}

				Expect(ValidateShootUpdate(newShoot, shoot)).To(BeEmpty())
			})

			It("should work to set worker pool kubernetes version lower one minor than control plane version", func() {
				defer test.WithFeatureGate(utilfeature.DefaultFeatureGate, features.WorkerPoolKubernetesVersion, true)()

				shoot.Spec.Provider.Workers[0].Kubernetes = &core.WorkerKubernetes{Version: pointer.String("1.20.2")}

				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.Kubernetes.Version = "1.21.0"

				Expect(ValidateShootUpdate(newShoot, shoot)).To(BeEmpty())
			})

			It("should work to set worker pool kubernetes version lower two minor than control plane version", func() {
				defer test.WithFeatureGate(utilfeature.DefaultFeatureGate, features.WorkerPoolKubernetesVersion, true)()

				shoot.Spec.Kubernetes.Version = "1.21.0"
				shoot.Spec.Provider.Workers[0].Kubernetes = &core.WorkerKubernetes{Version: pointer.String("1.20.2")}

				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.Kubernetes.Version = "1.22.0"

				Expect(ValidateShootUpdate(newShoot, shoot)).To(BeEmpty())
			})

			It("forbid to set worker pool kubernetes version lower three minor than control plane version", func() {
				defer test.WithFeatureGate(utilfeature.DefaultFeatureGate, features.WorkerPoolKubernetesVersion, true)()

				shoot.Spec.Kubernetes.Version = "1.22.0"
				shoot.Spec.Provider.Workers[0].Kubernetes = &core.WorkerKubernetes{Version: pointer.String("1.20.2")}

				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.Kubernetes.Version = "1.23.0"

				Expect(ValidateShootUpdate(newShoot, shoot)).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeForbidden),
					"Field":  Equal("spec.provider.workers[0].kubernetes.version"),
					"Detail": Equal("worker group kubernetes version must be at most two minor versions behind control plane version"),
				}))))
			})

			It("should work to set worker pool kubernetes version to nil with one minor difference", func() {
				defer test.WithFeatureGate(utilfeature.DefaultFeatureGate, features.WorkerPoolKubernetesVersion, true)()

				shoot.Spec.Kubernetes.Version = "1.21.0"
				shoot.Spec.Provider.Workers[0].Kubernetes = &core.WorkerKubernetes{Version: pointer.String("1.20.2")}

				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.Provider.Workers[0].Kubernetes = &core.WorkerKubernetes{Version: nil}

				Expect(ValidateShootUpdate(newShoot, shoot)).To(BeEmpty())
			})

			It("forbid to set worker pool kubernetes version to nil with two minor difference", func() {
				defer test.WithFeatureGate(utilfeature.DefaultFeatureGate, features.WorkerPoolKubernetesVersion, true)()

				shoot.Spec.Kubernetes.Version = "1.21.0"
				shoot.Spec.Provider.Workers[0].Kubernetes = &core.WorkerKubernetes{Version: pointer.String("1.19.2")}

				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.Provider.Workers[0].Kubernetes = &core.WorkerKubernetes{Version: nil}

				Expect(ValidateShootUpdate(newShoot, shoot)).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeForbidden),
					"Field":  Equal("spec.provider.workers[0].kubernetes.version"),
					"Detail": Equal("kubernetes version upgrade cannot skip a minor version"),
				}))))
			})
		})

		Context("networking section", func() {
			It("should forbid not specifying a networking type", func() {
				shoot.Spec.Networking.Type = ""

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.networking.type"),
				}))))
			})

			It("should forbid changing the networking type", func() {
				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.Networking.Type = "some-other-type"

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.networking.type"),
				}))))
			})
		})

		Context("maintenance section", func() {
			It("should forbid invalid formats for the time window begin and end values", func() {
				shoot.Spec.Maintenance.TimeWindow.Begin = "invalidformat"
				shoot.Spec.Maintenance.TimeWindow.End = "invalidformat"

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.maintenance.timeWindow.begin/end"),
				}))))
			})

			It("should forbid time windows greater than 6 hours", func() {
				shoot.Spec.Maintenance.TimeWindow.Begin = "145000+0100"
				shoot.Spec.Maintenance.TimeWindow.End = "215000+0100"

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("spec.maintenance.timeWindow"),
				}))))
			})

			It("should forbid time windows smaller than 30 minutes", func() {
				shoot.Spec.Maintenance.TimeWindow.Begin = "225000+0100"
				shoot.Spec.Maintenance.TimeWindow.End = "231000+0100"

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("spec.maintenance.timeWindow"),
				}))))
			})

			It("should allow time windows which overlap over two days", func() {
				shoot.Spec.Maintenance.TimeWindow.Begin = "230000+0100"
				shoot.Spec.Maintenance.TimeWindow.End = "010000+0100"

				errorList := ValidateShoot(shoot)

				Expect(errorList).To(HaveLen(0))
			})
		})

		It("should forbid updating the spec for shoots with deletion timestamp", func() {
			newShoot := prepareShootForUpdate(shoot)
			deletionTimestamp := metav1.NewTime(time.Now())
			shoot.DeletionTimestamp = &deletionTimestamp
			newShoot.DeletionTimestamp = &deletionTimestamp
			newShoot.Spec.Maintenance.AutoUpdate.KubernetesVersion = false

			errorList := ValidateShootUpdate(newShoot, shoot)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec"),
			}))))
		})

		It("should allow updating the metadata for shoots with deletion timestamp", func() {
			newShoot := prepareShootForUpdate(shoot)
			deletionTimestamp := metav1.NewTime(time.Now())
			shoot.DeletionTimestamp = &deletionTimestamp
			newShoot.DeletionTimestamp = &deletionTimestamp
			newShoot.Labels = map[string]string{
				"new-key": "new-value",
			}

			errorList := ValidateShootUpdate(newShoot, shoot)

			Expect(errorList).To(HaveLen(0))
		})

		Describe("kubeconfig rotation", func() {
			DescribeTable("DisallowKubeconfigRotationForShootInDeletion",
				func(oldAnnotations, newAnnotations map[string]string, newSetDeletionTimestamp, expectedError bool) {
					now := metav1.NewTime(time.Now())
					newShoot := prepareShootForUpdate(shoot)
					if oldAnnotations != nil {
						shoot.Annotations = oldAnnotations
					}

					if newSetDeletionTimestamp {
						newShoot.DeletionTimestamp = &now
					}
					newShoot.Annotations = newAnnotations

					errorList := ValidateShootObjectMetaUpdate(newShoot.ObjectMeta, shoot.ObjectMeta, field.NewPath("metadata"))

					if expectedError {
						Expect(errorList).ToNot(HaveLen(0))
						Expect(errorList).To(ConsistOfFields(Fields{
							"Type":   Equal(field.ErrorTypeInvalid),
							"Field":  Equal("metadata.annotations[gardener.cloud/operation]"),
							"Detail": ContainSubstring(`kubeconfig rotations is not allowed for clusters in deletion`),
						}))
					} else {
						Expect(errorList).To(HaveLen(0))
					}
				},
				Entry("should allow kubeconfig rotation for cluster not in deletion", nil, map[string]string{"gardener.cloud/operation": "rotate-kubeconfig-credentials"}, false, false),
				Entry("should allow reconcile operation for cluster in deletion", nil, map[string]string{"gardener.cloud/operation": "reconcile"}, true, false),
				Entry("should allow any annotations for cluster in deletion", nil, map[string]string{"foo": "bar"}, true, false),
				Entry("should allow other update request for cluster in deletion and already requested kubeconfig rotation operation", map[string]string{"gardener.cloud/operation": "rotate-kubeconfig-credentials"}, map[string]string{"gardener.cloud/operation": "reconcile"}, true, false),
				Entry("should allow any annotations for cluster in deletion with already requested kubeconfig rotation", map[string]string{"gardener.cloud/operation": "rotate-kubeconfig-credentials"}, map[string]string{"foo": "bar"}, true, false),
				Entry("should allow update request for cluster in deletion with already requested kubeconfig rotation", map[string]string{"gardener.cloud/operation": "rotate-kubeconfig-credentials"}, map[string]string{"gardener.cloud/operation": "rotate-kubeconfig-credentials", "foo": "bar"}, true, false),
				Entry("should not allow kubeconfig rotation for cluster in deletion", nil, map[string]string{"gardener.cloud/operation": "rotate-kubeconfig-credentials"}, true, true),
				Entry("should not allow kubeconfig rotation for cluster in deletion with already requested operation", map[string]string{"gardener.cloud/operation": "some-other-operation"}, map[string]string{"gardener.cloud/operation": "rotate-kubeconfig-credentials"}, true, true),
			)
		})

		Describe("#ValidateSystemComponents", func() {
			DescribeTable("validate system components",
				func(systemComponents *core.SystemComponents, matcher gomegatypes.GomegaMatcher) {
					Expect(ValidateSystemComponents(systemComponents, nil)).To(matcher)
				},
				Entry("no system components", nil, BeEmpty()),
				Entry("empty system components", &core.SystemComponents{}, BeEmpty()),
				Entry("empty core dns", &core.SystemComponents{CoreDNS: &core.CoreDNS{}}, BeEmpty()),
				Entry("horizontal core dns autoscaler", &core.SystemComponents{CoreDNS: &core.CoreDNS{Autoscaling: &core.CoreDNSAutoscaling{Mode: core.CoreDNSAutoscalingModeHorizontal}}}, BeEmpty()),
				Entry("cluster proportional core dns autoscaler", &core.SystemComponents{CoreDNS: &core.CoreDNS{Autoscaling: &core.CoreDNSAutoscaling{Mode: core.CoreDNSAutoscalingModeHorizontal}}}, BeEmpty()),
				Entry("incorrect core dns autoscaler", &core.SystemComponents{CoreDNS: &core.CoreDNS{Autoscaling: &core.CoreDNSAutoscaling{Mode: "dummy"}}}, ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(field.ErrorTypeNotSupported),
				})))),
			)
		})

		Context("operation validation", func() {
			It("should do nothing if the operation annotation is not set", func() {
				defer test.WithFeatureGate(utilfeature.DefaultFeatureGate, features.ShootCARotation, true)()

				Expect(ValidateShoot(shoot)).To(BeEmpty())
			})

			It("should do nothing if the feature gate is disabled", func() {
				defer test.WithFeatureGate(utilfeature.DefaultFeatureGate, features.ShootCARotation, false)()

				metav1.SetMetaDataAnnotation(&shoot.ObjectMeta, "gardener.cloud/operation", "rotate-ca-start")

				Expect(ValidateShoot(shoot)).To(BeEmpty())
			})

			DescribeTable("starting CA rotation",
				func(allowed bool, status core.ShootStatus) {
					defer test.WithFeatureGate(utilfeature.DefaultFeatureGate, features.ShootCARotation, true)()
					metav1.SetMetaDataAnnotation(&shoot.ObjectMeta, "gardener.cloud/operation", "rotate-ca-start")
					shoot.Status = status

					matcher := BeEmpty()
					if !allowed {
						matcher = ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
							"Type":  Equal(field.ErrorTypeForbidden),
							"Field": Equal("metadata.annotations[gardener.cloud/operation]"),
						})))
					}

					Expect(ValidateShoot(shoot)).To(matcher)
				},

				Entry("shoot was never created successfully", false, core.ShootStatus{}),
				Entry("shoot is still being created", false, core.ShootStatus{
					LastOperation: &core.LastOperation{
						Type:  core.LastOperationTypeCreate,
						State: core.LastOperationStateProcessing,
					},
				}),
				Entry("shoot was created successfully", true, core.ShootStatus{
					LastOperation: &core.LastOperation{
						Type:  core.LastOperationTypeCreate,
						State: core.LastOperationStateSucceeded,
					},
				}),
				Entry("shoot is in reconciliation phase", true, core.ShootStatus{
					LastOperation: &core.LastOperation{
						Type: core.LastOperationTypeReconcile,
					},
				}),
				Entry("shoot is in deletion phase", false, core.ShootStatus{
					LastOperation: &core.LastOperation{
						Type: core.LastOperationTypeDelete,
					},
				}),
				Entry("shoot is in migration phase", false, core.ShootStatus{
					LastOperation: &core.LastOperation{
						Type: core.LastOperationTypeMigrate,
					},
				}),
				Entry("shoot is in restoration phase", false, core.ShootStatus{
					LastOperation: &core.LastOperation{
						Type: core.LastOperationTypeRestore,
					},
				}),
				Entry("shoot was restored successfully", true, core.ShootStatus{
					LastOperation: &core.LastOperation{
						Type:  core.LastOperationTypeRestore,
						State: core.LastOperationStateSucceeded,
					},
				}),
				Entry("ca rotation phase is prepare", false, core.ShootStatus{
					LastOperation: &core.LastOperation{
						Type: core.LastOperationTypeReconcile,
					},
					Credentials: &core.ShootCredentials{
						Rotation: &core.ShootCredentialsRotation{
							CertificateAuthorities: &core.ShootCARotation{
								Phase: core.RotationPreparing,
							},
						},
					},
				}),
				Entry("ca rotation phase is prepared", false, core.ShootStatus{
					LastOperation: &core.LastOperation{
						Type: core.LastOperationTypeReconcile,
					},
					Credentials: &core.ShootCredentials{
						Rotation: &core.ShootCredentialsRotation{
							CertificateAuthorities: &core.ShootCARotation{
								Phase: core.RotationPrepared,
							},
						},
					},
				}),
				Entry("ca rotation phase is complete", false, core.ShootStatus{
					LastOperation: &core.LastOperation{
						Type: core.LastOperationTypeReconcile,
					},
					Credentials: &core.ShootCredentials{
						Rotation: &core.ShootCredentialsRotation{
							CertificateAuthorities: &core.ShootCARotation{
								Phase: core.RotationCompleting,
							},
						},
					},
				}),
				Entry("ca rotation phase is completed", true, core.ShootStatus{
					LastOperation: &core.LastOperation{
						Type: core.LastOperationTypeReconcile,
					},
					Credentials: &core.ShootCredentials{
						Rotation: &core.ShootCredentialsRotation{
							CertificateAuthorities: &core.ShootCARotation{
								Phase: core.RotationCompleted,
							},
						},
					},
				}),
			)

			DescribeTable("completing CA rotation",
				func(allowed bool, status core.ShootStatus) {
					defer test.WithFeatureGate(utilfeature.DefaultFeatureGate, features.ShootCARotation, true)()
					metav1.SetMetaDataAnnotation(&shoot.ObjectMeta, "gardener.cloud/operation", "rotate-ca-complete")
					shoot.Status = status

					matcher := BeEmpty()
					if !allowed {
						matcher = ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
							"Type":  Equal(field.ErrorTypeForbidden),
							"Field": Equal("metadata.annotations[gardener.cloud/operation]"),
						})))
					}

					Expect(ValidateShoot(shoot)).To(matcher)
				},

				Entry("ca rotation phase is prepare", false, core.ShootStatus{
					LastOperation: &core.LastOperation{
						Type: core.LastOperationTypeReconcile,
					},
					Credentials: &core.ShootCredentials{
						Rotation: &core.ShootCredentialsRotation{
							CertificateAuthorities: &core.ShootCARotation{
								Phase: core.RotationPreparing,
							},
						},
					},
				}),
				Entry("ca rotation phase is prepared", true, core.ShootStatus{
					LastOperation: &core.LastOperation{
						Type: core.LastOperationTypeReconcile,
					},
					Credentials: &core.ShootCredentials{
						Rotation: &core.ShootCredentialsRotation{
							CertificateAuthorities: &core.ShootCARotation{
								Phase: core.RotationPrepared,
							},
						},
					},
				}),
				Entry("ca rotation phase is complete", false, core.ShootStatus{
					LastOperation: &core.LastOperation{
						Type: core.LastOperationTypeReconcile,
					},
					Credentials: &core.ShootCredentials{
						Rotation: &core.ShootCredentialsRotation{
							CertificateAuthorities: &core.ShootCARotation{
								Phase: core.RotationCompleting,
							},
						},
					},
				}),
				Entry("ca rotation phase is completed", false, core.ShootStatus{
					LastOperation: &core.LastOperation{
						Type: core.LastOperationTypeReconcile,
					},
					Credentials: &core.ShootCredentials{
						Rotation: &core.ShootCredentialsRotation{
							CertificateAuthorities: &core.ShootCARotation{
								Phase: core.RotationCompleted,
							},
						},
					},
				}),
			)
		})
	})

	Describe("#ValidateShootStatus, #ValidateShootStatusUpdate", func() {
		var (
			shoot    *core.Shoot
			newShoot *core.Shoot
		)
		BeforeEach(func() {
			shoot = &core.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "shoot",
					Namespace: "my-namespace",
				},
				Spec:   core.ShootSpec{},
				Status: core.ShootStatus{},
			}

			newShoot = prepareShootForUpdate(shoot)
		})

		Context("uid checks", func() {
			It("should allow setting the uid", func() {
				newShoot.Status.UID = types.UID("1234")

				errorList := ValidateShootStatusUpdate(newShoot.Status, shoot.Status)

				Expect(errorList).To(HaveLen(0))
			})

			It("should forbid changing the uid", func() {
				shoot.Status.UID = types.UID("1234")
				newShoot.Status.UID = types.UID("1235")

				errorList := ValidateShootStatusUpdate(newShoot.Status, shoot.Status)

				Expect(errorList).To(HaveLen(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("status.uid"),
				}))
			})
		})

		Context("technical id checks", func() {
			It("should allow setting the technical id", func() {
				newShoot.Status.TechnicalID = "shoot--foo--bar"

				errorList := ValidateShootStatusUpdate(newShoot.Status, shoot.Status)

				Expect(errorList).To(HaveLen(0))
			})

			It("should forbid changing the technical id", func() {
				shoot.Status.TechnicalID = "shoot-foo-bar"
				newShoot.Status.TechnicalID = "shoot--foo--bar"

				errorList := ValidateShootStatusUpdate(newShoot.Status, shoot.Status)

				Expect(errorList).To(HaveLen(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("status.technicalID"),
				}))
			})
		})

		Context("validate shoot cluster identity update", func() {
			clusterIdentity := "newClusterIdentity"
			It("should not fail to set the cluster identity if it is missing", func() {
				newShoot.Status.ClusterIdentity = &clusterIdentity
				errorList := ValidateShootStatusUpdate(newShoot.Status, shoot.Status)
				Expect(errorList).To(HaveLen(0))
			})

			It("should fail to set the cluster identity if it is already set", func() {
				newShoot.Status.ClusterIdentity = &clusterIdentity
				shoot.Status.ClusterIdentity = pointer.String("oldClusterIdentity")
				errorList := ValidateShootStatusUpdate(newShoot.Status, shoot.Status)
				Expect(errorList).To(HaveLen(1))
				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("status.clusterIdentity"),
					"Detail": ContainSubstring(`field is immutable`),
				}))
			})
		})

		Context("validate shoot advertise address update", func() {
			It("should fail for empty name", func() {
				newShoot.Status.AdvertisedAddresses = []core.ShootAdvertisedAddress{
					{Name: ""},
				}

				errorList := ValidateShootStatusUpdate(newShoot.Status, shoot.Status)
				Expect(errorList).To(HaveLen(1))
				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("status.advertisedAddresses[0].name"),
					"Detail": ContainSubstring(`field must not be empty`),
				}))
			})
			It("should fail for duplicate name", func() {
				newShoot.Status.AdvertisedAddresses = []core.ShootAdvertisedAddress{
					{Name: "a", URL: "https://foo.bar"},
					{Name: "a", URL: "https://foo.bar"},
				}

				errorList := ValidateShootStatusUpdate(newShoot.Status, shoot.Status)
				Expect(errorList).To(HaveLen(1))
				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":  Equal(field.ErrorTypeDuplicate),
					"Field": Equal("status.advertisedAddresses[1].name"),
				}))
			})
			It("should fail for invalid URL", func() {
				newShoot.Status.AdvertisedAddresses = []core.ShootAdvertisedAddress{
					{Name: "a", URL: "://foo.bar"},
				}

				errorList := ValidateShootStatusUpdate(newShoot.Status, shoot.Status)
				Expect(errorList).To(HaveLen(1))
				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("status.advertisedAddresses[0].url"),
					"Detail": ContainSubstring(`url must be a valid URL:`),
				}))
			})
			It("should fail for http URL", func() {
				newShoot.Status.AdvertisedAddresses = []core.ShootAdvertisedAddress{
					{Name: "a", URL: "http://foo.bar"},
				}

				errorList := ValidateShootStatusUpdate(newShoot.Status, shoot.Status)
				Expect(errorList).To(HaveLen(1))
				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("status.advertisedAddresses[0].url"),
					"Detail": ContainSubstring(`'https' is the only allowed URL scheme`),
				}))
			})
			It("should fail for URL without host", func() {
				newShoot.Status.AdvertisedAddresses = []core.ShootAdvertisedAddress{
					{Name: "a", URL: "https://"},
				}

				errorList := ValidateShootStatusUpdate(newShoot.Status, shoot.Status)
				Expect(errorList).To(HaveLen(1))
				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("status.advertisedAddresses[0].url"),
					"Detail": ContainSubstring(`host must be provided`),
				}))
			})
			It("should fail for URL with path", func() {
				newShoot.Status.AdvertisedAddresses = []core.ShootAdvertisedAddress{
					{Name: "a", URL: "https://foo.bar/baz"},
				}

				errorList := ValidateShootStatusUpdate(newShoot.Status, shoot.Status)
				Expect(errorList).To(HaveLen(1))
				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("status.advertisedAddresses[0].url"),
					"Detail": ContainSubstring(`path is not permitted in the URL`),
				}))
			})
			It("should fail for URL with user information", func() {
				newShoot.Status.AdvertisedAddresses = []core.ShootAdvertisedAddress{
					{Name: "a", URL: "https://john:doe@foo.bar"},
				}

				errorList := ValidateShootStatusUpdate(newShoot.Status, shoot.Status)
				Expect(errorList).To(HaveLen(1))
				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("status.advertisedAddresses[0].url"),
					"Detail": ContainSubstring(`user information is not permitted in the URL`),
				}))
			})
			It("should fail for URL with fragment", func() {
				newShoot.Status.AdvertisedAddresses = []core.ShootAdvertisedAddress{
					{Name: "a", URL: "https://foo.bar#some-fragment"},
				}

				errorList := ValidateShootStatusUpdate(newShoot.Status, shoot.Status)
				Expect(errorList).To(HaveLen(1))
				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("status.advertisedAddresses[0].url"),
					"Detail": ContainSubstring(`fragments are not permitted in the URL`),
				}))
			})
			It("should fail for URL with query parameters", func() {
				newShoot.Status.AdvertisedAddresses = []core.ShootAdvertisedAddress{
					{Name: "a", URL: "https://foo.bar?some=query"},
				}

				errorList := ValidateShootStatusUpdate(newShoot.Status, shoot.Status)
				Expect(errorList).To(HaveLen(1))
				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("status.advertisedAddresses[0].url"),
					"Detail": ContainSubstring(`query parameters are not permitted in the URL`),
				}))
			})
			It("should succeed correct addresses", func() {
				newShoot.Status.AdvertisedAddresses = []core.ShootAdvertisedAddress{
					{Name: "a", URL: "https://foo.bar"},
					{Name: "b", URL: "https://foo.bar:443"},
				}

				errorList := ValidateShootStatusUpdate(newShoot.Status, shoot.Status)
				Expect(errorList).To(BeEmpty())
			})
		})
	})

	Describe("#ValidateWorker", func() {
		DescribeTable("validate worker machine",
			func(machine core.Machine, matcher gomegatypes.GomegaMatcher) {
				maxSurge := intstr.FromInt(1)
				maxUnavailable := intstr.FromInt(0)
				worker := core.Worker{
					Name:           "worker-name",
					Machine:        machine,
					MaxSurge:       &maxSurge,
					MaxUnavailable: &maxUnavailable,
				}
				errList := ValidateWorker(worker, "", nil, false)

				Expect(errList).To(matcher)
			},

			Entry("empty machine type",
				core.Machine{
					Type: "",
					Image: &core.ShootMachineImage{
						Name:    "image-name",
						Version: "1.0.0",
					},
				},
				ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("machine.type"),
				}))),
			),
			Entry("empty machine image name",
				core.Machine{
					Type: "large",
					Image: &core.ShootMachineImage{
						Name:    "",
						Version: "1.0.0",
					},
				},
				ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("machine.image.name"),
				}))),
			),
			Entry("empty machine image version",
				core.Machine{
					Type: "large",
					Image: &core.ShootMachineImage{
						Name:    "image-name",
						Version: "",
					},
				},
				ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("machine.image.version"),
				}))),
			),
		)

		DescribeTable("reject when maxUnavailable and maxSurge are invalid",
			func(maxUnavailable, maxSurge intstr.IntOrString, expectType field.ErrorType) {
				worker := core.Worker{
					Name: "worker-name",
					Machine: core.Machine{
						Type: "large",
						Image: &core.ShootMachineImage{
							Name:    "image-name",
							Version: "1.0.0",
						},
					},
					MaxSurge:       &maxSurge,
					MaxUnavailable: &maxUnavailable,
				}
				errList := ValidateWorker(worker, "", nil, false)

				Expect(errList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(expectType),
				}))))
			},

			// double zero values (percent or int)
			Entry("two zero integers", intstr.FromInt(0), intstr.FromInt(0), field.ErrorTypeInvalid),
			Entry("zero int and zero percent", intstr.FromInt(0), intstr.FromString("0%"), field.ErrorTypeInvalid),
			Entry("zero percent and zero int", intstr.FromString("0%"), intstr.FromInt(0), field.ErrorTypeInvalid),
			Entry("two zero percents", intstr.FromString("0%"), intstr.FromString("0%"), field.ErrorTypeInvalid),

			// greater than 100
			Entry("maxUnavailable greater than 100 percent", intstr.FromString("101%"), intstr.FromString("100%"), field.ErrorTypeInvalid),

			// below zero tests
			Entry("values are not below zero", intstr.FromInt(-1), intstr.FromInt(0), field.ErrorTypeInvalid),
			Entry("percentage is not less than zero", intstr.FromString("-90%"), intstr.FromString("90%"), field.ErrorTypeInvalid),
		)

		DescribeTable("reject when labels are invalid",
			func(labels map[string]string, expectType field.ErrorType) {
				maxSurge := intstr.FromInt(1)
				maxUnavailable := intstr.FromInt(0)
				worker := core.Worker{
					Name: "worker-name",
					Machine: core.Machine{
						Type: "large",
						Image: &core.ShootMachineImage{
							Name:    "image-name",
							Version: "1.0.0",
						},
					},
					MaxSurge:       &maxSurge,
					MaxUnavailable: &maxUnavailable,
					Labels:         labels,
				}
				errList := ValidateWorker(worker, "", nil, false)

				Expect(errList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(expectType),
				}))))
			},

			// invalid keys
			Entry("missing prefix", map[string]string{"/foo": "bar"}, field.ErrorTypeInvalid),
			Entry("too long name", map[string]string{"foo/somethingthatiswaylongerthanthelimitofthiswhichissixtythreecharacters": "baz"}, field.ErrorTypeInvalid),
			Entry("too many parts", map[string]string{"foo/bar/baz": "null"}, field.ErrorTypeInvalid),
			Entry("invalid name", map[string]string{"foo/bar%baz": "null"}, field.ErrorTypeInvalid),

			// invalid values
			Entry("too long", map[string]string{"foo": "somethingthatiswaylongerthanthelimitofthiswhichissixtythreecharacters"}, field.ErrorTypeInvalid),
			Entry("invalid", map[string]string{"foo": "no/slashes/allowed"}, field.ErrorTypeInvalid),
		)

		DescribeTable("reject when annotations are invalid",
			func(annotations map[string]string, expectType field.ErrorType) {
				maxSurge := intstr.FromInt(1)
				maxUnavailable := intstr.FromInt(0)
				worker := core.Worker{
					Name: "worker-name",
					Machine: core.Machine{
						Type: "large",
						Image: &core.ShootMachineImage{
							Name:    "image-name",
							Version: "1.0.0",
						},
					},
					MaxSurge:       &maxSurge,
					MaxUnavailable: &maxUnavailable,
					Annotations:    annotations,
				}
				errList := ValidateWorker(worker, "", nil, false)

				Expect(errList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(expectType),
				}))))
			},

			// invalid keys
			Entry("missing prefix", map[string]string{"/foo": "bar"}, field.ErrorTypeInvalid),
			Entry("too long name", map[string]string{"foo/somethingthatiswaylongerthanthelimitofthiswhichissixtythreecharacters": "baz"}, field.ErrorTypeInvalid),
			Entry("too many parts", map[string]string{"foo/bar/baz": "null"}, field.ErrorTypeInvalid),
			Entry("invalid name", map[string]string{"foo/bar%baz": "null"}, field.ErrorTypeInvalid),

			// invalid value
			Entry("too long", map[string]string{"foo": strings.Repeat("a", 262142)}, field.ErrorTypeTooLong),
		)

		DescribeTable("reject when taints are invalid",
			func(taints []corev1.Taint, expectType field.ErrorType) {
				maxSurge := intstr.FromInt(1)
				maxUnavailable := intstr.FromInt(0)
				worker := core.Worker{
					Name: "worker-name",
					Machine: core.Machine{
						Type: "large",
						Image: &core.ShootMachineImage{
							Name:    "image-name",
							Version: "1.0.0",
						},
					},
					MaxSurge:       &maxSurge,
					MaxUnavailable: &maxUnavailable,
					Taints:         taints,
				}
				errList := ValidateWorker(worker, "", nil, false)

				Expect(errList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(expectType),
				}))))
			},

			// invalid keys
			Entry("missing prefix", []corev1.Taint{{Key: "/foo", Value: "bar", Effect: corev1.TaintEffectNoSchedule}}, field.ErrorTypeInvalid),
			Entry("missing prefix", []corev1.Taint{{Key: "/foo", Value: "bar", Effect: corev1.TaintEffectNoSchedule}}, field.ErrorTypeInvalid),
			Entry("too long name", []corev1.Taint{{Key: "foo/somethingthatiswaylongerthanthelimitofthiswhichissixtythreecharacters", Value: "bar", Effect: corev1.TaintEffectNoSchedule}}, field.ErrorTypeInvalid),
			Entry("too many parts", []corev1.Taint{{Key: "foo/bar/baz", Value: "bar", Effect: corev1.TaintEffectNoSchedule}}, field.ErrorTypeInvalid),
			Entry("invalid name", []corev1.Taint{{Key: "foo/bar%baz", Value: "bar", Effect: corev1.TaintEffectNoSchedule}}, field.ErrorTypeInvalid),

			// invalid values
			Entry("too long", []corev1.Taint{{Key: "foo", Value: "somethingthatiswaylongerthanthelimitofthiswhichissixtythreecharacters", Effect: corev1.TaintEffectNoSchedule}}, field.ErrorTypeInvalid),
			Entry("invalid", []corev1.Taint{{Key: "foo", Value: "no/slashes/allowed", Effect: corev1.TaintEffectNoSchedule}}, field.ErrorTypeInvalid),

			// invalid effects
			Entry("no effect", []corev1.Taint{{Key: "foo", Value: "bar"}}, field.ErrorTypeRequired),
			Entry("non-existing", []corev1.Taint{{Key: "foo", Value: "bar", Effect: corev1.TaintEffect("does-not-exist")}}, field.ErrorTypeNotSupported),

			// uniqueness by key/effect
			Entry("not unique", []corev1.Taint{{Key: "foo", Value: "bar", Effect: corev1.TaintEffectNoSchedule}, {Key: "foo", Value: "baz", Effect: corev1.TaintEffectNoSchedule}}, field.ErrorTypeDuplicate),
		)

		It("should reject if volume is undefined and data volumes are defined", func() {
			maxSurge := intstr.FromInt(1)
			maxUnavailable := intstr.FromInt(0)
			dataVolumes := []core.DataVolume{{Name: "vol1-name", VolumeSize: "75Gi"}}
			worker := core.Worker{
				Name: "worker-name",
				Machine: core.Machine{
					Type: "large",
					Image: &core.ShootMachineImage{
						Name:    "image-name",
						Version: "1.0.0",
					},
				},
				MaxSurge:       &maxSurge,
				MaxUnavailable: &maxUnavailable,
				DataVolumes:    dataVolumes,
			}
			errList := ValidateWorker(worker, "", nil, false)
			Expect(errList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("volume"),
			}))))
		})

		It("should reject if data volume size does not match size regex", func() {
			maxSurge := intstr.FromInt(1)
			maxUnavailable := intstr.FromInt(0)
			name := "vol1-name"
			vol := core.Volume{Name: &name, VolumeSize: "75Gi"}
			dataVolumes := []core.DataVolume{{Name: name, VolumeSize: "75Gi"}, {Name: "vol2-name", VolumeSize: "12MiB"}}
			worker := core.Worker{
				Name: "worker-name",
				Machine: core.Machine{
					Type: "large",
					Image: &core.ShootMachineImage{
						Name:    "image-name",
						Version: "1.0.0",
					},
				},
				MaxSurge:       &maxSurge,
				MaxUnavailable: &maxUnavailable,
				Volume:         &vol,
				DataVolumes:    dataVolumes,
			}
			errList := ValidateWorker(worker, "", nil, false)
			Expect(errList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":     Equal(field.ErrorTypeInvalid),
				"Field":    Equal("dataVolumes[1].size"),
				"BadValue": Equal("12MiB"),
			}))))
		})

		It("should reject if data volume name is invalid", func() {
			maxSurge := intstr.FromInt(1)
			maxUnavailable := intstr.FromInt(0)
			name1 := "vol1-name-is-too-long-for-test"
			name2 := "not%dns/1123"
			vol := core.Volume{Name: &name1, VolumeSize: "75Gi"}
			dataVolumes := []core.DataVolume{{VolumeSize: "75Gi"}, {Name: name1, VolumeSize: "75Gi"}, {Name: name2, VolumeSize: "75Gi"}}
			worker := core.Worker{
				Name: "worker-name",
				Machine: core.Machine{
					Type: "large",
					Image: &core.ShootMachineImage{
						Name:    "image-name",
						Version: "1.0.0",
					},
				},
				MaxSurge:       &maxSurge,
				MaxUnavailable: &maxUnavailable,
				Volume:         &vol,
				DataVolumes:    dataVolumes,
			}
			errList := ValidateWorker(worker, "", nil, false)
			Expect(errList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("dataVolumes[0].name"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeTooLong),
					"Field": Equal("dataVolumes[1].name"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("dataVolumes[2].name"),
				})),
			))
		})

		It("should accept if kubeletDataVolumeName refers to defined data volume", func() {
			maxSurge := intstr.FromInt(1)
			maxUnavailable := intstr.FromInt(0)
			name := "vol1-name"
			vol := core.Volume{Name: &name, VolumeSize: "75Gi"}
			dataVolumes := []core.DataVolume{{Name: name, VolumeSize: "75Gi"}}
			worker := core.Worker{
				Name: "worker-name",
				Machine: core.Machine{
					Type: "large",
					Image: &core.ShootMachineImage{
						Name:    "image-name",
						Version: "1.0.0",
					},
				},
				MaxSurge:              &maxSurge,
				MaxUnavailable:        &maxUnavailable,
				Volume:                &vol,
				DataVolumes:           dataVolumes,
				KubeletDataVolumeName: &name,
			}
			errList := ValidateWorker(worker, "", nil, false)
			Expect(errList).To(ConsistOf())
		})

		It("should reject if kubeletDataVolumeName refers to undefined data volume", func() {
			maxSurge := intstr.FromInt(1)
			maxUnavailable := intstr.FromInt(0)
			name1 := "vol1-name"
			name2 := "vol2-name"
			name3 := "vol3-name"
			vol := core.Volume{Name: &name1, VolumeSize: "75Gi"}
			dataVolumes := []core.DataVolume{{Name: name1, VolumeSize: "75Gi"}, {Name: name2, VolumeSize: "75Gi"}}
			worker := core.Worker{
				Name: "worker-name",
				Machine: core.Machine{
					Type: "large",
					Image: &core.ShootMachineImage{
						Name:    "image-name",
						Version: "1.0.0",
					},
				},
				MaxSurge:              &maxSurge,
				MaxUnavailable:        &maxUnavailable,
				Volume:                &vol,
				DataVolumes:           dataVolumes,
				KubeletDataVolumeName: &name3,
			}
			errList := ValidateWorker(worker, "", nil, false)
			Expect(errList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("kubeletDataVolumeName"),
				})),
			))
		})

		It("should reject if data volume names are duplicated", func() {
			maxSurge := intstr.FromInt(1)
			maxUnavailable := intstr.FromInt(0)
			name1 := "vol1-name"
			name2 := "vol2-name"
			vol := core.Volume{Name: &name1, VolumeSize: "75Gi"}
			dataVolumes := []core.DataVolume{{Name: name1, VolumeSize: "75Gi"}, {Name: name1, VolumeSize: "75Gi"}, {Name: name2, VolumeSize: "75Gi"}, {Name: name1, VolumeSize: "75Gi"}}
			worker := core.Worker{
				Name: "worker-name",
				Machine: core.Machine{
					Type: "large",
					Image: &core.ShootMachineImage{
						Name:    "image-name",
						Version: "1.0.0",
					},
				},
				MaxSurge:       &maxSurge,
				MaxUnavailable: &maxUnavailable,
				Volume:         &vol,
				DataVolumes:    dataVolumes,
			}
			errList := ValidateWorker(worker, "", nil, false)
			Expect(errList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeDuplicate),
					"Field": Equal("dataVolumes[1].name"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeDuplicate),
					"Field": Equal("dataVolumes[3].name"),
				})),
			))
		})

		It("should reject if kubelet feature gates are invalid", func() {
			maxSurge := intstr.FromInt(1)
			maxUnavailable := intstr.FromInt(0)
			worker := core.Worker{
				Name: "worker-name",
				Machine: core.Machine{
					Type: "large",
					Image: &core.ShootMachineImage{
						Name:    "image-name",
						Version: "1.0.0",
					},
				},
				MaxSurge:       &maxSurge,
				MaxUnavailable: &maxUnavailable,
				Kubernetes: &core.WorkerKubernetes{
					Kubelet: &core.KubeletConfig{
						KubernetesConfig: core.KubernetesConfig{
							FeatureGates: map[string]bool{
								"AnyVolumeDataSource":      true,
								"CustomResourceValidation": true,
								"Foo":                      true,
							},
						},
					},
				},
			}
			errList := ValidateWorker(worker, "1.18.14", nil, false)
			Expect(errList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("kubernetes.kubelet.featureGates.CustomResourceValidation"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("kubernetes.kubelet.featureGates.Foo"),
				})),
			))
		})

		DescribeTable("validate CRI name depending on the kubernetes version",
			func(name core.CRIName, kubernetesVersion string, matcher gomegatypes.GomegaMatcher) {
				worker := core.Worker{
					Name: "worker",
					CRI:  &core.CRI{Name: name},
				}

				errList := ValidateCRI(worker.CRI, kubernetesVersion, field.NewPath("cri"))

				Expect(errList).To(matcher)
			},

			Entry("containerd is a valid CRI name for k8s < 1.23", core.CRINameContainerD, "1.22.0", HaveLen(0)),
			Entry("containerd is a valid CRI name for k8s >= 1.23", core.CRINameContainerD, "1.23.0", HaveLen(0)),
			Entry("docker is a valid CRI name for k8s < 1.23", core.CRINameDocker, "1.22.0", HaveLen(0)),
			Entry("docker is NOT a valid CRI name for k8s >= 1.23", core.CRINameDocker, "1.23.0", ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeForbidden),
				"Field": Equal("cri.name"),
			})))),
			Entry("not valid CRI name for k8s < 1.23", core.CRIName("other"), "1.22.0", ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeNotSupported),
				"Field": Equal("cri.name"),
			})))),
			Entry("not valid CRI name for k8s >= 1.23", core.CRIName("other"), "1.23.0", ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeNotSupported),
				"Field": Equal("cri.name"),
			})))),
		)

		It("validate that container runtime has a type", func() {
			worker := core.Worker{
				Name: "worker",
				CRI: &core.CRI{
					Name:              core.CRINameContainerD,
					ContainerRuntimes: []core.ContainerRuntime{{Type: "gVisor"}, {Type: ""}},
				},
			}

			errList := ValidateCRI(worker.CRI, "1.22.0", field.NewPath("cri"))
			Expect(errList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("cri.containerruntimes[1].type"),
				})),
			))
		})

		It("validate duplicate container runtime types", func() {
			worker := core.Worker{
				Name: "worker",
				CRI: &core.CRI{
					Name:              core.CRINameContainerD,
					ContainerRuntimes: []core.ContainerRuntime{{Type: "gVisor"}, {Type: "gVisor"}},
				},
			}

			errList := ValidateCRI(worker.CRI, "1.22.0", field.NewPath("cri"))
			Expect(errList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeDuplicate),
					"Field": Equal("cri.containerruntimes[1].type"),
				})),
			))
		})
	})

	Describe("#ValidateWorkers", func() {
		var (
			zero int32 = 0
			one  int32 = 1
		)

		DescribeTable("validate that at least one active worker pool is configured",
			func(min1, max1, min2, max2 int32, matcher gomegatypes.GomegaMatcher) {
				systemComponents := &core.WorkerSystemComponents{
					Allow: true,
				}
				workers := []core.Worker{
					{
						Name:             "one",
						Minimum:          min1,
						Maximum:          max1,
						SystemComponents: systemComponents,
					},
					{
						Name:             "two",
						Minimum:          min2,
						Maximum:          max2,
						SystemComponents: systemComponents,
					},
				}

				Expect(ValidateWorkers(workers, field.NewPath("workers"))).To(matcher)
			},

			Entry("at least one worker pool min>0, max>0", zero, zero, one, one, HaveLen(0)),
			Entry("all worker pools min=max=0", zero, zero, zero, zero, ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(field.ErrorTypeForbidden),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(field.ErrorTypeForbidden),
				})),
			)),
		)

		DescribeTable("validate that at least one worker pool is able to host system components",
			func(min1, max1, min2, max2 int32, allowSystemComponents1, allowSystemComponents2 bool, taints1, taints2 []corev1.Taint, matcher gomegatypes.GomegaMatcher) {
				workers := []core.Worker{
					{
						Name:    "one-active",
						Minimum: min1,
						Maximum: max1,
						SystemComponents: &core.WorkerSystemComponents{
							Allow: allowSystemComponents1,
						},
						Taints: taints1,
					},
					{
						Name:    "two-active",
						Minimum: min2,
						Maximum: max2,
						SystemComponents: &core.WorkerSystemComponents{
							Allow: allowSystemComponents2,
						},
						Taints: taints2,
					},
				}

				Expect(ValidateWorkers(workers, field.NewPath("workers"))).To(matcher)
			},

			Entry("all worker pools min=max=0", zero, zero, zero, zero, true, true, nil, nil, ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(field.ErrorTypeForbidden),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(field.ErrorTypeForbidden),
				})),
			)),
			Entry("at least one worker pool allows system components", zero, zero, one, one, true, true, nil, nil, HaveLen(0)),
			Entry("one active but taints prevent scheduling", one, one, zero, zero, true, true, []corev1.Taint{{Effect: corev1.TaintEffectNoSchedule}}, nil, ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(field.ErrorTypeForbidden),
				})),
			)),
		)

		DescribeTable("ensure that at least one worker pool exists that either has no taints or only those with `PreferNoSchedule` effect",
			func(matcher gomegatypes.GomegaMatcher, taints ...[]corev1.Taint) {
				var (
					workers          []core.Worker
					systemComponents = &core.WorkerSystemComponents{
						Allow: true,
					}
				)

				for i, t := range taints {
					workers = append(workers, core.Worker{
						Name:             "pool-" + strconv.Itoa(i),
						Minimum:          1,
						Maximum:          2,
						Taints:           t,
						SystemComponents: systemComponents,
					})
				}

				Expect(ValidateWorkers(workers, field.NewPath("workers"))).To(matcher)
			},

			Entry(
				"no pools with taints",
				HaveLen(0),
				[]corev1.Taint{},
			),
			Entry(
				"all pools with PreferNoSchedule taints",
				HaveLen(0),
				[]corev1.Taint{{Effect: corev1.TaintEffectPreferNoSchedule}},
			),
			Entry(
				"at least one pools with either no or PreferNoSchedule taints (1)",
				HaveLen(0),
				[]corev1.Taint{{Effect: corev1.TaintEffectNoExecute}},
				[]corev1.Taint{{Effect: corev1.TaintEffectPreferNoSchedule}},
			),
			Entry(
				"at least one pools with either no or PreferNoSchedule taints (2)",
				HaveLen(0),
				[]corev1.Taint{{Effect: corev1.TaintEffectNoSchedule}},
				[]corev1.Taint{{Effect: corev1.TaintEffectPreferNoSchedule}},
			),
			Entry(
				"all pools with NoSchedule taints",
				ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type": Equal(field.ErrorTypeForbidden),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type": Equal(field.ErrorTypeForbidden),
					})),
				),
				[]corev1.Taint{{Effect: corev1.TaintEffectNoSchedule}},
			),
			Entry(
				"all pools with NoExecute taints",
				ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type": Equal(field.ErrorTypeForbidden),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type": Equal(field.ErrorTypeForbidden),
					})),
				),
				[]corev1.Taint{{Effect: corev1.TaintEffectNoExecute}},
			),
			Entry(
				"all pools with either NoSchedule or NoExecute taints",
				ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type": Equal(field.ErrorTypeForbidden),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type": Equal(field.ErrorTypeForbidden),
					})),
				),
				[]corev1.Taint{{Effect: corev1.TaintEffectNoExecute}},
				[]corev1.Taint{{Effect: corev1.TaintEffectNoSchedule}},
			),
		)
	})

	Describe("#ValidateKubeletConfiguration", func() {
		validResourceQuantityValueMi := "100Mi"
		validResourceQuantityValueKi := "100"
		invalidResourceQuantityValue := "-100Mi"
		validPercentValue := "5%"
		invalidPercentValueLow := "-5%"
		invalidPercentValueHigh := "110%"
		invalidValue := "5X"

		DescribeTable("validate the kubelet configuration - EvictionHard & EvictionSoft",
			func(memoryAvailable, imagefsAvailable, imagefsInodesFree, nodefsAvailable, nodefsInodesFree string, matcher gomegatypes.GomegaMatcher) {
				kubeletConfig := core.KubeletConfig{
					EvictionHard: &core.KubeletConfigEviction{
						MemoryAvailable:   &memoryAvailable,
						ImageFSAvailable:  &imagefsAvailable,
						ImageFSInodesFree: &imagefsInodesFree,
						NodeFSAvailable:   &nodefsAvailable,
						NodeFSInodesFree:  &nodefsInodesFree,
					},
					EvictionSoft: &core.KubeletConfigEviction{
						MemoryAvailable:   &memoryAvailable,
						ImageFSAvailable:  &imagefsAvailable,
						ImageFSInodesFree: &imagefsInodesFree,
						NodeFSAvailable:   &nodefsAvailable,
						NodeFSInodesFree:  &nodefsInodesFree,
					},
				}

				errList := ValidateKubeletConfig(kubeletConfig, "", true, nil)

				Expect(errList).To(matcher)
			},

			Entry("valid configuration", validResourceQuantityValueMi, validResourceQuantityValueKi, validPercentValue, validPercentValue, validPercentValue, HaveLen(0)),
			Entry("only allow resource.Quantity or percent value for any value", invalidValue, validPercentValue, validPercentValue, validPercentValue, validPercentValue, ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(field.NewPath("evictionHard.memoryAvailable").String()),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(field.NewPath("evictionSoft.memoryAvailable").String()),
				})))),
			Entry("do not allow negative resource.Quantity", invalidResourceQuantityValue, validPercentValue, validPercentValue, validPercentValue, validPercentValue, ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(field.NewPath("evictionHard.memoryAvailable").String()),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(field.NewPath("evictionSoft.memoryAvailable").String()),
				})))),
			Entry("do not allow negative percentages", invalidPercentValueLow, validPercentValue, validPercentValue, validPercentValue, validPercentValue, ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(field.NewPath("evictionHard.memoryAvailable").String()),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(field.NewPath("evictionSoft.memoryAvailable").String()),
				})))),
			Entry("do not allow percentages > 100", invalidPercentValueHigh, validPercentValue, validPercentValue, validPercentValue, validPercentValue, ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(field.NewPath("evictionHard.memoryAvailable").String()),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(field.NewPath("evictionSoft.memoryAvailable").String()),
				})))),
		)

		Describe("pod pids limits", func() {
			It("should ensure pod pids limits are non-negative", func() {
				var podPIDsLimit int64 = -1
				kubeletConfig := core.KubeletConfig{
					PodPIDsLimit: &podPIDsLimit,
				}

				errList := ValidateKubeletConfig(kubeletConfig, "", true, nil)

				Expect(errList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("podPIDsLimit"),
				}))))
			})

			It("should ensure pod pids limits are at least 100", func() {
				var podPIDsLimit int64 = 99
				kubeletConfig := core.KubeletConfig{
					PodPIDsLimit: &podPIDsLimit,
				}

				errList := ValidateKubeletConfig(kubeletConfig, "", true, nil)

				Expect(errList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("podPIDsLimit"),
				}))))
			})

			It("should allow pod pids limits of at least 100", func() {
				var podPIDsLimit int64 = 100
				kubeletConfig := core.KubeletConfig{
					PodPIDsLimit: &podPIDsLimit,
				}

				errList := ValidateKubeletConfig(kubeletConfig, "", true, nil)

				Expect(errList).To(BeEmpty())
			})
		})

		validResourceQuantity := resource.MustParse(validResourceQuantityValueMi)
		invalidResourceQuantity := resource.MustParse(invalidResourceQuantityValue)

		DescribeTable("validate the kubelet configuration - EvictionMinimumReclaim",
			func(memoryAvailable, imagefsAvailable, imagefsInodesFree, nodefsAvailable, nodefsInodesFree resource.Quantity, matcher gomegatypes.GomegaMatcher) {
				kubeletConfig := core.KubeletConfig{
					EvictionMinimumReclaim: &core.KubeletConfigEvictionMinimumReclaim{
						MemoryAvailable:   &memoryAvailable,
						ImageFSAvailable:  &imagefsAvailable,
						ImageFSInodesFree: &imagefsInodesFree,
						NodeFSAvailable:   &nodefsAvailable,
						NodeFSInodesFree:  &nodefsInodesFree,
					},
				}

				errList := ValidateKubeletConfig(kubeletConfig, "", true, nil)

				Expect(errList).To(matcher)
			},

			Entry("valid configuration", validResourceQuantity, validResourceQuantity, validResourceQuantity, validResourceQuantity, validResourceQuantity, HaveLen(0)),
			Entry("only allow positive resource.Quantity for any value", invalidResourceQuantity, validResourceQuantity, validResourceQuantity, validResourceQuantity, validResourceQuantity, ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal(field.NewPath("evictionMinimumReclaim.memoryAvailable").String()),
			})))),
		)

		validDuration := metav1.Duration{Duration: 2 * time.Minute}
		invalidDuration := metav1.Duration{Duration: -2 * time.Minute}
		DescribeTable("validate the kubelet configuration - KubeletConfigEvictionSoftGracePeriod",
			func(memoryAvailable, imagefsAvailable, imagefsInodesFree, nodefsAvailable, nodefsInodesFree metav1.Duration, matcher gomegatypes.GomegaMatcher) {
				kubeletConfig := core.KubeletConfig{
					EvictionSoftGracePeriod: &core.KubeletConfigEvictionSoftGracePeriod{
						MemoryAvailable:   &memoryAvailable,
						ImageFSAvailable:  &imagefsAvailable,
						ImageFSInodesFree: &imagefsInodesFree,
						NodeFSAvailable:   &nodefsAvailable,
						NodeFSInodesFree:  &nodefsInodesFree,
					},
				}

				errList := ValidateKubeletConfig(kubeletConfig, "", true, nil)

				Expect(errList).To(matcher)
			},

			Entry("valid configuration", validDuration, validDuration, validDuration, validDuration, validDuration, HaveLen(0)),
			Entry("only allow positive Duration for any value", invalidDuration, validDuration, validDuration, validDuration, validDuration, ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(field.NewPath("evictionSoftGracePeriod.memoryAvailable").String()),
				})))),
		)

		DescribeTable("validate the kubelet configuration - EvictionPressureTransitionPeriod",
			func(evictionPressureTransitionPeriod metav1.Duration, matcher gomegatypes.GomegaMatcher) {
				kubeletConfig := core.KubeletConfig{
					EvictionPressureTransitionPeriod: &evictionPressureTransitionPeriod,
				}

				errList := ValidateKubeletConfig(kubeletConfig, "", true, nil)

				Expect(errList).To(matcher)
			},

			Entry("valid configuration", validDuration, HaveLen(0)),
			Entry("only allow positive Duration", invalidDuration, ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(field.NewPath("evictionPressureTransitionPeriod").String()),
				})),
			)),
		)

		Context("validate the kubelet configuration - reserved", func() {
			DescribeTable("validate the kubelet configuration - KubeReserved",
				func(cpu, memory, ephemeralStorage, pid *resource.Quantity, matcher gomegatypes.GomegaMatcher) {
					kubeletConfig := core.KubeletConfig{
						KubeReserved: &core.KubeletConfigReserved{
							CPU:              cpu,
							Memory:           memory,
							EphemeralStorage: ephemeralStorage,
							PID:              pid,
						},
					}
					Expect(ValidateKubeletConfig(kubeletConfig, "", true, nil)).To(matcher)
				},

				Entry("valid configuration (cpu)", &validResourceQuantity, nil, nil, nil, HaveLen(0)),
				Entry("valid configuration (memory)", nil, &validResourceQuantity, nil, nil, HaveLen(0)),
				Entry("valid configuration (storage)", nil, nil, &validResourceQuantity, nil, HaveLen(0)),
				Entry("valid configuration (pid)", nil, nil, nil, &validResourceQuantity, HaveLen(0)),
				Entry("valid configuration (all)", &validResourceQuantity, &validResourceQuantity, &validResourceQuantity, &validResourceQuantity, HaveLen(0)),
				Entry("only allow positive resource.Quantity for any value", &invalidResourceQuantity, &validResourceQuantity, &validResourceQuantity, &validResourceQuantity, ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(field.NewPath("kubeReserved.cpu").String()),
				})))),
			)

			DescribeTable("validate the kubelet configuration - SystemReserved",
				func(cpu, memory, ephemeralStorage, pid *resource.Quantity, matcher gomegatypes.GomegaMatcher) {
					kubeletConfig := core.KubeletConfig{
						SystemReserved: &core.KubeletConfigReserved{
							CPU:              cpu,
							Memory:           memory,
							EphemeralStorage: ephemeralStorage,
							PID:              pid,
						},
					}
					Expect(ValidateKubeletConfig(kubeletConfig, "", true, nil)).To(matcher)
				},

				Entry("valid configuration (cpu)", &validResourceQuantity, nil, nil, nil, HaveLen(0)),
				Entry("valid configuration (memory)", nil, &validResourceQuantity, nil, nil, HaveLen(0)),
				Entry("valid configuration (storage)", nil, nil, &validResourceQuantity, nil, HaveLen(0)),
				Entry("valid configuration (pid)", nil, nil, nil, &validResourceQuantity, HaveLen(0)),
				Entry("valid configuration (all)", &validResourceQuantity, &validResourceQuantity, &validResourceQuantity, &validResourceQuantity, HaveLen(0)),
				Entry("only allow positive resource.Quantity for any value", &invalidResourceQuantity, &validResourceQuantity, &validResourceQuantity, &validResourceQuantity, ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(field.NewPath("systemReserved.cpu").String()),
				})))),
			)
		})

		DescribeTable("validate the kubelet configuration - ImagePullProgressDeadline",
			func(imagePullProgressDeadline metav1.Duration, dockerConfigured bool, matcher gomegatypes.GomegaMatcher) {
				kubeletConfig := core.KubeletConfig{
					ImagePullProgressDeadline: &imagePullProgressDeadline,
				}

				errList := ValidateKubeletConfig(kubeletConfig, "", dockerConfigured, nil)

				Expect(errList).To(matcher)
			},

			Entry("valid configuration", validDuration, true, HaveLen(0)),
			Entry("only allow positive Duration", invalidDuration, true, ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(field.NewPath("imagePullProgressDeadline").String()),
				})),
			)),
			Entry("not allowed to be configured when not using docker, as it has no effect on other runtimes", validDuration, false, ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal(field.NewPath("imagePullProgressDeadline").String()),
				})),
			)),
		)

		DescribeTable("validate the kubelet configuration - ImageGCHighThresholdPercent",
			func(imageGCHighThresholdPercent int, matcher gomegatypes.GomegaMatcher) {
				kubeletConfig := core.KubeletConfig{
					ImageGCHighThresholdPercent: pointer.Int32(int32(imageGCHighThresholdPercent)),
				}

				errList := ValidateKubeletConfig(kubeletConfig, "", true, nil)

				Expect(errList).To(matcher)
			},

			Entry("0 <= value <= 100", 1, BeEmpty()),
			Entry("value < 0", -1, ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(field.NewPath("imageGCHighThresholdPercent").String()),
				})),
			)),
			Entry("value > 100", 101, ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(field.NewPath("imageGCHighThresholdPercent").String()),
				})),
			)),
		)

		DescribeTable("validate the kubelet configuration - ImageGCLowThresholdPercent",
			func(imageGCLowThresholdPercent int, matcher gomegatypes.GomegaMatcher) {
				kubeletConfig := core.KubeletConfig{
					ImageGCLowThresholdPercent: pointer.Int32(int32(imageGCLowThresholdPercent)),
				}

				errList := ValidateKubeletConfig(kubeletConfig, "", true, nil)

				Expect(errList).To(matcher)
			},

			Entry("0 <= value <= 100", 1, BeEmpty()),
			Entry("value < 0", -1, ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(field.NewPath("imageGCLowThresholdPercent").String()),
				})),
			)),
			Entry("value > 100", 101, ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(field.NewPath("imageGCLowThresholdPercent").String()),
				})),
			)),
		)

		It("should prevent that imageGCLowThresholdPercent is not less than imageGCHighThresholdPercent", func() {
			kubeletConfig := core.KubeletConfig{
				ImageGCLowThresholdPercent:  pointer.Int32(2),
				ImageGCHighThresholdPercent: pointer.Int32(1),
			}

			errList := ValidateKubeletConfig(kubeletConfig, "", true, nil)

			Expect(errList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal(field.NewPath("imageGCLowThresholdPercent").String()),
				})),
			))
		})

		DescribeTable("validate the kubelet configuration - EvictionMaxPodGracePeriod",
			func(evictionMaxPodGracePeriod int32, matcher gomegatypes.GomegaMatcher) {
				kubeletConfig := core.KubeletConfig{
					EvictionMaxPodGracePeriod: &evictionMaxPodGracePeriod,
				}

				errList := ValidateKubeletConfig(kubeletConfig, "", true, nil)

				Expect(errList).To(matcher)
			},

			Entry("valid configuration", int32(90), HaveLen(0)),
			Entry("only allow positive number", int32(-3), ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(field.NewPath("evictionMaxPodGracePeriod").String()),
				})),
			)),
		)

		DescribeTable("validate the kubelet configuration - MaxPods",
			func(maxPods int32, matcher gomegatypes.GomegaMatcher) {
				kubeletConfig := core.KubeletConfig{
					MaxPods: &maxPods,
				}

				errList := ValidateKubeletConfig(kubeletConfig, "", true, nil)

				Expect(errList).To(matcher)
			},

			Entry("valid configuration", int32(110), HaveLen(0)),
			Entry("only allow positive number", int32(-3), ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(field.NewPath("maxPods").String()),
				})),
			)),
		)
	})

	Describe("#ValidateHibernationSchedules", func() {
		DescribeTable("validate hibernation schedules",
			func(schedules []core.HibernationSchedule, matcher gomegatypes.GomegaMatcher) {
				Expect(ValidateHibernationSchedules(schedules, nil)).To(matcher)
			},
			Entry("valid schedules", []core.HibernationSchedule{{Start: pointer.String("1 * * * *"), End: pointer.String("2 * * * *")}}, BeEmpty()),
			Entry("nil schedules", nil, BeEmpty()),
			Entry("duplicate start and end value in same schedule",
				[]core.HibernationSchedule{{Start: pointer.String("* * * * *"), End: pointer.String("* * * * *")}},
				ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(field.ErrorTypeDuplicate),
				})))),
			Entry("duplicate start and end value in different schedules",
				[]core.HibernationSchedule{{Start: pointer.String("1 * * * *"), End: pointer.String("2 * * * *")}, {Start: pointer.String("1 * * * *"), End: pointer.String("3 * * * *")}},
				ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(field.ErrorTypeDuplicate),
				})))),
			Entry("invalid schedule",
				[]core.HibernationSchedule{{Start: pointer.String("foo"), End: pointer.String("* * * * *")}},
				ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(field.ErrorTypeInvalid),
				})))),
		)
	})

	Describe("#ValidateHibernationCronSpec", func() {
		DescribeTable("validate cron spec",
			func(seenSpecs sets.String, spec string, matcher gomegatypes.GomegaMatcher) {
				Expect(ValidateHibernationCronSpec(seenSpecs, spec, nil)).To(matcher)
			},
			Entry("invalid spec", sets.NewString(), "foo", ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type": Equal(field.ErrorTypeInvalid),
			})))),
			Entry("duplicate spec", sets.NewString("* * * * *"), "* * * * *", ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type": Equal(field.ErrorTypeDuplicate),
			})))),
		)

		It("should add the inspected cron spec to the set if there were no issues", func() {
			var (
				s    = sets.NewString()
				spec = "* * * * *"
			)
			Expect(ValidateHibernationCronSpec(s, spec, nil)).To(BeEmpty())
			Expect(s.Has(spec)).To(BeTrue())
		})

		It("should not add the inspected cron spec to the set if there were issues", func() {
			var (
				s    = sets.NewString()
				spec = "foo"
			)
			Expect(ValidateHibernationCronSpec(s, spec, nil)).NotTo(BeEmpty())
			Expect(s.Has(spec)).To(BeFalse())
		})
	})

	Describe("#ValidateHibernationScheduleLocation", func() {
		DescribeTable("validate hibernation schedule location",
			func(location string, matcher gomegatypes.GomegaMatcher) {
				Expect(ValidateHibernationScheduleLocation(location, nil)).To(matcher)
			},
			Entry("utc location", "UTC", BeEmpty()),
			Entry("empty location -> utc", "", BeEmpty()),
			Entry("invalid location", "should not exist", ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type": Equal(field.ErrorTypeInvalid),
			})))),
		)
	})

	Describe("#ValidateHibernationSchedule", func() {
		DescribeTable("validate schedule",
			func(seenSpecs sets.String, schedule *core.HibernationSchedule, matcher gomegatypes.GomegaMatcher) {
				errList := ValidateHibernationSchedule(seenSpecs, schedule, nil)
				Expect(errList).To(matcher)
			},

			Entry("valid schedule", sets.NewString(), &core.HibernationSchedule{Start: pointer.String("1 * * * *"), End: pointer.String("2 * * * *")}, BeEmpty()),
			Entry("invalid start value", sets.NewString(), &core.HibernationSchedule{Start: pointer.String(""), End: pointer.String("* * * * *")}, ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal(field.NewPath("start").String()),
			})))),
			Entry("invalid end value", sets.NewString(), &core.HibernationSchedule{Start: pointer.String("* * * * *"), End: pointer.String("")}, ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal(field.NewPath("end").String()),
			})))),
			Entry("invalid location", sets.NewString(), &core.HibernationSchedule{Start: pointer.String("1 * * * *"), End: pointer.String("2 * * * *"), Location: pointer.String("foo")}, ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal(field.NewPath("location").String()),
			})))),
			Entry("equal start and end value", sets.NewString(), &core.HibernationSchedule{Start: pointer.String("* * * * *"), End: pointer.String("* * * * *")}, ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeDuplicate),
				"Field": Equal(field.NewPath("end").String()),
			})))),
			Entry("nil start", sets.NewString(), &core.HibernationSchedule{End: pointer.String("* * * * *")}, BeEmpty()),
			Entry("nil end", sets.NewString(), &core.HibernationSchedule{Start: pointer.String("* * * * *")}, BeEmpty()),
			Entry("start and end nil", sets.NewString(), &core.HibernationSchedule{},
				ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type": Equal(field.ErrorTypeRequired),
				})))),
			Entry("invalid start and end value", sets.NewString(), &core.HibernationSchedule{Start: pointer.String(""), End: pointer.String("")},
				ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal(field.NewPath("start").String()),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal(field.NewPath("end").String()),
					})),
				)),
		)
	})
})

func prepareShootForUpdate(shoot *core.Shoot) *core.Shoot {
	s := shoot.DeepCopy()
	s.ResourceVersion = "1"
	return s
}
