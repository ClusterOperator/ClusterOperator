package dto

import "github.com/ClusterOperator/ClusterOperator/pkg/model"

type UserMsgDTO struct {
	model.UserMsg
	Content interface{} `json:"content"`
	Type    string      `json:"type"`
}

type UserMsgResponse struct {
	Items  []UserMsgDTO `json:"items"`
	Unread int          `json:"unread"`
	Total  int          `json:"total"`
}
