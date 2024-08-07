package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ClusterOperator/ClusterOperator/pkg/constant"
	"github.com/ClusterOperator/ClusterOperator/pkg/db"
	"github.com/ClusterOperator/ClusterOperator/pkg/dto"
	"github.com/ClusterOperator/ClusterOperator/pkg/logger"
	"github.com/ClusterOperator/ClusterOperator/pkg/model"
	"github.com/ClusterOperator/ClusterOperator/pkg/repository"
	clusterUtil "github.com/ClusterOperator/ClusterOperator/pkg/util/cluster"
	"github.com/icza/dyno"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type ClusterImportService interface {
	Import(clusterImport dto.ClusterImport) error
	LoadClusterInfo(loadInfo *dto.ClusterLoad) (dto.ClusterLoadInfo, error)
}

type clusterImportService struct {
	clusterRepo               repository.ClusterRepository
	projectRepository         repository.ProjectRepository
	projectResourceRepository repository.ProjectResourceRepository
	msgService                MsgService
	componentService          ComponentService
}

func NewClusterImportService() *clusterImportService {
	return &clusterImportService{
		clusterRepo:               repository.NewClusterRepository(),
		projectRepository:         repository.NewProjectRepository(),
		projectResourceRepository: repository.NewProjectResourceRepository(),
		msgService:                NewMsgService(),
		componentService:          NewComponentService(),
	}
}

func (c clusterImportService) Import(clusterImport dto.ClusterImport) error {
	loginfo, _ := json.Marshal(clusterImport)
	logger.Log.WithFields(logrus.Fields{"cluster_import_info": string(loginfo)}).Debugf("start to import the cluster %s", clusterImport.Name)

	project, err := c.projectRepository.Get(clusterImport.ProjectName)
	if err != nil {
		return err
	}
	cluster, err := clusterImport.ClusterImportDto2Mo()
	if err != nil {
		return err
	}
	cluster.ProjectID = project.ID

	tx := db.DB.Begin()
	if err := tx.Create(&cluster.Secret).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("can not create cluster secret %s", err.Error())
	}
	cluster.SecretID = cluster.Secret.ID
	if err := tx.Create(&cluster).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("can not create cluster secret %s", err.Error())
	}

	var synchosts []dto.HostSync
	tools := cluster.PrepareTools()
	if clusterImport.IsKoCluster {
		masterNum := 0
		workerNum := 0
		for _, node := range clusterImport.KoClusterInfo.Nodes {
			host := model.Host{
				Name:         node.Name,
				Ip:           node.Ip,
				Port:         node.Port,
				CredentialID: node.CredentialID,
				ClusterID:    cluster.ID,
				Status:       constant.StatusInitializing,
				Architecture: node.Architecture,
			}
			switch clusterImport.KoClusterInfo.NodeNameRule {
			case constant.NodeNameRuleIP, constant.NodeNameRuleHostName:
				host.Name = node.Name
			case constant.NodeNameRuleDefault:
				no := 0
				if node.Role == constant.NodeRoleNameMaster {
					masterNum++
					no = masterNum
				} else {
					workerNum++
					no = workerNum
				}
				host.Name = fmt.Sprintf("%s-%s-%d", cluster.Name, node.Role, no)
			}
			if err := tx.Create(&host).Error; err != nil {
				c.handlerImportError(tx, cluster, err)
				return err
			}

			synchosts = append(synchosts, dto.HostSync{HostName: node.Name, HostStatus: constant.StatusRunning})
			node := model.ClusterNode{
				Name:      node.Name,
				HostID:    host.ID,
				ClusterID: cluster.ID,
				Role:      node.Role,
				Status:    constant.StatusRunning,
			}
			if err := tx.Create(&node).Error; err != nil {
				c.handlerImportError(tx, cluster, err)
				return err
			}
			clusterResource := model.ClusterResource{
				ResourceType: constant.ResourceHost,
				ResourceID:   host.ID,
				ClusterID:    cluster.ID,
			}
			if err := tx.Create(&clusterResource).Error; err != nil {
				c.handlerImportError(tx, cluster, err)
				return err
			}
			projectResource := model.ProjectResource{
				ResourceType: constant.ResourceHost,
				ResourceID:   host.ID,
				ProjectID:    project.ID,
			}
			if err := tx.Create(&projectResource).Error; err != nil {
				c.handlerImportError(tx, cluster, err)
				return err
			}

		}
	} else {
		if err := gatherClusterInfo(cluster); err != nil {
			c.handlerImportError(tx, cluster, err)
			return err
		}
		for _, node := range cluster.Nodes {
			node.ClusterID = cluster.ID
			if err := tx.Create(&node).Error; err != nil {
				c.handlerImportError(tx, cluster, err)
				return fmt.Errorf("can not save node %s", err.Error())
			}
		}
	}

	cluster.SpecConf.ClusterID = cluster.ID
	if err := tx.Create(&cluster.SpecConf).Error; err != nil {
		tx.Rollback()
		return err
	}
	cluster.SpecRuntime.ClusterID = cluster.ID
	if err := tx.Create(&cluster.SpecRuntime).Error; err != nil {
		tx.Rollback()
		return err
	}
	cluster.SpecNetwork.ClusterID = cluster.ID
	if err := tx.Create(&cluster.SpecNetwork).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Save(&cluster).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("can not save cluster version %s", err.Error())
	}

	var (
		manifest model.ClusterManifest
		toolVars []model.VersionHelp
	)
	if err := tx.Where("version = ? OR name = ?", cluster.Version, cluster.Version).Order("created_at ASC").First(&manifest).Error; err != nil {
		logger.Log.Infof("can not find manifest version: %s", err.Error())
	}
	if manifest.ID != "" {
		if err := json.Unmarshal([]byte(manifest.ToolVars), &toolVars); err != nil {
			c.handlerImportError(tx, cluster, err)
			return fmt.Errorf("unmarshal manifest.toolvar error %s", err.Error())
		}
		for i := 0; i < len(tools); i++ {
			for _, item := range toolVars {
				if tools[i].Name == item.Name {
					tools[i].Version = item.Version
					break
				}
			}
		}
	}
	for _, tool := range tools {
		tool.ClusterID = cluster.ID
		if err := tx.Create(&tool).Error; err != nil {
			c.handlerImportError(tx, cluster, err)
			return fmt.Errorf("can not save tool %s", err.Error())
		}
	}
	subscribe := model.NewMsgSubscribe(constant.ClusterOperator, constant.Cluster, cluster.ID)
	if err := tx.Create(&subscribe).Error; err != nil {
		c.handlerImportError(tx, cluster, err)
	}
	var user model.User
	_ = db.DB.Model(&model.User{}).Where("name = 'admin'").First(&user).Error
	if user.ID != "" {
		subscribeUser := model.MsgSubscribeUser{
			SubscribeID: subscribe.ID,
			UserID:      user.ID,
		}
		if err := tx.Create(&subscribeUser).Error; err != nil {
			c.handlerImportError(tx, cluster, err)
		}
	}

	if err := c.projectResourceRepository.Create(model.ProjectResource{
		ResourceID:   cluster.ID,
		ProjectID:    project.ID,
		ResourceType: constant.ResourceCluster,
	}); err != nil {
		c.handlerImportError(tx, cluster, err)
		return fmt.Errorf("can not create project resource %s", err.Error())
	}
	tx.Commit()
	_ = c.msgService.SendMsg(constant.ClusterImport, constant.System, cluster, true, map[string]string{"detailName": cluster.Name})

	hostService := NewHostService()
	go func() {
		_ = hostService.SyncList(synchosts)
	}()
	go func() {
		syncList := []string{"metrics-server", "traefik", "ingress-nginx", "gpu", "dns-cache", "istio", "npd"}
		syncDTO := &dto.ComponentSync{ClusterName: cluster.Name, Names: syncList}
		_ = c.componentService.Sync(syncDTO)
	}()
	return nil
}

func (c clusterImportService) handlerImportError(tx *gorm.DB, cluster *model.Cluster, err error) {
	tx.Rollback()
	_ = c.msgService.SendMsg(constant.ClusterImport, constant.Cluster, cluster, false, map[string]string{"errMsg": err.Error(), "detailName": cluster.Name})
}

func gatherClusterInfo(cluster *model.Cluster) error {
	c, err := clusterUtil.NewClusterClient(cluster)
	if err != nil {
		return err
	}
	cluster.Version, err = getServerVersion(c, false)
	if err != nil {
		return err
	}
	var nodesFromK8s []dto.NodesFromK8s
	nodesFromK8s, cluster.SpecRuntime.RuntimeType, _, err = getKubeNodes(false, c)
	if err != nil {
		return err
	}
	for _, n := range nodesFromK8s {
		cluster.Nodes = append(cluster.Nodes, model.ClusterNode{
			Name:   n.Name,
			Role:   n.Role,
			Status: constant.StatusRunning,
		})
	}
	cluster.SpecNetwork.NetworkType, err = getInfoFromDaemonset(c)
	if err != nil {
		return err
	}
	return nil
}

func getServerVersion(client *kubernetes.Clientset, isKoCluster bool) (string, error) {
	v, err := client.ServerVersion()
	if err != nil {
		return "", fmt.Errorf("get version from cluster failed: %v", err.Error())
	}
	if !isKoCluster {
		return v.GitVersion, nil
	}
	var manifest model.ClusterManifest
	if err := db.DB.Where("version = ?", v.GitVersion).Order("created_at ASC").First(&manifest).Error; err != nil {
		return "", fmt.Errorf("get manifest %s from db failed: %v", v.GitVersion, err.Error())
	}
	return manifest.Name, nil
}

func getKubeNodes(isKoImport bool, client *kubernetes.Clientset) ([]dto.NodesFromK8s, string, string, error) {
	var (
		k8sNodes    []dto.NodesFromK8s
		runtimeType string
		clusterName string
	)

	nodes, err := client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return k8sNodes, runtimeType, clusterName, fmt.Errorf("get nodes from cluster failed: %v", err)
	}
	for i, node := range nodes.Items {
		if i == 0 {
			if strings.Contains(node.Status.NodeInfo.ContainerRuntimeVersion, "docker") {
				runtimeType = "docker"
			} else {
				runtimeType = "containerd"
			}
		}

		if isKoImport {
			if i == 0 {
				if strings.Contains(node.ObjectMeta.Name, "-master") {
					clusterName = strings.Split(node.ObjectMeta.Name, "-master")[0]
				} else if strings.Contains(node.ObjectMeta.Name, "-worker") {
					clusterName = strings.Split(node.ObjectMeta.Name, "-worker")[0]
				}
			} else {
				if strings.Contains(node.ObjectMeta.Name, "-master") {
					if clusterName != strings.Split(node.ObjectMeta.Name, "-master")[0] {
						clusterName = ""
					}
				} else if strings.Contains(node.ObjectMeta.Name, "-worker") {
					if clusterName != strings.Split(node.ObjectMeta.Name, "-worker")[0] {
						clusterName = ""
					}
				} else {
					clusterName = ""
				}
			}
		}

		var item dto.NodesFromK8s
		item.Name = node.ObjectMeta.Name
		if _, ok := node.ObjectMeta.Labels["node-role.kubernetes.io/master"]; ok {
			item.Role = "master"
		} else {
			item.Role = "worker"
		}
		item.Architecture = node.Status.NodeInfo.Architecture
		item.Port = 22
		for _, addr := range node.Status.Addresses {
			if addr.Type == "InternalIP" {
				item.Ip = addr.Address
			}
		}
		k8sNodes = append(k8sNodes, item)
	}
	return k8sNodes, runtimeType, clusterName, nil
}

func getInfoFromDaemonset(client *kubernetes.Clientset) (string, error) {
	var networkType string
	daemonsets, err := client.AppsV1().DaemonSets("kube-system").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return networkType, fmt.Errorf("get daemonsets from cluster failed: %v", err.Error())
	}
	for _, daemonset := range daemonsets.Items {
		if strings.Contains(daemonset.ObjectMeta.Name, "calico-node") {
			networkType = "calico"
		}
		if strings.Contains(daemonset.ObjectMeta.Name, "kube-flannel-ds") {
			networkType = "flannel"
		}
		if strings.Contains(daemonset.ObjectMeta.Name, "cilium") {
			networkType = "cilium"
		}
	}
	return networkType, nil
}

func (c clusterImportService) LoadClusterInfo(loadInfo *dto.ClusterLoad) (dto.ClusterLoadInfo, error) {
	var clusterInfo dto.ClusterLoadInfo
	if loadInfo.AuthenticationMode == constant.AuthenticationModeConfigFile {
		server, err := dto.LoadApiserverFromConfig(loadInfo.ConfigContent)
		if err != nil {
			return clusterInfo, fmt.Errorf("load failed. Check the imported Config file again, err: %v", err)
		}
		loadInfo.ApiServer = server
	}
	if strings.HasSuffix(loadInfo.ApiServer, "/") {
		loadInfo.ApiServer = strings.Replace(loadInfo.ApiServer, "/", "", -1)
	}
	loadInfo.ApiServer = strings.Replace(loadInfo.ApiServer, "http://", "", -1)
	loadInfo.ApiServer = strings.Replace(loadInfo.ApiServer, "https://", "", -1)
	if !strings.Contains(loadInfo.ApiServer, ":") {
		return clusterInfo, fmt.Errorf("check whether apiserver(%s) has no ports", loadInfo.ApiServer)
	}
	clusterInfo.LbKubeApiserverIp = strings.Split(loadInfo.ApiServer, ":")[0]
	clusterInfo.Architectures = loadInfo.Architectures

	var connConf rest.Config
	connConf.Insecure = true
	switch loadInfo.AuthenticationMode {
	case constant.AuthenticationModeBearer:
		connConf.Host = loadInfo.ApiServer
		connConf.BearerToken = loadInfo.Token
	case constant.AuthenticationModeCertificate:
		connConf.CertData = []byte(loadInfo.CertDataStr)
		connConf.KeyData = []byte(loadInfo.KeyDataStr)
	case constant.AuthenticationModeConfigFile:
		apiConfig, err := clusterUtil.PauseConfigApi(&loadInfo.ConfigContent)
		if err != nil {
			return clusterInfo, err
		}
		getter := func() (*api.Config, error) {
			return apiConfig, nil
		}
		itemConfig, err := clientcmd.BuildConfigFromKubeconfigGetter("", getter)
		if err != nil {
			return clusterInfo, err
		}
		connConf = *itemConfig
	}

	kubeClient, err := kubernetes.NewForConfig(&connConf)
	if err != nil {
		return clusterInfo, err
	}

	// load version
	clusterInfo.Version, err = getServerVersion(kubeClient, true)
	if err != nil {
		return clusterInfo, err
	}

	//load nodes
	clusterInfo.Nodes, clusterInfo.RuntimeType, clusterInfo.Name, err = getKubeNodes(true, kubeClient)
	if err != nil {
		return clusterInfo, err
	}
	if clusterInfo.Name == "" {
		clusterInfo.Name = loadInfo.Name
	}

	// load kubeadm-config
	kubeAdmMap, err := kubeClient.CoreV1().ConfigMaps("kube-system").Get(context.TODO(), "kubeadm-config", metav1.GetOptions{})
	if err != nil {
		return clusterInfo, fmt.Errorf("can not load kubeadm-config from cluster: %s", err.Error())
	}
	admConfig := kubeAdmMap.Data["ClusterConfiguration"]
	admCfy := make(map[interface{}]interface{})
	if err := yaml.Unmarshal([]byte(admConfig), &admCfy); err != nil {
		return clusterInfo, fmt.Errorf("kubeadm-config yaml unmarshall failed: %s", err.Error())
	}
	admInterface := dyno.ConvertMapI2MapS(admCfy)
	kk, err := json.Marshal(admInterface)
	if err != nil {
		return clusterInfo, fmt.Errorf("kubeadm-config json marshall failed: %s", err.Error())
	}
	var data admConfigStruct
	if err := json.Unmarshal(kk, &data); err != nil {
		return clusterInfo, fmt.Errorf("kubeadm-config json unmarshall failed: %s", err.Error())
	}
	if strings.Contains(data.ControlPlaneEndpoint, ":") {
		apiServerInCM := strings.Split(data.ControlPlaneEndpoint, ":")[0]
		if apiServerInCM == "127.0.0.1" {
			clusterInfo.LbKubeApiserverIp = strings.Split(loadInfo.ApiServer, ":")[0]
			clusterInfo.LbMode = constant.ClusterSourceInternal
		} else {
			clusterInfo.LbKubeApiserverIp = apiServerInCM
			clusterInfo.LbMode = constant.ClusterSourceExternal
		}
		clusterInfo.KubeApiServerPort, _ = strconv.Atoi(strings.Split(data.ControlPlaneEndpoint, ":")[1])
	} else {
		return clusterInfo, fmt.Errorf("err controlPlaneEndpoint from cluster configmap")
	}
	clusterInfo.KubeServiceNodePortRange = data.ApiServer.ExtraArgs.ServiceNodePortRange
	mask, _ := strconv.Atoi(data.Controller.ExtraArgs.NodeCidrMaskSize)
	clusterInfo.KubeNetworkNodePrefix = mask
	clusterInfo.KubeMaxPods = clusterUtil.MaxNodePodNumMap[mask]
	clusterInfo.KubePodSubnet = data.Network.PodSubnet
	clusterInfo.KubeDnsDomain = data.Network.DnsDomain
	clusterInfo.MaxNodePodNum = 2 << (31 - mask)
	if strings.Contains(clusterInfo.KubePodSubnet, "/") {
		subnets := strings.Split(clusterInfo.KubePodSubnet, "/")
		podMask, _ := strconv.Atoi(subnets[1])
		clusterInfo.MaxNodeNum = (2 << (31 - podMask)) / clusterInfo.MaxNodePodNum
	}
	clusterInfo.KubeServiceSubnet = data.Network.ServiceSubnet
	if len(data.ApiServer.ExtraArgs.AuditLogPath) == 0 {
		clusterInfo.KubernetesAudit = "no"
	} else {
		clusterInfo.KubernetesAudit = "yes"
	}

	// load kube-proxy
	kubeProxyMap, err := kubeClient.CoreV1().ConfigMaps("kube-system").Get(context.TODO(), "kube-proxy", metav1.GetOptions{})
	if err != nil {
		return clusterInfo, fmt.Errorf("can not load kube-proxy from cluster: %s", err.Error())
	}
	proxyConfig := kubeProxyMap.Data["config.conf"]
	proxyCfy := make(map[interface{}]interface{})
	if err := yaml.Unmarshal([]byte(proxyConfig), &proxyCfy); err != nil {
		return clusterInfo, fmt.Errorf("kube-proxy yaml unmarshall failed: %s", err.Error())
	}
	proxyInterface := dyno.ConvertMapI2MapS(proxyCfy)
	kk2, err := json.Marshal(proxyInterface)
	if err != nil {
		return clusterInfo, fmt.Errorf("kube-proxy json marshall failed: %s", err.Error())
	}
	var data2 proxyConfigStruct
	if err := json.Unmarshal(kk2, &data2); err != nil {
		return clusterInfo, fmt.Errorf("kube-proxy json unmarshall failed: %s", err.Error())
	}
	clusterInfo.KubeProxyMode = data2.Mode
	if len(data2.NodePortAddresses) != 0 {
		clusterInfo.NodeportAddress = data2.NodePortAddresses
	}

	// load network
	clusterInfo.NetworkType, err = getInfoFromDaemonset(kubeClient)
	if err != nil {
		return clusterInfo, err
	}

	return clusterInfo, nil
}

type admConfigStruct struct {
	ApiServer            apiServerStruct  `json:"apiServer"`
	ControlPlaneEndpoint string           `json:"controlPlaneEndpoint"`
	Controller           ControllerStruct `json:"controllerManager"`
	Network              networkStruct    `json:"networking"`
}

type proxyConfigStruct struct {
	Mode              string `json:"mode"`
	NodePortAddresses string `json:"nodePortAddresses"`
}

type apiServerStruct struct {
	ExtraArgs extraArgsStruct `json:"extraArgs"`
}

type ControllerStruct struct {
	ExtraArgs extraArgsStruct `json:"extraArgs"`
}

type networkStruct struct {
	DnsDomain     string `json:"dnsDomain"`
	PodSubnet     string `json:"podSubnet"`
	ServiceSubnet string `json:"serviceSubnet"`
}

type extraArgsStruct struct {
	ServiceNodePortRange string `json:"service-node-port-range"`
	NodeCidrMaskSize     string `json:"node-cidr-mask-size"`
	AuditLogPath         string `json:"audit-log-path"`
}
