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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	. "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/gardener/pkg/apis/core/validation"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
)

var _ = Describe("validation", func() {
	var controllerInstallation *core.ControllerInstallation

	BeforeEach(func() {
		controllerInstallation = &core.ControllerInstallation{
			ObjectMeta: metav1.ObjectMeta{
				Name: "extension-abc",
			},
			Spec: core.ControllerInstallationSpec{
				RegistrationRef: corev1.ObjectReference{
					Name:            "extension",
					ResourceVersion: "1",
				},
				SeedRef: corev1.ObjectReference{
					Name:            "aws",
					ResourceVersion: "1",
				},
			},
		}
	})

	Describe("#ValidateControllerInstallation", func() {
		DescribeTable("ControllerInstallation metadata",
			func(objectMeta metav1.ObjectMeta, matcher gomegatypes.GomegaMatcher) {
				controllerInstallation.ObjectMeta = objectMeta

				errorList := ValidateControllerInstallation(controllerInstallation)

				Expect(errorList).To(matcher)
			},

			Entry("should forbid ControllerInstallation with empty metadata",
				metav1.ObjectMeta{},
				ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("metadata.name"),
				}))),
			),
			Entry("should forbid ControllerInstallation with empty name",
				metav1.ObjectMeta{Name: ""},
				ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("metadata.name"),
				}))),
			),
			Entry("should forbid ControllerInstallation with '.' in the name (not a DNS-1123 label compliant name)",
				metav1.ObjectMeta{Name: "extension-abc.test"},
				ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("metadata.name"),
				}))),
			),
			Entry("should forbid ControllerInstallation with '_' in the name (not a DNS-1123 subdomain)",
				metav1.ObjectMeta{Name: "extension-abc_test"},
				ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("metadata.name"),
				}))),
			),
		)

		It("should forbid empty ControllerInstallation resources", func() {
			errorList := ValidateControllerInstallation(&core.ControllerInstallation{})

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("metadata.name"),
			})), PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("spec.registrationRef.name"),
			})), PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("spec.seedRef.name"),
			}))))
		})

		It("should allow valid ControllerInstallation resources", func() {
			errorList := ValidateControllerInstallation(controllerInstallation)

			Expect(errorList).To(BeEmpty())
		})
	})

	Describe("#ValidateControllerInstallationUpdate", func() {
		It("should prevent updating anything if deletion time stamp is set", func() {
			now := metav1.Now()

			newControllerInstallation := prepareControllerInstallationForUpdate(controllerInstallation)
			controllerInstallation.DeletionTimestamp = &now
			newControllerInstallation.DeletionTimestamp = &now
			newControllerInstallation.Spec.RegistrationRef.APIVersion = "another-api-version"

			errorList := ValidateControllerInstallationUpdate(newControllerInstallation, controllerInstallation)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec"),
			}))))
		})

		It("should prevent updating immutable fields", func() {
			newControllerInstallation := prepareControllerInstallationForUpdate(controllerInstallation)
			newControllerInstallation.Spec.RegistrationRef.Name = "another-name"
			newControllerInstallation.Spec.RegistrationRef.ResourceVersion = "2"
			newControllerInstallation.Spec.SeedRef.Name = "another-name"
			newControllerInstallation.Spec.SeedRef.ResourceVersion = "2"

			errorList := ValidateControllerInstallationUpdate(newControllerInstallation, controllerInstallation)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.registrationRef.name"),
			})), PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.seedRef.name"),
			}))))
		})
	})
})

func prepareControllerInstallationForUpdate(obj *core.ControllerInstallation) *core.ControllerInstallation {
	newObj := obj.DeepCopy()
	newObj.ResourceVersion = "1"
	return newObj
}
