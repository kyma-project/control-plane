package monitoring

import (
	"math/rand"
	"net/http"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

const (
	Driver     = "secret"
	MaxHistory = 1
)

type Config struct {
	Namespace       string       `envconfig:"default=kcp-system"`
	ChartUrl        string       `envconfig:"default="`
	RemoteWriteUrl  string       `envconfig:"default="`
	RemoteWritePath string       `envconfig:"default=/api/v1/write"`
	Disabled        bool         `envconfig:"default=true"`
	LocalChart      *chart.Chart `envconfig:"-"`
}

type Parameters struct {
	ReleaseName     string
	InstanceID      string
	GlobalAccountID string
	SubaccountID    string
	ShootName       string
	PlanName        string
	Region          string
	Username        string
	Password        string
}

//go:generate mockery -name=Client
type Client interface {
	IsDeployed(releaseName string) (bool, error)
	IsPresent(releaseName string) (bool, error)
	InstallRelease(params Parameters) (*release.Release, error)
	UpgradeRelease(params Parameters) (*release.Release, error)
	UninstallRelease(releaseName string) (*release.UninstallReleaseResponse, error)
}

type client struct {
	k8sconfig        *rest.Config
	monitoringConfig Config
}

func NewClient(k8sconfig *rest.Config, monitoringConfig Config) (Client, error) {
	if !monitoringConfig.Disabled {
		res, err := http.Get(monitoringConfig.ChartUrl)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		chart, err := loader.LoadArchive(res.Body)
		if err != nil {
			return nil, err
		}
		monitoringConfig.LocalChart = chart
	}
	return &client{
		k8sconfig:        k8sconfig,
		monitoringConfig: monitoringConfig,
	}, nil
}

func (c *client) IsDeployed(releaseName string) (bool, error) {
	cfg, err := c.newActionConfig(c.monitoringConfig.Namespace)
	if err != nil {
		return false, err
	}
	listAction := action.NewList(cfg)
	listAction.Deployed = true

	releases, err := listAction.Run()
	if err != nil {
		return false, err
	}
	for _, rel := range releases {
		if rel.Name == releaseName {
			return true, nil
		}
	}

	return false, nil
}

func (c *client) IsPresent(releaseName string) (bool, error) {
	cfg, err := c.newActionConfig(c.monitoringConfig.Namespace)
	if err != nil {
		return false, err
	}
	listAction := action.NewList(cfg)
	listAction.Deployed = true
	listAction.Superseded = true
	listAction.Uninstalling = true
	listAction.Deployed = true
	listAction.Failed = true
	listAction.Pending = true

	releases, err := listAction.Run()
	if err != nil {
		return false, err
	}
	for _, rel := range releases {
		if rel.Name == releaseName {
			return true, nil
		}
	}

	return false, nil
}

func (c *client) InstallRelease(params Parameters) (*release.Release, error) {
	cfg, err := c.newActionConfig(c.monitoringConfig.Namespace)
	if err != nil {
		return nil, err
	}

	installAction := action.NewInstall(cfg)
	installAction.ReleaseName = params.ReleaseName
	installAction.Namespace = c.monitoringConfig.Namespace
	installAction.Timeout = 6 * time.Minute
	installAction.Wait = true

	overrides := c.generateOverrideMap(params)
	reponse, err := installAction.Run(c.monitoringConfig.LocalChart, overrides)
	if err != nil {
		return nil, err
	}

	return reponse, nil
}

func (c *client) UpgradeRelease(params Parameters) (*release.Release, error) {
	cfg, err := c.newActionConfig(c.monitoringConfig.Namespace)
	if err != nil {
		return nil, err
	}

	upgradeAction := action.NewUpgrade(cfg)
	upgradeAction.Timeout = 6 * time.Minute

	releaseName := params.ReleaseName
	overrides := c.generateOverrideMap(params)

	response, err := upgradeAction.Run(releaseName, c.monitoringConfig.LocalChart, overrides)
	if err != nil {
		return nil, err
	}

	return response, err
}

func (c *client) UninstallRelease(releaseName string) (*release.UninstallReleaseResponse, error) {
	cfg, err := c.newActionConfig(c.monitoringConfig.Namespace)
	if err != nil {
		return nil, err
	}

	uninstallAction := action.NewUninstall(cfg)
	uninstallAction.Timeout = 6 * time.Minute
	response, err := uninstallAction.Run(releaseName)
	if err != nil {
		return nil, err
	}

	return response, err
}

func (c *client) newActionConfig(namespace string) (*action.Configuration, error) {
	config := c.k8sconfig
	kubeConfig := genericclioptions.NewConfigFlags(false)
	kubeConfig.APIServer = &config.Host
	kubeConfig.BearerToken = &config.BearerToken
	kubeConfig.CAFile = &config.CAFile

	cfg := new(action.Configuration)
	if err := cfg.Init(kubeConfig, namespace, Driver, klog.Infof); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func (c *client) generateOverrideMap(params Parameters) map[string]interface{} {
	overrideMap := make(map[string]interface{})
	overrideMap["runtime"] = map[string]string{
		"instanceID":      params.InstanceID,
		"globalAccountID": params.GlobalAccountID,
		"subaccountID":    params.SubaccountID,
		"shootName":       params.ShootName,
		"planName":        params.PlanName,
		"region":          params.Region,
	}
	overrideMap["auth"] = map[string]string{
		"username": params.Username,
		"password": params.Password,
	}
	overrideMap["remoteWrite"] = map[string]string{
		"url":  c.monitoringConfig.RemoteWriteUrl,
		"path": c.monitoringConfig.RemoteWritePath,
	}
	return overrideMap
}

func GeneratePassword(length int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	rand.Seed(time.Now().UnixNano())
	bytes := make([]rune, length)
	for i := range bytes {
		bytes[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	VMPassword := string(bytes)

	return VMPassword
}
