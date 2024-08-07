package helm

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ClusterOperator/ClusterOperator/pkg/db"
	"github.com/ClusterOperator/ClusterOperator/pkg/model"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/encrypt"

	"github.com/ClusterOperator/ClusterOperator/pkg/constant"
	"github.com/ClusterOperator/ClusterOperator/pkg/logger"
	"github.com/ClusterOperator/ClusterOperator/pkg/repository"
	clusterUtil "github.com/ClusterOperator/ClusterOperator/pkg/util/cluster"
	"github.com/ghodss/yaml"
	"github.com/gofrs/flock"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

const (
	helmDriver = "configmap"
)

func nolog(format string, v ...interface{}) {}

type Interface interface {
	Install(name string, chartName string, chartVersion string, values map[string]interface{}) (*release.Release, error)
	Upgrade(name string, chartName string, chartVersion string, values map[string]interface{}) (*release.Release, error)
	Uninstall(name string) (*release.UninstallReleaseResponse, error)
	List() ([]*release.Release, error)
	GetRepoIP(arch string) (string, string, int, int, error)
	SyncRepoCharts(arch string) error
}

type Config struct {
	OldNamespace  string
	Namespace     string
	Architectures string

	AuthenticationMode string
	Host               string
	BearerToken        string
	CertDataStr        string
	KeyDataStr         string
	ConfigContent      string
}
type Client struct {
	installActionConfig   *action.Configuration
	unInstallActionConfig *action.Configuration
	Namespace             string
	settings              *cli.EnvSettings
	Architectures         string
}

func NewClient(config *Config) (*Client, error) {
	client := Client{
		Architectures: config.Architectures,
	}
	client.settings = GetSettings()
	cf := genericclioptions.NewConfigFlags(true)
	inscure := true

	switch config.AuthenticationMode {
	case constant.AuthenticationModeBearer:
		apiServer := fmt.Sprintf("https://%s", config.Host)
		cf.APIServer = &apiServer
		cf.BearerToken = &config.BearerToken
	case constant.AuthenticationModeCertificate:
		cf.CertFile = &config.CertDataStr
		cf.KeyFile = &config.KeyDataStr
	case constant.AuthenticationModeConfigFile:
		apiConfig, err := clusterUtil.PauseConfigApi(&config.ConfigContent)
		if err != nil {
			return nil, err
		}
		getter := func() (*api.Config, error) {
			return apiConfig, nil
		}
		itemConfig, err := clientcmd.BuildConfigFromKubeconfigGetter("", getter)
		if err != nil {
			return nil, err
		}
		cf.WrapConfigFn = func(config *rest.Config) *rest.Config {
			return itemConfig
		}
	}

	cf.Insecure = &inscure
	if config.Namespace == "" {
		client.Namespace = constant.DefaultNamespace
	} else {
		client.Namespace = config.Namespace
	}
	cf.Namespace = &client.Namespace
	installActionConfig := new(action.Configuration)
	if err := installActionConfig.Init(cf, client.Namespace, helmDriver, nolog); err != nil {
		return nil, err
	}
	client.installActionConfig = installActionConfig
	unInstallActionConfig := new(action.Configuration)
	if err := unInstallActionConfig.Init(cf, config.OldNamespace, helmDriver, nolog); err != nil {
		return nil, err
	}
	client.unInstallActionConfig = unInstallActionConfig
	return &client, nil
}

func (c Client) Install(name, chartName, chartVersion string, values map[string]interface{}) (*release.Release, error) {
	if err := updateRepo(c.Architectures); err != nil {
		return nil, err
	}
	client := action.NewInstall(c.installActionConfig)
	client.ReleaseName = name
	client.Namespace = c.Namespace
	client.ChartPathOptions.InsecureSkipTLSverify = true
	if len(chartVersion) != 0 {
		client.ChartPathOptions.Version = chartVersion
	}
	p, err := client.ChartPathOptions.LocateChart(chartName, c.settings)
	if err != nil {
		return nil, fmt.Errorf("locate chart %s failed: %v", chartName, err)
	}
	ct, err := loader.Load(p)
	if err != nil {
		return nil, fmt.Errorf("load chart %s failed: %v", chartName, err)
	}

	release, err := client.Run(ct, values)
	if err != nil {
		return release, fmt.Errorf("install tool %s with chart %s failed: %v", name, chartName, err)
	}
	return release, nil
}

func (c Client) Upgrade(name, chartName, chartVersion string, values map[string]interface{}) (*release.Release, error) {
	if err := updateRepo(c.Architectures); err != nil {
		return nil, err
	}
	client := action.NewUpgrade(c.installActionConfig)
	client.Namespace = c.Namespace
	client.DryRun = false
	client.ChartPathOptions.InsecureSkipTLSverify = true
	client.ChartPathOptions.Version = chartVersion
	p, err := client.ChartPathOptions.LocateChart(chartName, c.settings)
	if err != nil {
		return nil, fmt.Errorf("locate chart %s failed: %v", chartName, err)
	}
	ct, err := loader.Load(p)
	if err != nil {
		return nil, fmt.Errorf("load chart %s failed: %v", chartName, err)
	}

	release, err := client.Run(name, ct, values)
	if err != nil {
		return release, fmt.Errorf("upgrade tool %s with chart %s failed: %v", name, chartName, err)
	}
	return release, nil
}

func (c Client) Uninstall(name string) (*release.UninstallReleaseResponse, error) {
	client := action.NewUninstall(c.unInstallActionConfig)
	release, err := client.Run(name)
	if err != nil {
		return release, fmt.Errorf("uninstall tool %s failed: %v", name, err)
	}
	return release, nil
}

func (c Client) List() ([]*release.Release, error) {
	client := action.NewList(c.unInstallActionConfig)
	client.All = true
	release, err := client.Run()
	if err != nil {
		return release, fmt.Errorf("list chart failed: %v", err)
	}
	return release, nil
}

func GetSettings() *cli.EnvSettings {
	return &cli.EnvSettings{
		PluginsDirectory: helmpath.DataPath("plugins"),
		RegistryConfig:   helmpath.ConfigPath("registry.json"),
		RepositoryConfig: helmpath.ConfigPath("repositories.yaml"),
		RepositoryCache:  helmpath.CachePath("repository"),
	}

}

// 每次启用或升级的时候执行，存在 nexus 则不采取操作
func updateRepo(arch string) error {
	repos, err := ListRepo()
	if err != nil {
		logger.Log.Infof("list repo failed: %v, start reading from db repo", err)
	}
	flag := false
	for _, r := range repos {
		if r.Name == "nexus" {
			logger.Log.Infof("my nexus addr is %s", r.URL)
			flag = true
		}
	}
	if !flag {
		if err := addRepo(arch); err != nil {
			return err
		}
		if err := updateCharts(); err != nil {
			return err
		}
	}
	return nil
}

func (c Client) SyncRepoCharts(arch string) error {
	if err := addRepo(arch); err != nil {
		return err
	}
	if err := updateCharts(); err != nil {
		return err
	}
	return nil
}

func addRepo(arch string) error {
	username := "admin"
	name := "nexus"
	repository := repository.NewSystemSettingRepository()
	p, err := repository.Get("REGISTRY_PROTOCOL")
	if err != nil {
		return fmt.Errorf("load system repo failed: %v", err)
	}
	var c Client
	repoIP, password, repoPort, _, err := c.GetRepoIP(arch)
	if err != nil {
		return fmt.Errorf("load system repo of arch %s failed: %v", arch, err)
	}
	url := fmt.Sprintf("%s://%s:%d/repository/applications", p.Value, repoIP, repoPort)
	logger.Log.Infof("my helm repo url is %s", url)

	settings := GetSettings()

	repoFile := settings.RepositoryConfig

	if err := os.MkdirAll(filepath.Dir(repoFile), os.ModePerm); err != nil && !os.IsExist(err) {
		return err
	}

	fileLock := flock.New(strings.Replace(repoFile, filepath.Ext(repoFile), ".lock", 1))
	lockCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	locked, err := fileLock.TryLockContext(lockCtx, time.Second)
	if err == nil && locked {
		defer func() {
			if err := fileLock.Unlock(); err != nil {
				logger.Log.Errorf("addRepo fileLock.Unlock failed, error: %s", err.Error())
			}
		}()
	}
	if err != nil {
		return err
	}

	b, err := ioutil.ReadFile(repoFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var f repo.File
	if err := yaml.Unmarshal(b, &f); err != nil {
		return err
	}

	e := repo.Entry{
		Name:                  name,
		URL:                   url,
		Username:              username,
		Password:              password,
		InsecureSkipTLSverify: true,
	}

	r, err := repo.NewChartRepository(&e, getter.All(settings))
	if err != nil {
		return err
	}
	r.CachePath = settings.RepositoryCache
	if _, err := r.DownloadIndexFile(); err != nil {
		return errors.Wrapf(err, "looks like %q is not a valid chart repository or cannot be reached", url)
	}

	f.Update(&e)

	if err := f.WriteFile(repoFile, 0644); err != nil {
		return err
	}
	return nil
}

func updateCharts() error {
	logger.Log.Debug("Hang tight while we grab the latest from your chart repositories...")
	settings := GetSettings()
	repoFile := settings.RepositoryConfig
	repoCache := settings.RepositoryCache
	f, err := repo.LoadFile(repoFile)
	if err != nil {
		return fmt.Errorf("load file of repo %s failed: %v", repoFile, err)
	}
	var rps []*repo.ChartRepository
	for _, cfg := range f.Repositories {
		r, err := repo.NewChartRepository(cfg, getter.All(settings))
		if err != nil {
			return fmt.Errorf("get new chart repository failed, err: %v", err.Error())
		}
		if repoCache != "" {
			r.CachePath = repoCache
		}
		rps = append(rps, r)
	}

	var wg sync.WaitGroup
	for _, re := range rps {
		wg.Add(1)
		go func(re *repo.ChartRepository) {
			defer wg.Done()
			if _, err := re.DownloadIndexFile(); err != nil {
				logger.Log.Debugf("...Unable to get an update from the %q chart repository (%s):\n\t%s\n", re.Config.Name, re.Config.URL, err)
			} else {
				logger.Log.Debugf("...Successfully got an update from the %q chart repository\n", re.Config.Name)
			}
		}(re)
	}
	wg.Wait()
	logger.Log.Debugf("Update Complete. ⎈ Happy Helming!⎈ ")
	return nil
}

func (c Client) GetRepoIP(arch string) (string, string, int, int, error) {
	var repo model.SystemRegistry
	switch arch {
	case "amd64":
		if err := db.DB.Where("architecture = ?", constant.ArchitectureOfAMD64).First(&repo).Error; err != nil {
			return "", "", 0, 0, err
		}
		p, err := encrypt.StringDecrypt(repo.NexusPassword)
		if err != nil {
			return repo.Hostname, repo.NexusPassword, repo.RepoPort, repo.RegistryPort, fmt.Errorf("decrypt password %s failed, err: %v", p, err)
		}
		return repo.Hostname, p, repo.RepoPort, repo.RegistryPort, nil
	case "arm64", "all":
		if err := db.DB.Where("architecture = ?", constant.ArchitectureOfARM64).First(&repo).Error; err != nil {
			return "", "", 0, 0, err
		}
		p, err := encrypt.StringDecrypt(repo.NexusPassword)
		if err != nil {
			return repo.Hostname, repo.NexusPassword, repo.RepoPort, repo.RegistryPort, fmt.Errorf("decrypt password %s failed, err: %v", p, err)
		}
		return repo.Hostname, p, repo.RepoPort, repo.RegistryPort, nil
	}
	return "", "", 0, 0, errors.New("no such architecture")
}

func ListRepo() ([]*repo.Entry, error) {
	settings := GetSettings()
	var repos []*repo.Entry
	f, err := repo.LoadFile(settings.RepositoryConfig)
	if err != nil {
		return repos, err
	}
	return f.Repositories, nil
}
