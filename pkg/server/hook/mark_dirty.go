package hook

import (
	"time"

	"github.com/ClusterOperator/ClusterOperator/pkg/constant"
	"github.com/ClusterOperator/ClusterOperator/pkg/db"
	"github.com/ClusterOperator/ClusterOperator/pkg/logger"
	"github.com/ClusterOperator/ClusterOperator/pkg/model"
)

func init() {
	BeforeApplicationStart.AddFunc(recoverClusterTask)
}

var stableStatus = []string{constant.StatusRunning, constant.StatusFailed, constant.StatusNotReady, constant.StatusLost}
var statleTaskStatus = []string{constant.TaskLogStatusSuccess, constant.TaskLogStatusFailed}

// cluster
func recoverClusterTask() error {
	logger.Log.Info("Update status to failed caused by task cancel")
	tx := db.DB.Begin()
	if err := db.DB.Model(&model.Cluster{}).Where("status not in (?)", stableStatus).Updates(map[string]interface{}{
		"status":  constant.StatusFailed,
		"message": constant.TaskCancel,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := db.DB.Model(&model.TaskLog{}).Where("phase not in (?)", statleTaskStatus).Updates(map[string]interface{}{
		"phase":    constant.TaskLogStatusFailed,
		"message":  constant.TaskCancel,
		"end_time": time.Now().Unix(),
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&model.TaskLogDetail{}).Where("status = ?", constant.TaskLogStatusRunning).Updates(map[string]interface{}{
		"status":   constant.TaskLogStatusFailed,
		"message":  constant.TaskCancel,
		"end_time": time.Now().Unix(),
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&model.ClusterSpecComponent{}).Where("status not in (?)", []string{constant.StatusDisabled, constant.StatusEnabled, constant.StatusFailed}).Updates(map[string]interface{}{
		"status":  constant.StatusFailed,
		"message": constant.TaskCancel,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&model.Host{}).Where("status != ? AND status != ?", constant.StatusRunning, constant.StatusFailed).Updates(map[string]interface{}{
		"status":  constant.StatusFailed,
		"message": constant.TaskCancel,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&model.ClusterStorageProvisioner{}).Where("status not in (?)", stableStatus).Updates(map[string]interface{}{
		"status":  constant.StatusFailed,
		"message": constant.TaskCancel,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	var nodes []model.ClusterNode
	if err := db.DB.Where("status not in (?) AND status != ''", stableStatus).Find(&nodes).Error; err != nil {
		tx.Rollback()
		return err
	}
	for _, statu := range nodes {
		statu.Status = constant.StatusFailed
		statu.Message = constant.TaskCancel
		if err := tx.Save(&statu).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	tx.Commit()

	logger.Log.Info("update status successful !")
	return nil
}
