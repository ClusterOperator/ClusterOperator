package model

import (
	"github.com/ClusterOperator/ClusterOperator/pkg/model/common"
	uuid "github.com/satori/go.uuid"
)

type ClusterStorageProvisioner struct {
	common.BaseModel
	ID        string `json:"id"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	Name      string `json:"name"    gorm:"not null;unique"`
	Namespace string `json:"namespace"`
	Message   string `json:"message" gorm:"type:text(65535)"`
	Vars      string `json:"-"    gorm:"type:text(65535)"`
	ClusterID string `json:"clusterId"`
}

func (c *ClusterStorageProvisioner) BeforeCreate() (err error) {
	c.ID = uuid.NewV4().String()
	return nil
}
