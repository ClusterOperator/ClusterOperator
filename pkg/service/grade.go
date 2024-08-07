package service

import (
	"errors"

	"github.com/ClusterOperator/ClusterOperator/pkg/constant"
	"github.com/ClusterOperator/ClusterOperator/pkg/dto"
	"github.com/ClusterOperator/ClusterOperator/pkg/repository"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/polaris"
)

type GradeService interface {
	GetGrade(clusterName string) (*dto.ClusterGrade, error)
}

type gradeService struct {
	clusterRepo repository.ClusterRepository
}

func NewGradeService() GradeService {
	return &gradeService{
		clusterRepo: repository.NewClusterRepository(),
	}
}

func (g gradeService) GetGrade(clusterName string) (*dto.ClusterGrade, error) {
	cluster, err := g.clusterRepo.GetWithPreload(clusterName, []string{"SpecConf", "Secret"})
	if err != nil {
		return nil, err
	}

	if cluster.Status == constant.StatusRunning {
		result, err := polaris.RunGrade(&polaris.Config{
			Host:  cluster.SpecConf.LbKubeApiserverIp,
			Port:  cluster.SpecConf.KubeApiServerPort,
			Token: cluster.Secret.KubernetesToken,

			AuthenticationMode: cluster.SpecConf.AuthenticationMode,
			CertDataStr:        cluster.Secret.CertDataStr,
			KeyDataStr:         cluster.Secret.KeyDataStr,
			ConfigContent:      cluster.Secret.ConfigContent,
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	} else {
		return nil, errors.New("CLUSTER_IS_NOT_RUNNING")
	}
}
