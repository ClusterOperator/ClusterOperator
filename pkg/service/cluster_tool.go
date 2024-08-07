package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ClusterOperator/ClusterOperator/pkg/constant"
	"github.com/ClusterOperator/ClusterOperator/pkg/db"
	"github.com/ClusterOperator/ClusterOperator/pkg/dto"
	"github.com/ClusterOperator/ClusterOperator/pkg/logger"
	"github.com/ClusterOperator/ClusterOperator/pkg/model"
	"github.com/ClusterOperator/ClusterOperator/pkg/repository"
	"github.com/ClusterOperator/ClusterOperator/pkg/service/cluster/tools"
	clusterUtil "github.com/ClusterOperator/ClusterOperator/pkg/util/cluster"
	helm2 "github.com/ClusterOperator/ClusterOperator/pkg/util/helm"
	appv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterToolService interface {
	List(clusterName string) ([]dto.ClusterTool, error)
	GetNodePort(clusterName, toolName string) (string, error)
	SyncStatus(clusterName string) ([]dto.ClusterTool, error)
	Enable(clusterName string, tool dto.ClusterTool) (dto.ClusterTool, error)
	Upgrade(clusterName string, tool dto.ClusterTool) (dto.ClusterTool, error)
	Disable(clusterName string, tool dto.ClusterTool) (dto.ClusterTool, error)
	GetFlex(clusterName string) (string, error)
	EnableFlex(clusterName string) error
	DisableFlex(clusterName string) error
}

func NewClusterToolService() ClusterToolService {
	return &clusterToolService{
		toolRepo:        repository.NewClusterToolRepository(),
		clusterRepo:     repository.NewClusterRepository(),
		clusterNodeRepo: repository.NewClusterNodeRepository(),
		clusterSpecRepo: repository.NewClusterSpecRepository(),
		clusterService:  NewClusterService(),
	}
}

type clusterToolService struct {
	toolRepo        repository.ClusterToolRepository
	clusterRepo     repository.ClusterRepository
	clusterNodeRepo repository.ClusterNodeRepository
	clusterSpecRepo repository.ClusterSpecRepository
	clusterService  ClusterService
}

func (c clusterToolService) List(clusterName string) ([]dto.ClusterTool, error) {
	var items []dto.ClusterTool
	ms, err := c.toolRepo.List(clusterName)
	if err != nil {
		return items, err
	}
	for _, m := range ms {
		d := dto.ClusterTool{ClusterTool: m}
		d.Vars = map[string]interface{}{}
		_ = json.Unmarshal([]byte(m.Vars), &d.Vars)
		items = append(items, d)
	}
	return items, nil
}

func (c clusterToolService) GetNodePort(clusterName, toolName string) (string, error) {
	var (
		tool      model.ClusterTool
		svcName   string
		namespace string
	)
	cluster, err := c.clusterRepo.GetWithPreload(clusterName, []string{"SpecConf", "Secret", "Nodes", "Nodes.Host", "Nodes.Host.Credential"})
	if err != nil {
		return "", err
	}
	if err := db.DB.Where("name = ? AND cluster_id = ?", toolName, cluster.ID).First(&tool).Error; err != nil {
		return "", err
	}
	client, err := clusterUtil.NewClusterClient(&cluster)
	if err != nil {
		return "", err
	}

	valueMap := map[string]interface{}{}
	if err := json.Unmarshal([]byte(tool.Vars), &valueMap); err != nil {
		return "", err
	}
	if _, ok := valueMap["namespace"]; ok {
		namespace = fmt.Sprint(valueMap["namespace"])
	} else {
		return "", fmt.Errorf("cant not find namespace in tool vars: %s", tool.Vars)
	}
	switch toolName {
	case "prometheus":
		svcName = "prometheus-server"
	}
	d, err := client.CoreV1().Services(namespace).Get(context.TODO(), svcName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	if len(d.Spec.Ports) != 0 {
		return fmt.Sprintf("http://%s:%v", cluster.SpecConf.KubeRouter, d.Spec.Ports[0].NodePort), nil
	}
	return "", fmt.Errorf("can't get nodeport %s(%s) from cluster %s", svcName, namespace, clusterName)
}

func (c clusterToolService) SyncStatus(clusterName string) ([]dto.ClusterTool, error) {
	var (
		tools     []model.ClusterTool
		backTools []dto.ClusterTool
	)
	cluster, err := c.clusterRepo.GetWithPreload(clusterName, []string{"SpecConf", "Secret", "Nodes", "Nodes.Host", "Nodes.Host.Credential"})
	if err != nil {
		return backTools, err
	}
	client, err := clusterUtil.NewClusterClient(&cluster)
	if err != nil {
		return backTools, err
	}
	if err := db.DB.Where("cluster_id = ?", cluster.ID).Find(&tools).Error; err != nil {
		return backTools, err
	}
	var (
		allDeployments  []appv1.Deployment
		allStatefulsets []appv1.StatefulSet
	)
	namespaceList, err := client.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return backTools, err
	}
	for _, ns := range namespaceList.Items {
		deployments, err := client.AppsV1().Deployments(ns.Name).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return backTools, err
		}
		allDeployments = append(allDeployments, deployments.Items...)
		statefulsets, err := client.AppsV1().StatefulSets(ns.Name).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return backTools, err
		}
		allStatefulsets = append(allStatefulsets, statefulsets.Items...)
	}
	for _, tool := range tools {
		dtoItem := dto.ClusterTool{
			ClusterTool: tool,
			Vars:        map[string]interface{}{},
		}
		isExist := false
		sourceName := ""
		sourceType := "deployment"
		switch tool.Name {
		case "registry":
			sourceName = constant.DefaultRegistryDeploymentName
		case "chartmuseum":
			sourceName = constant.DefaultChartmuseumDeploymentName
		case "gatekeeper":
			sourceName = constant.DefaultGatekeeperDeploymentName
		case "kubeapps":
			sourceName = constant.DefaultKubeappsDeploymentName
		case "grafana":
			sourceName = constant.DefaultGrafanaDeploymentName
		case "prometheus":
			sourceName = constant.DefaultPrometheusDeploymentName
		case "logging":
			sourceName = constant.DefaultLoggingStateSetsfulName
			sourceType = "statefulset"
		case "loki":
			sourceName = constant.DefaultLokiStateSetsfulName
			sourceType = "statefulset"
		}
		if sourceType == "deployment" {
			for _, deploy := range allDeployments {
				if deploy.ObjectMeta.Name == sourceName {
					isExist = true
					if deploy.Status.ReadyReplicas == deploy.Status.Replicas {
						tool.Status = constant.StatusRunning
					} else {
						tool.Status = constant.StatusInitializing
					}
					dtoItem.Vars["namespace"] = deploy.ObjectMeta.Namespace
					buf, _ := json.Marshal(&dtoItem.Vars)
					tool.Vars = string(buf)
					_ = db.DB.Model(&model.ClusterTool{}).Updates(&tool)
					break
				}
			}
		}
		if sourceType == "statefulset" {
			for _, statefulset := range allStatefulsets {
				if statefulset.ObjectMeta.Name == sourceName {
					isExist = true
					if statefulset.Status.ReadyReplicas == statefulset.Status.Replicas {
						tool.Status = constant.StatusRunning
					} else {
						tool.Status = constant.StatusInitializing
					}
					dtoItem.Vars["namespace"] = statefulset.ObjectMeta.Namespace
					buf, _ := json.Marshal(&dtoItem.Vars)
					tool.Vars = string(buf)
					_ = db.DB.Model(&model.ClusterTool{}).Updates(&tool)
					break
				}
			}
		}
		if !isExist {
			if tool.Status != constant.StatusWaiting {
				tool.Status = constant.StatusWaiting
				_ = db.DB.Model(&model.ClusterTool{}).Updates(&tool)
			}
		}
		dtoItem.ClusterTool = tool
		backTools = append(backTools, dtoItem)
	}

	var h helm2.Client
	err = h.SyncRepoCharts(cluster.Architectures)
	return backTools, err
}

func (c clusterToolService) Disable(clusterName string, tool dto.ClusterTool) (dto.ClusterTool, error) {
	cluster, err := c.clusterRepo.GetWithPreload(clusterName, []string{"SpecConf", "Secret", "Nodes", "Nodes.Host", "Nodes.Host.Credential"})
	if err != nil {
		return tool, err
	}
	var hosts []string
	port := cluster.SpecConf.KubeApiServerPort
	for _, node := range cluster.Nodes {
		if node.Role == constant.NodeRoleNameMaster {
			if len(node.Host.Ip) != 0 {
				hosts = append(hosts, fmt.Sprintf("%s:%d", node.Host.Ip, port))
			}
		}
	}
	availableHost, err := clusterUtil.SelectAliveHost(hosts)
	if err != nil {
		return tool, err
	}

	tool.ClusterID = cluster.ID
	mo := tool.ClusterTool
	buf, _ := json.Marshal(&tool.Vars)
	mo.Vars = string(buf)
	tool.ClusterTool = mo

	itemValue, ok := tool.Vars["namespace"]
	namespace := ""
	if !ok {
		namespace = constant.DefaultNamespace
	} else {
		namespace = itemValue.(string)
	}

	ct, err := tools.NewClusterTool(&tool.ClusterTool, cluster, availableHost, namespace, namespace, false)
	if err != nil {
		return tool, err
	}
	mo.Status = constant.StatusTerminating
	_ = c.toolRepo.Save(&mo)
	go c.doUninstall(ct, &tool.ClusterTool)
	return tool, nil
}

func (c clusterToolService) Enable(clusterName string, tool dto.ClusterTool) (dto.ClusterTool, error) {
	cluster, err := c.clusterRepo.GetWithPreload(clusterName, []string{"SpecConf", "Secret", "Nodes", "Nodes.Host", "Nodes.Host.Credential"})
	if err != nil {
		return tool, err
	}
	availableHost := cluster.SpecConf.KubeRouter
	if cluster.Source != constant.ClusterSourceExternal {
		var hosts []string
		port := cluster.SpecConf.KubeApiServerPort
		for _, node := range cluster.Nodes {
			if node.Role == constant.NodeRoleNameMaster {
				if len(node.Host.Ip) != 0 {
					hosts = append(hosts, fmt.Sprintf("%s:%d", node.Host.Ip, port))
				}
			}
		}
		availableHost, err = clusterUtil.SelectAliveHost(hosts)
		if err != nil {
			return tool, err
		}
	}

	var toolDetail model.ClusterToolDetail
	if err := db.DB.Where("name = ? AND version = ?", tool.Name, tool.Version).Find(&toolDetail).Error; err != nil {
		return tool, err
	}

	tool.ClusterID = cluster.ID
	mo := tool.ClusterTool
	buf, _ := json.Marshal(&tool.Vars)
	mo.Vars = string(buf)
	tool.ClusterTool = mo

	oldNamespace, namespace := c.getNamespace(cluster.ID, tool)
	ct, err := tools.NewClusterTool(&tool.ClusterTool, cluster, availableHost, oldNamespace, namespace, true)
	if err != nil {
		return tool, err
	}
	mo.Status = constant.StatusInitializing
	_ = c.toolRepo.Save(&mo)
	go c.doInstall(ct, &tool.ClusterTool, toolDetail)
	return tool, nil
}

func (c clusterToolService) Upgrade(clusterName string, tool dto.ClusterTool) (dto.ClusterTool, error) {
	cluster, err := c.clusterRepo.GetWithPreload(clusterName, []string{"SpecConf", "Secret", "Nodes", "Nodes.Host", "Nodes.Host.Credential"})
	if err != nil {
		return tool, err
	}
	availableHost := cluster.SpecConf.KubeRouter
	if cluster.Source != constant.ClusterSourceExternal {
		var hosts []string
		port := cluster.SpecConf.KubeApiServerPort
		for _, node := range cluster.Nodes {
			if node.Role == constant.NodeRoleNameMaster {
				if len(node.Host.Ip) != 0 {
					hosts = append(hosts, fmt.Sprintf("%s:%d", node.Host.Ip, port))
				}
			}
		}
		availableHost, err = clusterUtil.SelectAliveHost(hosts)
		if err != nil {
			return tool, err
		}
	}

	var toolDetail model.ClusterToolDetail
	if err := db.DB.Where("name = ? AND version = ?", tool.Name, tool.HigherVersion).Find(&toolDetail).Error; err != nil {
		return tool, err
	}

	tool.ClusterID = cluster.ID
	mo := tool.ClusterTool
	buf, _ := json.Marshal(&tool.Vars)
	mo.Vars = string(buf)
	mo.Status = constant.StatusUpgrading
	mo.Version = mo.HigherVersion
	mo.HigherVersion = ""
	tool.ClusterTool = mo

	itemValue, ok := tool.Vars["namespace"]
	namespace := ""
	if !ok {
		namespace = constant.DefaultNamespace
	} else {
		namespace = itemValue.(string)
	}
	ct, err := tools.NewClusterTool(&tool.ClusterTool, cluster, availableHost, namespace, namespace, true)
	if err != nil {
		return tool, err
	}

	_ = c.toolRepo.Save(&mo)
	go c.doUpgrade(ct, &tool.ClusterTool, toolDetail)
	return tool, nil
}

func (c clusterToolService) GetFlex(clusterName string) (string, error) {
	cluster, err := c.clusterRepo.Get(clusterName)
	if err != nil {
		return "", err
	}
	master, err := c.clusterNodeRepo.FirstMaster(cluster.ID)
	if err != nil {
		return "", err
	}
	return master.Host.FlexIp, nil
}

func (c clusterToolService) EnableFlex(clusterName string) error {
	cluster, err := c.clusterRepo.Get(clusterName)
	if err != nil {
		return err
	}
	master, err := c.clusterNodeRepo.FirstMaster(cluster.ID)
	if err != nil {
		return err
	}
	if len(master.Host.FlexIp) == 0 {
		return errors.New("CLUSTER_NO_FLEX")
	}
	cluster.SpecConf.KubeRouter = master.Host.FlexIp
	if err := c.clusterSpecRepo.SaveConf(&cluster.SpecConf); err != nil {
		return err
	}
	return nil
}

func (c clusterToolService) DisableFlex(clusterName string) error {
	cluster, err := c.clusterRepo.Get(clusterName)
	if err != nil {
		return err
	}
	master, err := c.clusterNodeRepo.FirstMaster(cluster.ID)
	if err != nil {
		return err
	}
	cluster.SpecConf.KubeRouter = master.Host.Ip
	if err := c.clusterSpecRepo.SaveConf(&cluster.SpecConf); err != nil {
		return err
	}
	return nil
}

func (c clusterToolService) doInstall(p tools.Interface, tool *model.ClusterTool, toolDetail model.ClusterToolDetail) {
	err := p.Install(toolDetail)
	if err != nil {
		logger.Log.Errorf("install tool %s failed: %+v", tool.Name, err)
		tool.Status = constant.StatusFailed
		tool.Message = err.Error()
	} else {
		logger.Log.Infof("install tool %s successful: %+v", tool.Name, err)
		tool.Status = constant.StatusRunning
	}
	_ = c.toolRepo.Save(tool)
}

func (c clusterToolService) doUpgrade(p tools.Interface, tool *model.ClusterTool, toolDetail model.ClusterToolDetail) {
	err := p.Upgrade(toolDetail)
	if err != nil {
		logger.Log.Errorf("upgrade tool %s failed: %+v", tool.Name, err)
		tool.Status = constant.StatusFailed
		tool.Message = err.Error()
	} else {
		logger.Log.Infof("upgrade tool %s successful: %+v", tool.Name, err)
		tool.Status = constant.StatusRunning
	}
	_ = c.toolRepo.Save(tool)
}

func (c clusterToolService) doUninstall(p tools.Interface, tool *model.ClusterTool) {
	if err := p.Uninstall(); err != nil {
		logger.Log.Errorf("uninstall %s failed: %+v", tool.Name, err)
	} else {
		logger.Log.Infof("uninstall tool %s successful: %+v", tool.Name, err)
	}
	tool.Status = constant.StatusWaiting
	_ = c.toolRepo.Save(tool)
}

func (c clusterToolService) getNamespace(clusterID string, tool dto.ClusterTool) (string, string) {
	namespace := ""
	Sp, ok := tool.Vars["namespace"]
	if !ok {
		namespace = constant.DefaultNamespace
	} else {
		namespace = Sp.(string)
	}
	var oldTools model.ClusterTool
	if err := db.DB.Where("cluster_id = ? AND name = ?", clusterID, tool.Name).First(&oldTools).Error; err != nil {
		return namespace, namespace
	}
	oldVars := map[string]interface{}{}
	_ = json.Unmarshal([]byte(oldTools.Vars), &oldVars)
	oldSp, ok := oldVars["namespace"]
	if !ok {
		return namespace, namespace
	} else {
		return oldSp.(string), namespace
	}
}
