package model

import (
	"github.com/ClusterOperator/ClusterOperator/pkg/model/common"
	uuid "github.com/satori/go.uuid"
)

type UserMsg struct {
	common.BaseModel
	ID         string `json:"id"`
	Receive    string `json:"receive"`
	UserID     string `json:"userId"`
	MsgID      string `json:"msgId"`
	SendStatus string `json:"sendStatus"`
	ReadStatus string `json:"readStatus"`
	SendType   string `json:"sendType"`
	Msg        Msg    `json:"msg"`
}

func (m *UserMsg) BeforeCreate() error {
	m.ID = uuid.NewV4().String()
	return nil
}
