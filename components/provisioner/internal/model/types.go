package model

import (
	"time"
)

type OperationState string

//TODO: Remove after schema migration
//      Is it ok to remove it now? Which schema migration exactly?
const AWS = "aws"

const (
	InProgress OperationState = "IN_PROGRESS"
	Succeeded  OperationState = "SUCCEEDED"
	Failed     OperationState = "FAILED"
)

type OperationType string

const (
	Provision            OperationType = "PROVISION"
	ProvisionNoInstall   OperationType = "PROVISION_NO_INSTALL"
	Upgrade              OperationType = "UPGRADE"
	UpgradeShoot         OperationType = "UPGRADE_SHOOT"
	Deprovision          OperationType = "DEPROVISION"
	DeprovisionNoInstall OperationType = "DEPROVISION_NO_INSTALL"
	ReconnectRuntime     OperationType = "RECONNECT_RUNTIME"
	Hibernate            OperationType = "HIBERNATE"
)

type OperationStage string

const (
	WaitingForClusterDomain      OperationStage = "WaitingForClusterDomain"
	WaitingForClusterCreation    OperationStage = "WaitingForClusterCreation"
	CreatingBindingsForOperators OperationStage = "CreatingBindingsForOperators"
	StartingInstallation         OperationStage = "StartingInstallation"
	WaitingForInstallation       OperationStage = "WaitingForInstallation"
	ConnectRuntimeAgent          OperationStage = "ConnectRuntimeAgent"
	WaitForAgentToConnect        OperationStage = "WaitForAgentToConnect"

	TriggerKymaUninstall   OperationStage = "TriggerKymaUninstall"
	WaitForClusterDeletion OperationStage = "WaitForClusterDeletion"
	DeleteCluster          OperationStage = "DeprovisionCluster"
	CleanupCluster         OperationStage = "CleanupCluster"

	StartingUpgrade      OperationStage = "StartingUpgrade"
	UpdatingUpgradeState OperationStage = "UpdatingUpgradeState"

	WaitingForShootUpgrade    OperationStage = "WaitingForShootUpgrade"
	WaitingForShootNewVersion OperationStage = "WaitingForShootNewVersion"

	WaitForHibernation OperationStage = "WaitForHibernation"

	FinishedStage OperationStage = "Finished"
)

type Cluster struct {
	ID                 string
	Kubeconfig         *string
	CreationTimestamp  time.Time
	Deleted            bool
	Tenant             string
	SubAccountId       *string
	ActiveKymaConfigId *string
	Administrators     []string

	ClusterConfig GardenerConfig `db:"-"`
	KymaConfig    *KymaConfig    `db:"-"`
}

type LastError struct {
	ErrMessage string
	Reason     string
	Component  string
}

type Operation struct {
	ID             string
	Type           OperationType
	StartTimestamp time.Time
	EndTimestamp   *time.Time
	State          OperationState
	Message        string
	ClusterID      string
	Stage          OperationStage
	LastTransition *time.Time
	LastError
}

type RuntimeAgentConnectionStatus int

const (
	RuntimeAgentConnectionStatusPending      RuntimeAgentConnectionStatus = iota
	RuntimeAgentConnectionStatusConnected    RuntimeAgentConnectionStatus = iota
	RuntimeAgentConnectionStatusDisconnected RuntimeAgentConnectionStatus = iota
)

type RuntimeStatus struct {
	LastOperationStatus     Operation
	RuntimeConnectionStatus RuntimeAgentConnectionStatus
	RuntimeConfiguration    Cluster
	HibernationStatus       HibernationStatus
}

type OperationsCount struct {
	Count map[OperationType]int
}

type HibernationStatus struct {
	Hibernated          bool
	HibernationPossible bool
}
