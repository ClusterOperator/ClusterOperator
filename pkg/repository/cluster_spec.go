package repository

import (
	"github.com/ClusterOperator/ClusterOperator/pkg/db"
	"github.com/ClusterOperator/ClusterOperator/pkg/model"
)

type ClusterSpecRepository interface {
	SaveConf(conf *model.ClusterSpecConf) error
}

func NewClusterSpecRepository() ClusterSpecRepository {
	return &clusterSpecRepository{}
}

type clusterSpecRepository struct{}

func (c clusterSpecRepository) SaveConf(spec *model.ClusterSpecConf) error {
	if err := db.DB.Save(&spec).Error; err != nil {
		return err
	}
	return nil
}
