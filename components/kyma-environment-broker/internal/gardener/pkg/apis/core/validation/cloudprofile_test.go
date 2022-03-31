// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/gardener/pkg/apis/core"
	. "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/gardener/pkg/apis/core/validation"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"
)

var (
	metadata = metav1.ObjectMeta{
		Name: "profile",
	}
	machineType = core.MachineType{
		Name:   "machine-type-1",
		CPU:    resource.MustParse("2"),
		GPU:    resource.MustParse("0"),
		Memory: resource.MustParse("100Gi"),
	}
	machineTypesConstraint = []core.MachineType{
		machineType,
	}
	volumeTypesConstraint = []core.VolumeType{
		{
			Name:  "volume-type-1",
			Class: "super-premium",
		},
	}

	negativeQuantity = resource.MustParse("-1")
	validQuantity    = resource.MustParse("100Gi")

	invalidMachineType = core.MachineType{
		Name:   "",
		CPU:    negativeQuantity,
		GPU:    negativeQuantity,
		Memory: resource.MustParse("-100Gi"),
		Storage: &core.MachineTypeStorage{
			MinSize: &negativeQuantity,
		},
	}
	invalidMachineType2 = core.MachineType{
		Name:   "negative-storage-size",
		CPU:    resource.MustParse("2"),
		GPU:    resource.MustParse("0"),
		Memory: resource.MustParse("100Gi"),
		Storage: &core.MachineTypeStorage{
			StorageSize: &negativeQuantity,
		},
	}
	invalidMachineType3 = core.MachineType{
		Name:   "min-size-and-storage-size",
		CPU:    resource.MustParse("2"),
		GPU:    resource.MustParse("0"),
		Memory: resource.MustParse("100Gi"),
		Storage: &core.MachineTypeStorage{
			MinSize:     &validQuantity,
			StorageSize: &validQuantity,
		},
	}
	invalidMachineType4 = core.MachineType{
		Name:    "empty-storage-config",
		CPU:     resource.MustParse("2"),
		GPU:     resource.MustParse("0"),
		Memory:  resource.MustParse("100Gi"),
		Storage: &core.MachineTypeStorage{},
	}
	invalidMachineTypes = []core.MachineType{
		invalidMachineType,
		invalidMachineType2,
		invalidMachineType3,
		invalidMachineType4,
	}
	invalidVolumeTypes = []core.VolumeType{
		{
			Name:    "",
			Class:   "",
			MinSize: &negativeQuantity,
		},
	}

	regionName = "region1"
	zoneName   = "zone1"

	supportedClassification  = core.ClassificationSupported
	previewClassification    = core.ClassificationPreview
	deprecatedClassification = core.ClassificationDeprecated
)

var _ = Describe("CloudProfile Validation Tests ", func() {
	Describe("#ValidateCloudProfile", func() {
		It("should forbid empty CloudProfile resources", func() {
			cloudProfile := &core.CloudProfile{
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       core.CloudProfileSpec{},
			}

			errorList := ValidateCloudProfile(cloudProfile)

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("metadata.name"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.type"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.kubernetes.versions"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.kubernetes.versions"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.machineImages"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.machineTypes"),
				})), PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.regions"),
				}))))
		})

		Context("tests for unknown cloud profiles", func() {
			var (
				cloudProfile *core.CloudProfile

				duplicatedKubernetes = core.KubernetesSettings{
					Versions: []core.ExpirableVersion{{Version: "1.11.4"}, {Version: "1.11.4", Classification: &previewClassification}},
				}
				duplicatedRegions = []core.Region{
					{
						Name: regionName,
						Zones: []core.AvailabilityZone{
							{Name: zoneName},
						},
					},
					{
						Name: regionName,
						Zones: []core.AvailabilityZone{
							{Name: zoneName},
						},
					},
				}
				duplicatedZones = []core.Region{
					{
						Name: regionName,
						Zones: []core.AvailabilityZone{
							{Name: zoneName},
							{Name: zoneName},
						},
					},
				}
			)

			BeforeEach(func() {
				cloudProfile = &core.CloudProfile{
					ObjectMeta: metadata,
					Spec: core.CloudProfileSpec{
						Type: "unknown",
						SeedSelector: &core.SeedSelector{
							LabelSelector: metav1.LabelSelector{
								MatchLabels: map[string]string{"foo": "bar"},
							},
						},
						Kubernetes: core.KubernetesSettings{
							Versions: []core.ExpirableVersion{{
								Version: "1.11.4",
							}},
						},
						MachineImages: []core.MachineImage{
							{
								Name: "some-machineimage",
								Versions: []core.MachineImageVersion{
									{
										ExpirableVersion: core.ExpirableVersion{
											Version: "1.2.3",
										},
									},
								},
							},
						},
						Regions: []core.Region{
							{
								Name: regionName,
								Zones: []core.AvailabilityZone{
									{Name: zoneName},
								},
							},
						},
						MachineTypes: machineTypesConstraint,
						VolumeTypes:  volumeTypesConstraint,
					},
				}
			})

			It("should not return any errors", func() {
				errorList := ValidateCloudProfile(cloudProfile)

				Expect(errorList).To(BeEmpty())
			})

			DescribeTable("CloudProfile metadata",
				func(objectMeta metav1.ObjectMeta, matcher gomegatypes.GomegaMatcher) {
					cloudProfile.ObjectMeta = objectMeta

					errorList := ValidateCloudProfile(cloudProfile)

					Expect(errorList).To(matcher)
				},

				Entry("should forbid CloudProfile with empty metadata",
					metav1.ObjectMeta{},
					ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("metadata.name"),
					}))),
				),
				Entry("should forbid CloudProfile with empty name",
					metav1.ObjectMeta{Name: ""},
					ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("metadata.name"),
					}))),
				),
				Entry("should allow CloudProfile with '.' in the name",
					metav1.ObjectMeta{Name: "profile.test"},
					BeEmpty(),
				),
				Entry("should forbid CloudProfile with '_' in the name (not a DNS-1123 subdomain)",
					metav1.ObjectMeta{Name: "profile_test"},
					ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("metadata.name"),
					}))),
				),
			)

			It("should forbid not specifying a type", func() {
				cloudProfile.Spec.Type = ""

				errorList := ValidateCloudProfile(cloudProfile)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.type"),
				}))))
			})

			It("should forbid ca bundles with unsupported format", func() {
				cloudProfile.Spec.CABundle = pointer.String("unsupported")

				errorList := ValidateCloudProfile(cloudProfile)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.caBundle"),
				}))))
			})

			Context("kubernetes version constraints", func() {
				It("should enforce that at least one version has been defined", func() {
					cloudProfile.Spec.Kubernetes.Versions = []core.ExpirableVersion{}

					errorList := ValidateCloudProfile(cloudProfile)

					Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("spec.kubernetes.versions"),
					})), PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.kubernetes.versions"),
					}))))
				})

				It("should forbid versions of a not allowed pattern", func() {
					cloudProfile.Spec.Kubernetes.Versions = []core.ExpirableVersion{{Version: "1.11"}}

					errorList := ValidateCloudProfile(cloudProfile)

					Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.kubernetes.versions[0]"),
					}))))
				})

				It("should forbid expiration date on latest kubernetes version", func() {
					expirationDate := &metav1.Time{Time: time.Now().AddDate(0, 0, 1)}
					cloudProfile.Spec.Kubernetes.Versions = []core.ExpirableVersion{
						{
							Version:        "1.1.0",
							Classification: &supportedClassification,
						},
						{
							Version:        "1.2.0",
							Classification: &deprecatedClassification,
							ExpirationDate: expirationDate,
						},
					}

					errorList := ValidateCloudProfile(cloudProfile)

					Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.kubernetes.versions[].expirationDate"),
					}))))
				})

				It("should forbid duplicated kubernetes versions", func() {
					cloudProfile.Spec.Kubernetes = duplicatedKubernetes

					errorList := ValidateCloudProfile(cloudProfile)

					Expect(errorList).To(ConsistOf(
						PointTo(MatchFields(IgnoreExtras, Fields{
							"Type":  Equal(field.ErrorTypeDuplicate),
							"Field": Equal(fmt.Sprintf("spec.kubernetes.versions[%d].version", len(duplicatedKubernetes.Versions)-1)),
						}))))
				})

				It("should forbid invalid classification for kubernetes versions", func() {
					classification := core.VersionClassification("dummy")
					cloudProfile.Spec.Kubernetes.Versions = []core.ExpirableVersion{
						{
							Version:        "1.1.0",
							Classification: &classification,
						},
					}

					errorList := ValidateCloudProfile(cloudProfile)

					Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":     Equal(field.ErrorTypeNotSupported),
						"Field":    Equal("spec.kubernetes.versions[0].classification"),
						"BadValue": Equal(classification),
					}))))
				})

				It("only allow one supported version per minor version", func() {
					cloudProfile.Spec.Kubernetes.Versions = []core.ExpirableVersion{
						{
							Version:        "1.1.0",
							Classification: &supportedClassification,
						},
						{
							Version:        "1.1.1",
							Classification: &supportedClassification,
						},
					}
					errorList := ValidateCloudProfile(cloudProfile)

					Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeForbidden),
						"Field": Equal("spec.kubernetes.versions[1]"),
					})), PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeForbidden),
						"Field": Equal("spec.kubernetes.versions[0]"),
					}))))
				})
			})

			Context("machine image validation", func() {
				It("should forbid an empty list of machine images", func() {
					cloudProfile.Spec.MachineImages = []core.MachineImage{}

					errorList := ValidateCloudProfile(cloudProfile)

					Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("spec.machineImages"),
					}))))
				})

				It("should forbid duplicate names in list of machine images", func() {
					cloudProfile.Spec.MachineImages = []core.MachineImage{
						{
							Name: "some-machineimage",
							Versions: []core.MachineImageVersion{
								{
									ExpirableVersion: core.ExpirableVersion{
										Version:        "3.4.6",
										Classification: &supportedClassification,
									},
								},
							},
						},
						{
							Name: "some-machineimage",
							Versions: []core.MachineImageVersion{
								{
									ExpirableVersion: core.ExpirableVersion{
										Version:        "3.4.5",
										Classification: &previewClassification,
									},
								},
							},
						},
					}

					errorList := ValidateCloudProfile(cloudProfile)

					Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeDuplicate),
						"Field": Equal("spec.machineImages[1]"),
					}))))
				})

				It("should forbid machine images with no version", func() {
					cloudProfile.Spec.MachineImages = []core.MachineImage{
						{Name: "some-machineimage"},
					}

					errorList := ValidateCloudProfile(cloudProfile)

					Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("spec.machineImages[0].versions"),
					})), PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.machineImages"),
					}))))
				})

				It("should forbid nonSemVer machine image versions", func() {
					cloudProfile.Spec.MachineImages = []core.MachineImage{
						{
							Name: "some-machineimage",
							Versions: []core.MachineImageVersion{
								{
									ExpirableVersion: core.ExpirableVersion{
										Version:        "0.1.2",
										Classification: &supportedClassification,
									},
								},
							},
						},
						{
							Name: "xy",
							Versions: []core.MachineImageVersion{
								{
									ExpirableVersion: core.ExpirableVersion{
										Version:        "a.b.c",
										Classification: &supportedClassification,
									},
								},
							},
						},
					}

					errorList := ValidateCloudProfile(cloudProfile)

					Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.machineImages"),
					})), PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.machineImages[1].versions[0].version"),
					}))))
				})

				It("should allow expiration date on latest machine image version", func() {
					expirationDate := &metav1.Time{Time: time.Now().AddDate(0, 0, 1)}
					cloudProfile.Spec.MachineImages = []core.MachineImage{
						{
							Name: "some-machineimage",
							Versions: []core.MachineImageVersion{
								{
									ExpirableVersion: core.ExpirableVersion{
										Version:        "0.1.2",
										ExpirationDate: expirationDate,
										Classification: &previewClassification,
									},
								},
								{
									ExpirableVersion: core.ExpirableVersion{
										Version:        "0.1.1",
										Classification: &supportedClassification,
									},
								},
							},
						},
						{
							Name: "xy",
							Versions: []core.MachineImageVersion{
								{
									ExpirableVersion: core.ExpirableVersion{
										Version:        "0.1.1",
										ExpirationDate: expirationDate,
										Classification: &supportedClassification,
									},
								},
							},
						},
					}

					errorList := ValidateCloudProfile(cloudProfile)
					Expect(errorList).To(BeEmpty())
				})

				It("should forbid invalid classification for machine image versions", func() {
					classification := core.VersionClassification("dummy")
					cloudProfile.Spec.MachineImages = []core.MachineImage{
						{
							Name: "some-machineimage",
							Versions: []core.MachineImageVersion{
								{
									ExpirableVersion: core.ExpirableVersion{
										Version:        "0.1.2",
										Classification: &classification,
									},
								},
							},
						},
					}

					errorList := ValidateCloudProfile(cloudProfile)
					Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":     Equal(field.ErrorTypeNotSupported),
						"Field":    Equal("spec.machineImages[0].versions[0].classification"),
						"BadValue": Equal(classification),
					}))))
				})
			})

			It("should forbid non-supported container runtime interface names", func() {
				cloudProfile.Spec.MachineImages = []core.MachineImage{
					{
						Name: "invalid-cri-name",
						Versions: []core.MachineImageVersion{
							{
								ExpirableVersion: core.ExpirableVersion{
									Version: "0.1.2",
								},
								CRI: []core.CRI{
									{
										Name: "invalid-cri-name",
									},
								},
							},
						},
					},
					{
						Name: "valid-cri-name",
						Versions: []core.MachineImageVersion{
							{
								ExpirableVersion: core.ExpirableVersion{
									Version: "0.1.2",
								},
								CRI: []core.CRI{
									{
										Name: core.CRINameContainerD,
									},
								},
							},
						},
					},
				}

				errorList := ValidateCloudProfile(cloudProfile)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeNotSupported),
					"Field": Equal("spec.machineImages[0].versions[0].cri[0]"),
				}))))
			})

			It("should forbid duplicated container runtime interface names", func() {
				cloudProfile.Spec.MachineImages[0].Versions[0].CRI = []core.CRI{
					{
						Name: core.CRINameContainerD,
					},
					{
						Name: core.CRINameContainerD,
					},
				}

				errorList := ValidateCloudProfile(cloudProfile)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeDuplicate),
					"Field": Equal("spec.machineImages[0].versions[0].cri[1]"),
				}))))
			})

			It("should forbid duplicated container runtime names", func() {
				cloudProfile.Spec.MachineImages[0].Versions[0].CRI = []core.CRI{
					{
						Name: core.CRINameContainerD,
						ContainerRuntimes: []core.ContainerRuntime{
							{
								Type: "cr1",
							},
							{
								Type: "cr1",
							},
						},
					},
				}

				errorList := ValidateCloudProfile(cloudProfile)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeDuplicate),
					"Field": Equal("spec.machineImages[0].versions[0].cri[0].containerRuntimes[1].type"),
				}))))
			})

			Context("machine types validation", func() {
				It("should enforce that at least one machine type has been defined", func() {
					cloudProfile.Spec.MachineTypes = []core.MachineType{}

					errorList := ValidateCloudProfile(cloudProfile)

					Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("spec.machineTypes"),
					}))))
				})

				It("should enforce uniqueness of machine type names", func() {
					cloudProfile.Spec.MachineTypes = []core.MachineType{
						cloudProfile.Spec.MachineTypes[0],
						cloudProfile.Spec.MachineTypes[0],
					}

					errorList := ValidateCloudProfile(cloudProfile)

					Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeDuplicate),
						"Field": Equal("spec.machineTypes[1].name"),
					}))))
				})

				It("should forbid machine types with unsupported property values", func() {
					cloudProfile.Spec.MachineTypes = invalidMachineTypes

					errorList := ValidateCloudProfile(cloudProfile)

					Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("spec.machineTypes[0].name"),
					})), PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.machineTypes[0].cpu"),
					})), PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.machineTypes[0].gpu"),
					})), PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.machineTypes[0].memory"),
					})), PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.machineTypes[0].storage.minSize"),
					})), PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.machineTypes[1].storage.size"),
					})), PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.machineTypes[2].storage"),
					})), PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.machineTypes[3].storage"),
					})),
					))
				})
			})

			Context("regions validation", func() {
				It("should forbid regions with unsupported name values", func() {
					cloudProfile.Spec.Regions = []core.Region{
						{
							Name:  "",
							Zones: []core.AvailabilityZone{{Name: ""}},
						},
					}

					errorList := ValidateCloudProfile(cloudProfile)

					Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("spec.regions[0].name"),
					})), PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("spec.regions[0].zones[0].name"),
					}))))
				})

				It("should forbid duplicated region names", func() {
					cloudProfile.Spec.Regions = duplicatedRegions

					errorList := ValidateCloudProfile(cloudProfile)

					Expect(errorList).To(ConsistOf(
						PointTo(MatchFields(IgnoreExtras, Fields{
							"Type":  Equal(field.ErrorTypeDuplicate),
							"Field": Equal(fmt.Sprintf("spec.regions[%d].name", len(duplicatedRegions)-1)),
						}))))
				})

				It("should forbid duplicated zone names", func() {
					cloudProfile.Spec.Regions = duplicatedZones

					errorList := ValidateCloudProfile(cloudProfile)

					Expect(errorList).To(ConsistOf(
						PointTo(MatchFields(IgnoreExtras, Fields{
							"Type":  Equal(field.ErrorTypeDuplicate),
							"Field": Equal(fmt.Sprintf("spec.regions[0].zones[%d].name", len(duplicatedZones[0].Zones)-1)),
						}))))
				})

				It("should forbid invalid label specifications", func() {
					cloudProfile.Spec.Regions[0].Labels = map[string]string{
						"this-is-not-allowed": "?*&!@",
						"toolongtoolongtoolongtoolongtoolongtoolongtoolongtoolongtoolongtoolongtoolong": "toolongtoolongtoolongtoolongtoolongtoolongtoolongtoolongtoolongtoolongtoolong",
					}

					errorList := ValidateCloudProfile(cloudProfile)

					Expect(errorList).To(ConsistOf(
						PointTo(MatchFields(IgnoreExtras, Fields{
							"Type":  Equal(field.ErrorTypeInvalid),
							"Field": Equal("spec.regions[0].labels"),
						})),
						PointTo(MatchFields(IgnoreExtras, Fields{
							"Type":  Equal(field.ErrorTypeInvalid),
							"Field": Equal("spec.regions[0].labels"),
						})),
						PointTo(MatchFields(IgnoreExtras, Fields{
							"Type":  Equal(field.ErrorTypeInvalid),
							"Field": Equal("spec.regions[0].labels"),
						})),
					))
				})
			})

			Context("volume types validation", func() {
				It("should enforce uniqueness of volume type names", func() {
					cloudProfile.Spec.VolumeTypes = []core.VolumeType{
						cloudProfile.Spec.VolumeTypes[0],
						cloudProfile.Spec.VolumeTypes[0],
					}

					errorList := ValidateCloudProfile(cloudProfile)

					Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeDuplicate),
						"Field": Equal("spec.volumeTypes[1].name"),
					}))))
				})

				It("should forbid volume types with unsupported property values", func() {
					cloudProfile.Spec.VolumeTypes = invalidVolumeTypes

					errorList := ValidateCloudProfile(cloudProfile)

					Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("spec.volumeTypes[0].name"),
					})), PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("spec.volumeTypes[0].class"),
					})), PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("spec.volumeTypes[0].minSize"),
					})),
					))
				})
			})

			It("should forbid unsupported seed selectors", func() {
				cloudProfile.Spec.SeedSelector.MatchLabels["foo"] = "no/slash/allowed"

				errorList := ValidateCloudProfile(cloudProfile)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.seedSelector.matchLabels"),
				}))))
			})
		})
	})

	var (
		cloudProfileNew *core.CloudProfile
		cloudProfileOld *core.CloudProfile
		dateInThePast   = &metav1.Time{Time: time.Now().AddDate(-5, 0, 0)}
	)

	Describe("#ValidateCloudProfileSpecUpdate", func() {
		BeforeEach(func() {
			cloudProfileNew = &core.CloudProfile{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "dummy",
					Name:            "dummy",
				},
				Spec: core.CloudProfileSpec{
					Type: "aws",
					MachineImages: []core.MachineImage{
						{
							Name: "some-machineimage",
							Versions: []core.MachineImageVersion{
								{
									ExpirableVersion: core.ExpirableVersion{
										Version:        "1.2.3",
										Classification: &supportedClassification,
									},
								},
							},
						},
					},
					Kubernetes: core.KubernetesSettings{
						Versions: []core.ExpirableVersion{
							{
								Version:        "1.17.2",
								Classification: &deprecatedClassification,
							},
						},
					},
					Regions: []core.Region{
						{
							Name: regionName,
							Zones: []core.AvailabilityZone{
								{Name: zoneName},
							},
						},
					},
					MachineTypes: machineTypesConstraint,
					VolumeTypes:  volumeTypesConstraint,
				},
			}
			cloudProfileOld = cloudProfileNew.DeepCopy()
		})

		Context("Removed Kubernetes versions", func() {
			It("deleting version - should not return any errors", func() {
				versions := []core.ExpirableVersion{
					{Version: "1.17.2", Classification: &deprecatedClassification},
					{Version: "1.17.1", Classification: &deprecatedClassification, ExpirationDate: dateInThePast},
					{Version: "1.17.0", Classification: &deprecatedClassification, ExpirationDate: dateInThePast},
				}
				cloudProfileNew.Spec.Kubernetes.Versions = versions[0:1]
				cloudProfileOld.Spec.Kubernetes.Versions = versions
				errorList := ValidateCloudProfileUpdate(cloudProfileNew, cloudProfileOld)

				Expect(errorList).To(BeEmpty())
			})
		})

		Context("Removed MachineImage versions", func() {
			It("deleting version - should not return any errors", func() {
				versions := []core.MachineImageVersion{
					{
						ExpirableVersion: core.ExpirableVersion{
							Version:        "2135.6.2",
							Classification: &deprecatedClassification,
						},
					},
					{
						ExpirableVersion: core.ExpirableVersion{
							Version:        "2135.6.1",
							Classification: &deprecatedClassification,
							ExpirationDate: dateInThePast,
						},
					},
					{
						ExpirableVersion: core.ExpirableVersion{
							Version:        "2135.6.0",
							Classification: &deprecatedClassification,
							ExpirationDate: dateInThePast,
						},
					},
				}
				cloudProfileNew.Spec.MachineImages[0].Versions = versions[0:1]
				cloudProfileOld.Spec.MachineImages[0].Versions = versions
				errorList := ValidateCloudProfileUpdate(cloudProfileNew, cloudProfileOld)

				Expect(errorList).To(BeEmpty())
			})
		})
	})

	Context("#ValidateCloudProfileAddedVersions - Update adding versions to CloudProfile (Kubernetes/MachineImage versions)", func() {
		var versions []core.ExpirableVersion
		BeforeEach(func() {
			versions = []core.ExpirableVersion{
				{Version: "2135.6.3"},
				{Version: "2135.6.2"},
				{Version: "2135.6.1"},
				{Version: "2135.6.0"},
			}
		})

		It("Allow adding versions", func() {
			addedVersions := map[string]int{
				versions[0].Version: 0,
			}
			errorList := ValidateCloudProfileAddedVersions(versions, addedVersions, field.NewPath("spec.kubernetes.versions"))

			Expect(errorList).To(BeEmpty())
		})

		It("Do not allow adding versions with expiration date in the past", func() {
			addedVersions := map[string]int{
				versions[0].Version: 0,
				versions[2].Version: 2,
			}
			versions[2].ExpirationDate = dateInThePast
			errorList := ValidateCloudProfileAddedVersions(versions, addedVersions, field.NewPath("spec.kubernetes.versions"))

			Expect(errorList).To(HaveLen(1))
			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeForbidden),
				"Field":  Equal("spec.kubernetes.versions[2]"),
				"Detail": ContainSubstring("a version with expiration date in the past is not allowed"),
			}))))
		})
	})

	Describe("#ValidateCloudProfileCreation", func() {
		var cloudProfile *core.CloudProfile
		BeforeEach(func() {
			cloudProfile = &core.CloudProfile{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "dummy",
					Name:            "dummy",
				},
				Spec: core.CloudProfileSpec{
					Type: "aws",
					MachineImages: []core.MachineImage{
						{
							Name: "some-machineimage",
							Versions: []core.MachineImageVersion{
								{
									ExpirableVersion: core.ExpirableVersion{
										Version: "1.2.3",
									},
								},
							},
						},
					},
					Kubernetes: core.KubernetesSettings{Versions: []core.ExpirableVersion{
						{Version: "1.17.2"}}},
					Regions: []core.Region{
						{
							Name: regionName,
							Zones: []core.AvailabilityZone{
								{Name: zoneName},
							},
						},
					},
					MachineTypes: machineTypesConstraint,
					VolumeTypes:  volumeTypesConstraint,
				},
			}
		})

		Context("Kubernetes versions", func() {
			It("it should return an error when creating a CloudProfile having a version with expiration date in the past", func() {
				versions := []core.ExpirableVersion{
					{Version: "1.17.2"},
					{Version: "1.17.1", ExpirationDate: dateInThePast},
				}
				cloudProfile.Spec.Kubernetes.Versions = versions
				errorList := ValidateCloudProfileCreation(cloudProfile)

				Expect(errorList).To(HaveLen(1))
				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeForbidden),
					"Field":  Equal("spec.kubernetes.versions[1]"),
					"Detail": ContainSubstring("a version with expiration date in the past is not allowed"),
				}))))
			})
		})

		Context("Machine image versions", func() {
			It("it should return an error when creating a CloudProfile having a Machine image version with expiration date in the past", func() {
				versions := []core.MachineImageVersion{
					{
						ExpirableVersion: core.ExpirableVersion{
							Version: "1.17.2",
						},
					},
					{
						ExpirableVersion: core.ExpirableVersion{
							Version:        "1.17.1",
							ExpirationDate: dateInThePast,
						},
					},
				}
				cloudProfile.Spec.MachineImages[0].Versions = versions
				errorList := ValidateCloudProfileCreation(cloudProfile)

				Expect(errorList).To(HaveLen(1))
				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeForbidden),
					"Field":  Equal("spec.machineImages[0].versions[1]"),
					"Detail": ContainSubstring("a version with expiration date in the past is not allowed"),
				}))))
			})
		})
	})
})
