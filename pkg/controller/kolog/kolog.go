package kolog

import (
	"github.com/ClusterOperator/ClusterOperator/pkg/dto"
	"github.com/ClusterOperator/ClusterOperator/pkg/logger"
	"github.com/ClusterOperator/ClusterOperator/pkg/service"
)

func Save(name, operation, operationInfo string) {
	lS := service.NewSystemLogService()
	logInfo := dto.SystemLogCreate{
		Name:          name,
		Operation:     operation,
		OperationInfo: operationInfo,
	}
	if err := lS.Create(logInfo); err != nil {
		logger.Log.Errorf("save system logs failed, error: %s", err.Error())
	}
}
