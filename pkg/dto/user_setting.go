package dto

import (
	"encoding/json"
	"github.com/ClusterOperator/ClusterOperator/pkg/model"
)

type UserSettingDTO struct {
	model.UserSetting
	MsgConfig interface{} `json:"msgConfig"`
}

func (u UserSettingDTO) GetMsgConfig() (string, error) {
	var re string
	by, err := json.Marshal(u.MsgConfig)
	if err != nil {
		return re, err
	}
	return string(by), nil
}
