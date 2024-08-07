package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ClusterOperator/ClusterOperator/pkg/constant"
	"github.com/ClusterOperator/ClusterOperator/pkg/db"
	"github.com/ClusterOperator/ClusterOperator/pkg/dto"
	"github.com/ClusterOperator/ClusterOperator/pkg/logger"
	"github.com/ClusterOperator/ClusterOperator/pkg/model"
	"github.com/ClusterOperator/ClusterOperator/pkg/repository"
	"github.com/ClusterOperator/ClusterOperator/pkg/service/cluster/adm"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/ansible"
)

type ClusterUpgradeService interface {
	Upgrade(upgrade dto.ClusterUpgrade) error
}

func NewClusterUpgradeService() ClusterUpgradeService {
	return &clusterUpgradeService{
		clusterService: NewClusterService(),
		msgService:     NewMsgService(),
		clusterRepo:    repository.NewClusterRepository(),
		taskLogService: NewTaskLogService(),
	}
}

type clusterUpgradeService struct {
	clusterService ClusterService
	msgService     MsgService
	clusterRepo    repository.ClusterRepository
	taskLogService TaskLogService
}

func (c *clusterUpgradeService) Upgrade(upgrade dto.ClusterUpgrade) error {
	loginfo, _ := json.Marshal(upgrade)
	logger.Log.WithFields(logrus.Fields{"cluster_upgrade_info": string(loginfo)}).Debugf("start to upgrade the cluster %s", upgrade.ClusterName)

	cluster, err := c.clusterRepo.GetWithPreload(upgrade.ClusterName, []string{"SpecConf", "SpecNetwork", "SpecRuntime", "Nodes", "Nodes.Host", "Nodes.Host.Credential", "Nodes.Host.Zone", "MultiClusterRepositories"})
	if err != nil {
		return fmt.Errorf("can not get cluster %s error %s", upgrade.ClusterName, err.Error())
	}
	if cluster.Source == constant.ClusterSourceExternal {
		return errors.New("CLUSTER_IS_NOT_LOCAL")
	}
	if cluster.Status != constant.StatusRunning && cluster.Status != constant.StatusFailed {
		return fmt.Errorf("cluster status error %s", cluster.Status)
	}

	tasklog, _ := c.taskLogService.GetByID(cluster.CurrentTaskID)
	if tasklog.ID != "" {
		cluster.TaskLog = tasklog
	}

	//从错误后继续
	if cluster.TaskLog.Phase == constant.TaskLogStatusFailed && cluster.TaskLog.Type == constant.TaskLogTypeClusterUpgrade {
		if err := c.taskLogService.RestartTask(&cluster, constant.TaskLogTypeClusterUpgrade); err != nil {
			return err
		}
	} else {
		isON := c.taskLogService.IsTaskOn(upgrade.ClusterName)
		if isON {
			return errors.New("TASK_IN_EXECUTION")
		}
		cluster.TaskLog = model.TaskLog{
			ClusterID: cluster.ID,
			Type:      constant.TaskLogTypeClusterUpgrade,
			Phase:     constant.TaskLogStatusWaiting,
		}
		if err := c.taskLogService.Save(&cluster.TaskLog); err != nil {
			return fmt.Errorf("reset contidion err %s", err.Error())
		}
	}

	// 创建日志
	writer, err := ansible.CreateAnsibleLogWriterWithId(cluster.Name, cluster.TaskLog.ID)
	if err != nil {
		_ = c.msgService.SendMsg(constant.ClusterUpgrade, constant.Cluster, cluster, false, map[string]string{"errMsg": err.Error(), "detailName": cluster.Name})
		return fmt.Errorf("create log error %s", err.Error())
	}
	if len(upgrade.Version) != 0 {
		cluster.UpgradeVersion = upgrade.Version
	}
	cluster.Status = constant.StatusUpgrading
	cluster.CurrentTaskID = cluster.TaskLog.ID
	if err := c.clusterRepo.Save(&cluster); err != nil {
		_ = c.msgService.SendMsg(constant.ClusterUpgrade, constant.Cluster, cluster, false, map[string]string{"errMsg": err.Error(), "detailName": cluster.Name})
		return fmt.Errorf("save cluster spec error %s", err.Error())
	}
	// 更新工具版本状态
	if err := c.updateToolVersion(upgrade.Version, cluster.ID); err != nil {
		logger.Log.Infof("update tool version of cluster %s failed, err: %v", cluster.Name, err)
	}

	logger.Log.Infof("update db data of cluster %s successful, now start to upgrade cluster", cluster.Name)
	go c.do(&cluster, writer)
	return nil
}

func (c *clusterUpgradeService) do(cluster *model.Cluster, writer io.Writer) {
	ctx, cancel := context.WithCancel(context.Background())
	admCluster := adm.NewAnsibleHelper(*cluster, writer)
	statusChan := make(chan adm.AnsibleHelper)
	go c.doUpgrade(ctx, *admCluster, statusChan)
	for {
		result := <-statusChan
		switch cluster.TaskLog.Phase {
		case constant.TaskLogStatusSuccess:
			if err := c.taskLogService.End(&cluster.TaskLog, true, ""); err != nil {
				logger.Log.Infof("save task failed %v", err)
			}
			logger.Log.Infof("cluster %s upgrade successful!", cluster.Name)
			cluster.Status = constant.StatusRunning
			cluster.Message = result.Message
			cluster.CurrentTaskID = ""
			_ = c.clusterRepo.Save(cluster)

			_ = c.msgService.SendMsg(constant.ClusterUpgrade, constant.Cluster, cluster, true, map[string]string{"detailName": cluster.Name})
			cluster.Version = cluster.UpgradeVersion
			_ = db.DB.Save(&cluster).Error
			cancel()
			return
		case constant.TaskLogStatusFailed:
			if err := c.taskLogService.End(&cluster.TaskLog, false, result.Message); err != nil {
				logger.Log.Infof("save task failed %v", err)
			}
			logger.Log.Infof("cluster %s upgrade failed!", cluster.Name)
			cluster.Status = constant.StatusFailed
			cluster.Message = result.Message
			_ = c.clusterRepo.Save(cluster)

			_ = c.msgService.SendMsg(constant.ClusterUpgrade, constant.Cluster, cluster, false, map[string]string{"errMsg": result.Message, "detailName": cluster.Name})
			cancel()
			return
		default:
			cluster.TaskLog.Phase = result.Status
			cluster.TaskLog.Message = result.Message
			cluster.TaskLog.Details = result.LogDetail
			if err := c.taskLogService.Save(&cluster.TaskLog); err != nil {
				logger.Log.Infof("save task failed %v", err)
			}
		}
	}
}

func (c clusterUpgradeService) doUpgrade(ctx context.Context, aHelper adm.AnsibleHelper, statusChan chan adm.AnsibleHelper) {
	ad := adm.NewClusterAdm()
	for {
		if err := ad.OnUpgrade(&aHelper); err != nil {
			aHelper.Message = err.Error()
		}
		select {
		case <-ctx.Done():
			return
		case statusChan <- aHelper:
		}
		time.Sleep(5 * time.Second)
	}
}

func (c clusterUpgradeService) updateToolVersion(version, clusterID string) error {
	var (
		tools    []model.ClusterTool
		manifest model.ClusterManifest
		toolVars []model.VersionHelp
	)
	if err := db.DB.Where("name = ?", version).First(&manifest).Error; err != nil {
		return fmt.Errorf("get manifest error %s", err.Error())
	}
	if err := db.DB.Where("cluster_id = ?", clusterID).Find(&tools).Error; err != nil {
		return fmt.Errorf("get tools error %s", err.Error())
	}
	if err := json.Unmarshal([]byte(manifest.ToolVars), &toolVars); err != nil {
		return fmt.Errorf("unmarshal manifest.toolvar error %s", err.Error())
	}
	for _, tool := range tools {
		for _, item := range toolVars {
			if tool.Name == item.Name {
				if tool.Version != item.Version {
					if tool.Status == constant.StatusWaiting {
						tool.Version = item.Version
					} else {
						tool.HigherVersion = item.Version
					}
					if err := db.DB.Save(&tool).Error; err != nil {
						return fmt.Errorf("update tool version error %s", err.Error())
					}
				}
				break
			}
		}
	}
	return nil
}
