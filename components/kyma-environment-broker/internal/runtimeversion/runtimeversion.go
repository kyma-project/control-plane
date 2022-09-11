package runtimeversion

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pkg/errors"
)

type RuntimeVersionConfigurator struct {
	defaultVersion string
	accountMapping *AccountVersionMapping
	runtimeStateDB storage.RuntimeStates
}

func NewRuntimeVersionConfigurator(defaultVersion string, accountMapping *AccountVersionMapping, runtimeStates storage.RuntimeStates) *RuntimeVersionConfigurator {
	if defaultVersion == "" {
		panic("Default version not provided")
	}

	return &RuntimeVersionConfigurator{
		defaultVersion: defaultVersion,
		accountMapping: accountMapping,
		runtimeStateDB: runtimeStates,
	}
}

func (rvc *RuntimeVersionConfigurator) ForUpdating(op internal.UpdatingOperation) (*internal.RuntimeVersionData, error) {
	r, err := rvc.runtimeStateDB.GetLatestWithKymaVersionByRuntimeID(op.RuntimeID)
	if err != nil {
		return nil, err
	}

	return internal.NewRuntimeVersionFromDefaults(r.GetKymaVersion()), nil
}

func (rvc *RuntimeVersionConfigurator) ForProvisioning(op internal.Operation) (*internal.RuntimeVersionData, error) {

	pp := op.ProvisioningParameters

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
		return 0, fmt.Errorf("cannot convert major version (version: \"%s\") to int", version)
	}
	return majorVerNum, nil
}

func (rvc *RuntimeVersionConfigurator) ForUpgrade(op internal.UpgradeKymaOperation) (*internal.RuntimeVersionData, error) {
	version, found, err := rvc.accountMapping.Get(op.GlobalAccountID, op.RuntimeOperation.SubAccountID)
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
