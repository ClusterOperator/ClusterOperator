package cloud_provider

import (
	"github.com/ClusterOperator/ClusterOperator/pkg/cloud_provider/client"
	"github.com/ClusterOperator/ClusterOperator/pkg/constant"
)

type CloudClient interface {
	ListDatacenter() ([]string, string, error)
	ListClusters() ([]interface{}, error)
	ListTemplates() ([]interface{}, error)
	ListFlavors() ([]interface{}, error)
	GetIpInUsed(network string) ([]string, error)
	UploadImage() error
	ImageExist(template string) (bool, error)
	CreateDefaultFolder() error
	ListDatastores() ([]client.DatastoreResult, error)
	ListFolders() ([]string, error)
}

func NewCloudClient(vars map[string]interface{}) CloudClient {
	switch vars["provider"] {
	case constant.OpenStack:
		return client.NewOpenStackClient(vars)
	case constant.VSphere:
		return client.NewVSphereClient(vars)
	case constant.FusionCompute:
		return client.NewFusionComputeClient(vars)
	}
	return nil
}
