package runtimeversion

import (
	"strconv"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/pkg/errors"
)

type RuntimeVersionConfigurator struct {
	defaultVersion        string
	defaultPreviewVersion string
	accountMapping        *AccountVersionMapping
}

func NewRuntimeVersionConfigurator(defaultVersion string, previewVersion string, accountMapping *AccountVersionMapping) *RuntimeVersionConfigurator {
	return &RuntimeVersionConfigurator{
		defaultVersion:        defaultVersion,
		defaultPreviewVersion: previewVersion,
		accountMapping:        accountMapping,
	}
}

func (rvc *RuntimeVersionConfigurator) ForProvisioning(op internal.ProvisioningOperation) (*internal.RuntimeVersionData, error) {

	pp := op.ProvisioningParameters

	if broker.IsPreviewPlan(pp.PlanID) && rvc.defaultPreviewVersion != "" {

		if pp.Parameters.KymaVersion != "" {
			majorVer, err := determineMajorVersion(pp.Parameters.KymaVersion, rvc.defaultPreviewVersion)
			if err != nil {
				return nil, errors.Wrap(err, "while determining Kyma's major version")
			}
			return internal.NewRuntimeVersionFromParameters(pp.Parameters.KymaVersion, majorVer), nil
		}

		_, found, err := rvc.accountMapping.Get(pp.ErsContext.GlobalAccountID, pp.ErsContext.SubAccountID)
		if err != nil {
			return nil, err
		}
		if found {
			majorVer, err := determineMajorVersion(rvc.defaultPreviewVersion, rvc.defaultPreviewVersion)
			if err != nil {
				return nil, errors.Wrap(err, "while determining Kyma's major version")
			}
			return internal.NewRuntimeVersionFromAccountMapping(rvc.defaultPreviewVersion, majorVer), nil
		}

		return internal.NewRuntimeVersionFromDefaults(rvc.defaultPreviewVersion), nil
	}

	if pp.Parameters.KymaVersion == "" {
		version, found, err := rvc.accountMapping.Get(pp.ErsContext.GlobalAccountID, pp.ErsContext.SubAccountID)
		if err != nil {
			return nil, err
		}
		if found {
			majorVer, err := determineMajorVersion(version, rvc.defaultVersion)
			if err != nil {
				return nil, errors.Wrap(err, "while determining Kyma's major version")
			}
			return internal.NewRuntimeVersionFromAccountMapping(version, majorVer), nil
		}
		return internal.NewRuntimeVersionFromDefaults(rvc.defaultVersion), nil
	}
	majorVer, err := determineMajorVersion(pp.Parameters.KymaVersion, rvc.defaultVersion)
	if err != nil {
		return nil, errors.Wrap(err, "while determining Kyma's major version")
	}
	return internal.NewRuntimeVersionFromParameters(pp.Parameters.KymaVersion, majorVer), nil
}

func determineMajorVersion(version, defaultVersion string) (int, error) {
	if isCustomVersion(version) {
		return extractMajorVersionNumberFromVersionString(defaultVersion)
	}
	return extractMajorVersionNumberFromVersionString(version)
}

func isCustomVersion(version string) bool {
	return strings.HasPrefix(version, "PR-") || strings.HasPrefix(version, "main-")
}

func extractMajorVersionNumberFromVersionString(version string) (int, error) {
	splitVer := strings.Split(version, ".")
	majorVerNum, err := strconv.Atoi(splitVer[0])
	if err != nil {
		return 0, errors.New("cannot convert major version to int")
	}
	return majorVerNum, nil
}

func (rvc *RuntimeVersionConfigurator) ForUpgrade(op internal.UpgradeKymaOperation) (*internal.RuntimeVersionData, error) {
	version, found, err := rvc.accountMapping.Get(op.GlobalAccountID, op.RuntimeOperation.SubAccountID)
	if err != nil {
		return nil, err
	}
	if found {
		majorVer, err := determineMajorVersion(version, rvc.defaultPreviewVersion)
		if err != nil {
			return nil, errors.Wrap(err, "while determining Kyma's major version")
		}
		return internal.NewRuntimeVersionFromAccountMapping(version, majorVer), nil
	}

	return internal.NewRuntimeVersionFromDefaults(rvc.defaultVersion), nil
}
