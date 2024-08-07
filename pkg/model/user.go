package model

import (
	"errors"

	"github.com/ClusterOperator/ClusterOperator/pkg/db"
	"github.com/ClusterOperator/ClusterOperator/pkg/model/common"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/encrypt"
	uuid "github.com/satori/go.uuid"
)

var (
	AdminCanNotDelete = "ADMIN_CAN_NOT_DELETE"
	LdapCanNotUpdate  = "LDAP_CAN_NOT_UPDATE"
)

const (
	EN string = "en-US"
	ZH string = "zh-CN"
)

type User struct {
	common.BaseModel
	ID               string  `json:"id" gorm:"type:varchar(64)"`
	CurrentProjectID string  `json:"-" gorm:"type:varchar(64)"`
	CurrentProject   Project `json:"-" gorm:"save_associations:false"`
	Name             string  `json:"name" gorm:"type:varchar(256);not null;unique"`
	Password         string  `json:"password" gorm:"type:varchar(256)"`
	Email            string  `json:"email" gorm:"type:varchar(256);not null;unique"`
	Language         string  `json:"language" gorm:"type:varchar(64)"`
	IsAdmin          bool    `json:"-" gorm:"type:boolean;default:false"`
	IsSuper          bool    `json:"-" gorm:"type:boolean;default:false"`
	IsActive         bool    `json:"-" gorm:"type:boolean;default:true"`
	Type             string  `json:"type" gorm:"type:varchar(64)"`
}

type Token struct {
	Token string `json:"access_token"`
}

func (u *User) BeforeCreate() (err error) {
	u.ID = uuid.NewV4().String()
	return err
}

func (u *User) AfterCreate() (err error) {
	setting := NewUserSetting(u.ID)
	err = db.DB.Model(UserSetting{}).Create(&setting).Error
	return
}

func (u *User) BeforeDelete() (err error) {
	if u.Name == "admin" {
		return errors.New(AdminCanNotDelete)
	}
	err = db.DB.Model(ProjectMember{}).Where("user_id =?", u.ID).Delete(&ProjectMember{}).Error
	if err != nil {
		return err
	}
	err = db.DB.Model(ClusterMember{}).Where("user_id =?", u.ID).Delete(&ClusterMember{}).Error
	if err != nil {
		return err
	}
	err = db.DB.Model(UserMsg{}).Where("user_id =?", u.ID).Delete(&UserMsg{}).Error
	if err != nil {
		return err
	}
	err = db.DB.Model(MsgSubscribeUser{}).Where("user_id =?", u.ID).Delete(&MsgSubscribeUser{}).Error
	if err != nil {
		return err
	}
	err = db.DB.Model(UserSetting{}).Where("user_id =?", u.ID).Delete(&UserSetting{}).Error
	if err != nil {
		return err
	}
	return err
}

func (u *User) ValidateOldPassword(password string) (bool, error) {
	oldPassword, err := encrypt.StringDecrypt(u.Password)
	if err != nil {
		return false, err
	}
	if oldPassword != password {
		return false, err
	}
	return true, err
}
