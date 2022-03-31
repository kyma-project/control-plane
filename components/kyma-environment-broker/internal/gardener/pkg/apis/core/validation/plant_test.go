// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/gardener/pkg/apis/core"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	. "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/gardener/pkg/apis/core/validation"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
)

var _ = Describe("validation", func() {
	var plant *core.Plant

	BeforeEach(func() {
		plant = &core.Plant{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-plant",
				Namespace: "test-namespace",
			},
			Spec: core.PlantSpec{
				SecretRef: v1.LocalObjectReference{
					Name: "test",
				},
			},
		}
	})

	Describe("#ValidPlant", func() {
		DescribeTable("Plant metadata",
			func(objectMeta metav1.ObjectMeta, matcher gomegatypes.GomegaMatcher) {
				plant.ObjectMeta = objectMeta

				errorList := ValidatePlant(plant)

				Expect(errorList).To(matcher)
			},

			Entry("should forbid Plant with empty metadata",
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
			Entry("should forbid Plant with empty name",
				metav1.ObjectMeta{Name: "", Namespace: "test-namespace"},
				ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("metadata.name"),
				}))),
			),
			Entry("should forbid Plant with '.' in the name (not a DNS-1123 label compliant name)",
				metav1.ObjectMeta{Name: "plant.test", Namespace: "test-namespace"},
				ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("metadata.name"),
				}))),
			),
			Entry("should forbid Plant with '_' in the name (not a DNS-1123 subdomain)",
				metav1.ObjectMeta{Name: "plant_test", Namespace: "test-namespace"},
				ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("metadata.name"),
				}))),
			),
		)

		It("should forbid empty Plant resources", func() {
			errorList := ValidatePlant(&core.Plant{
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       core.PlantSpec{},
			})

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("metadata.name"),
			})), PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("metadata.namespace"),
			})), PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("spec.secretRef.name"),
			}))))
		})

		It("should allow valid plant resources", func() {
			errorList := ValidatePlant(plant)

			Expect(errorList).To(BeEmpty())
		})
	})

	Describe("#ValidPlantUpdate", func() {
		It("should prevent updating anything if deletion time stamp is set", func() {
			now := metav1.Now()
			plant.DeletionTimestamp = &now

			newPlant := preparePlantForUpdate(plant)
			newPlant.DeletionTimestamp = &now
			newPlant.Spec.SecretRef.Name = "changedName"

			errorList := ValidatePlantUpdate(newPlant, plant)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec"),
			}))))
		})
	})
})

func preparePlantForUpdate(obj *core.Plant) *core.Plant {
	newObj := obj.DeepCopy()
	newObj.ResourceVersion = "1"
	return newObj
}
