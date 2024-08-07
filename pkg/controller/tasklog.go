package controller

import (
	"github.com/ClusterOperator/ClusterOperator/pkg/constant"
	"github.com/ClusterOperator/ClusterOperator/pkg/controller/page"
	"github.com/ClusterOperator/ClusterOperator/pkg/dto"
	"github.com/ClusterOperator/ClusterOperator/pkg/service"
	"github.com/kataras/iris/v12/context"
)

type TaskLogController struct {
	Ctx            context.Context
	TaskLogService service.TaskLogService
}

func NewTaskLogController() *TaskLogController {
	return &TaskLogController{
		TaskLogService: service.NewTaskLogService(),
	}
}

// Search TaskLog
// @Tags task_logs
// @Summary Search tasklog
// @Description 过滤任务日志
// @Accept  json
// @Produce  json
// @Param conditions body condition.Conditions true "conditions"
// @Success 200 {object} page.Page
// @Security ApiKeyAuth
// @Router /tasklog/ [post]
func (c TaskLogController) Post() (*page.Page, error) {
	p, _ := c.Ctx.Values().GetBool("page")
	if p {
		num, _ := c.Ctx.Values().GetInt(constant.PageNumQueryKey)
		size, _ := c.Ctx.Values().GetInt(constant.PageSizeQueryKey)
		cluster := c.Ctx.URLParam("cluster")
		logtype := c.Ctx.URLParam("logtype")
		p, err := c.TaskLogService.Page(num, size, cluster, logtype)
		return p, err
	}
	return nil, nil
}

func (c TaskLogController) GetDetailBy(id string) (*dto.TaskLog, error) {
	return c.TaskLogService.GetTaskDetailByID(id)
}

func (c TaskLogController) GetLog1By(clusterId, logId string) (*dto.Logs, error) {
	return c.TaskLogService.GetTaskLogByID(clusterId, logId)
}

func (c TaskLogController) GetLog2By(clusterName, logId string) (*dto.Logs, error) {
	return c.TaskLogService.GetTaskLogByName(clusterName, logId)
}

func (c TaskLogController) GetBackupLogsBy(clusterName string) (*page.Page, error) {
	p, _ := c.Ctx.Values().GetBool("page")
	if p {
		num, _ := c.Ctx.Values().GetInt(constant.PageNumQueryKey)
		size, _ := c.Ctx.Values().GetInt(constant.PageSizeQueryKey)

		p, err := c.TaskLogService.GetBackupLogs(num, size, clusterName)
		return p, err
	}
	return nil, nil
}
