package service

import (
	"github.com/ClusterOperator/ClusterOperator/pkg/constant"
	"github.com/ClusterOperator/ClusterOperator/pkg/db"
	"github.com/ClusterOperator/ClusterOperator/pkg/dto"
	"github.com/ClusterOperator/ClusterOperator/pkg/logger"
	"github.com/ClusterOperator/ClusterOperator/pkg/model"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/msg"
	"github.com/jinzhu/gorm"
	"reflect"
	"time"
)

type MsgAccountService interface {
	GetByName(name string) (dto.MsgAccountDTO, error)
	CreateOrUpdate(msgDTO dto.MsgAccountDTO) (dto.MsgAccountDTO, error)
	Verify(msgDTO dto.MsgAccountDTO) error
}

type msgAccountService struct {
	MsgService MsgService
}

func NewMsgAccountService() MsgAccountService {
	return &msgAccountService{
		MsgService: NewMsgService(),
	}
}

func (m msgAccountService) GetByName(name string) (dto.MsgAccountDTO, error) {
	var msgAccountDTO dto.MsgAccountDTO
	var msgAccount model.MsgAccount
	msgAccount.Name = name
	err := db.DB.Model(&model.MsgAccount{}).Where("name = ?", name).First(&msgAccount).Error
	if err != nil && !gorm.IsRecordNotFoundError(err) {
		return msgAccountDTO, nil
	}
	msgAccountDTO = dto.CoverToDTO(msgAccount)
	return msgAccountDTO, nil
}

func (m msgAccountService) CreateOrUpdate(msgDTO dto.MsgAccountDTO) (dto.MsgAccountDTO, error) {

	var old model.MsgAccount
	err := db.DB.Model(&model.MsgAccount{}).Where("name = ?", msgDTO.Name).First(&old).Error
	if err != nil && !gorm.IsRecordNotFoundError(err) {
		return msgDTO, err
	}
	mo := dto.CoverToModel(msgDTO)
	if reflect.DeepEqual(old, model.MsgAccount{}) {
		return msgDTO, db.DB.Create(&mo).Error
	} else {
		mo.ID = old.ID
		return msgDTO, db.DB.Save(&mo).Error
	}
}

func (m msgAccountService) Verify(msgDTO dto.MsgAccountDTO) error {

	client, err := msg.NewMsgClient(msgDTO.Name, msgDTO.Config)
	if err != nil {
		return err
	}
	vars := msgDTO.Config.(map[string]interface{})
	testUser := vars["testUser"].(string)

	content := make(map[string]string)
	content["message"] = constant.TestMessage
	date := time.Now().Add(time.Hour * 8).Format("2006-01-02 15:04:05")
	content["date"] = date
	detail, err := GetMsgContent(constant.MsgTest, msgDTO.Name, content)
	if err != nil {
		return err
	}
	if err := client.Send([]string{testUser}, constant.MsgTitle[constant.MsgTest], []byte(detail)); err != nil {
		logger.Log.Infof("verify msg account error,%s", err.Error())
		return err
	}
	return nil
}
