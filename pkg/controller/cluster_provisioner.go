package controller

import (
	"fmt"

	"github.com/ClusterOperator/ClusterOperator/pkg/constant"
	"github.com/ClusterOperator/ClusterOperator/pkg/controller/kolog"
	"github.com/ClusterOperator/ClusterOperator/pkg/dto"
	"github.com/ClusterOperator/ClusterOperator/pkg/logger"
	"github.com/ClusterOperator/ClusterOperator/pkg/service"
	"github.com/kataras/iris/v12/context"
)

type ProvisionerController struct {
	Ctx                              context.Context
	ClusterStorageProvisionerService service.ClusterStorageProvisionerService
}

func NewProvisionerController() *ProvisionerController {
	return &ProvisionerController{
		ClusterStorageProvisionerService: service.NewClusterStorageProvisionerService(),
	}
}

func (c ProvisionerController) GetBy(name string) ([]dto.ClusterStorageProvisioner, error) {
	csp, err := c.ClusterStorageProvisionerService.ListStorageProvisioner(name)
	if err != nil {
		logger.Log.Info(fmt.Sprintf("%+v", err))
		return nil, err
	}
	return csp, nil
}

func (c ProvisionerController) PostBy(name string) error {
	var req dto.ClusterStorageProvisionerCreation
	err := c.Ctx.ReadJSON(&req)
	if err != nil {
		return err
	}
	if err := c.ClusterStorageProvisionerService.CreateStorageProvisioner(name, req); err != nil {
		logger.Log.Info(fmt.Sprintf("%+v", err))
		return err
	}

	operator := c.Ctx.Values().GetString("operator")
	go kolog.Save(operator, constant.CREATE_CLUSTER_STORAGE_SUPPLIER, name+"-"+req.Name+"("+req.Type+")")

	return nil
}

func (c ProvisionerController) PostSyncBy(name string) error {
	var req []dto.ClusterStorageProvisionerSync
	err := c.Ctx.ReadJSON(&req)
	if err != nil {
		return err
	}
	if err := c.ClusterStorageProvisionerService.SyncStorageProvisioner(name, req); err != nil {
		logger.Log.Info(fmt.Sprintf("%+v", err))
		return err
	}

	var proStr string
	for _, pro := range req {
		proStr += (pro.Name + ",")
	}
	operator := c.Ctx.Values().GetString("operator")
	go kolog.Save(operator, constant.SYNC_CLUSTER_STORAGE_SUPPLIER, proStr)

	return nil
}

func (c ProvisionerController) PostDeleteBy(clusterName string) error {
	var item dto.ClusterStorageProvisioner
	if err := c.Ctx.ReadJSON(&item); err != nil {
		logger.Log.Info(fmt.Sprintf("%+v", err))
		return err
	}
	operator := c.Ctx.Values().GetString("operator")
	go kolog.Save(operator, constant.DELETE_CLUSTER_STORAGE_SUPPLIER, clusterName+"-"+item.Name)

	return c.ClusterStorageProvisionerService.DeleteStorageProvisioner(clusterName, item.Name)
}
