package service

import (
	"fmt"
	"sync"

	"github.com/ClusterOperator/ClusterOperator/pkg/controller/condition"
	dbUtil "github.com/ClusterOperator/ClusterOperator/pkg/util/db"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/encrypt"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/nexus"

	"github.com/ClusterOperator/ClusterOperator/pkg/controller/page"
	"github.com/ClusterOperator/ClusterOperator/pkg/model/common"

	"github.com/ClusterOperator/ClusterOperator/pkg/constant"
	"github.com/ClusterOperator/ClusterOperator/pkg/db"
	"github.com/ClusterOperator/ClusterOperator/pkg/dto"
	"github.com/ClusterOperator/ClusterOperator/pkg/model"
	"github.com/ClusterOperator/ClusterOperator/pkg/repository"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/message"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/message/client"
	"github.com/jinzhu/gorm"
)

type SystemSettingService interface {
	Get(name string) (dto.SystemSetting, error)
	GetLocalIPs() ([]model.SystemRegistry, error)
	List() (dto.SystemSettingResult, error)
	Create(creation dto.SystemSettingCreate) ([]dto.SystemSetting, error)
	ListByTab(tabName string) (dto.SystemSettingResult, error)
	CheckSettingByType(tabName string, creation dto.SystemSettingCreate) error
	ListRegistry(conditions condition.Conditions) ([]dto.SystemRegistry, error)
	PageRegistry(num, size int, conditions condition.Conditions) (*page.Page, error)
	GetRegistryByID(id string) (dto.SystemRegistry, error)
	GetRegistryByArch(arch string) (dto.SystemRegistry, error)
	CreateRegistry(creation dto.SystemRegistryCreate) (*dto.SystemRegistry, error)
	UpdateRegistry(arch string, creation dto.SystemRegistryUpdate) (*dto.SystemRegistry, error)
	BatchRegistry(op dto.SystemRegistryBatchOp) error
	DeleteRegistry(id string) error
	ChangePassword(repo dto.RepoChangePassword) error
}

type systemSettingService struct {
	systemSettingRepo  repository.SystemSettingRepository
	systemRegistryRepo repository.SystemRegistryRepository
	userRepo           repository.UserRepository
}

func NewSystemSettingService() SystemSettingService {
	return &systemSettingService{
		systemSettingRepo:  repository.NewSystemSettingRepository(),
		systemRegistryRepo: repository.NewSystemRegistryRepository(),
		userRepo:           repository.NewUserRepository(),
	}
}

func (s systemSettingService) Get(key string) (dto.SystemSetting, error) {
	var systemSettingDTO dto.SystemSetting
	mo, err := s.systemSettingRepo.Get(key)
	if err != nil {
		return systemSettingDTO, err
	}
	systemSettingDTO.SystemSetting = mo
	return systemSettingDTO, err
}

func (s systemSettingService) List() (dto.SystemSettingResult, error) {
	var systemSettingResult dto.SystemSettingResult
	vars := make(map[string]string)
	mos, err := s.systemSettingRepo.List()
	if err != nil {
		return systemSettingResult, err
	}
	for _, mo := range mos {
		vars[mo.Key] = mo.Value
	}
	systemSettingResult.Vars = vars
	return systemSettingResult, err
}

func (s systemSettingService) ListByTab(tabName string) (dto.SystemSettingResult, error) {
	var systemSettingResult dto.SystemSettingResult
	vars := make(map[string]string)
	mos, err := s.systemSettingRepo.ListByTab(tabName)
	if err != nil {
		return systemSettingResult, err
	}
	for _, mo := range mos {
		vars[mo.Key] = mo.Value
	}
	if len(mos) > 0 {
		systemSettingResult.Tab = tabName
	}
	systemSettingResult.Vars = vars
	return systemSettingResult, err
}

func (s systemSettingService) Create(creation dto.SystemSettingCreate) ([]dto.SystemSetting, error) {

	var result []dto.SystemSetting
	for k, v := range creation.Vars {
		systemSetting, err := s.systemSettingRepo.Get(k)
		if err != nil {
			if gorm.IsRecordNotFoundError(err) {
				systemSetting.Key = k
				systemSetting.Value = v
				systemSetting.Tab = creation.Tab
				err := s.systemSettingRepo.Save(&systemSetting)
				if err != nil {
					return result, err
				}
				result = append(result, dto.SystemSetting{SystemSetting: systemSetting})
			} else {
				return result, err
			}
		} else if systemSetting.ID != "" {
			systemSetting.Value = v
			if systemSetting.Tab == "" {
				systemSetting.Tab = creation.Tab
			}
			err := s.systemSettingRepo.Save(&systemSetting)
			if err != nil {
				return result, err
			}
			result = append(result, dto.SystemSetting{SystemSetting: systemSetting})
		}
	}
	return result, nil
}

func (s systemSettingService) GetLocalIPs() ([]model.SystemRegistry, error) {
	var sysRepo []model.SystemRegistry
	if err := db.DB.Find(&sysRepo).Error; err != nil {
		return sysRepo, fmt.Errorf("can't found repo from system registry, err %s", err.Error())
	}
	return sysRepo, nil
}

func (s systemSettingService) CheckSettingByType(tabName string, creation dto.SystemSettingCreate) error {

	vars := make(map[string]interface{})
	for k, value := range creation.Vars {
		vars[k] = value
	}
	if tabName == constant.Email {
		vars["type"] = constant.Email
		vars["RECEIVERS"] = vars["SMTP_TEST_USER"]
		vars["TITLE"] = "KubeOperator测试邮件"
		vars["CONTENT"] = "此邮件由 KubeOperator 发送，用于测试邮件发送，请勿回复"
	} else if tabName == constant.DingTalk {
		vars["type"] = constant.DingTalk
		vars["RECEIVERS"] = vars["DING_TALK_TEST_USER"]
		vars["TITLE"] = "KubeOperator测试消息"
		vars["CONTENT"] = "此邮件由 KubeOperator 发送，用于测试消息发送"
	} else if tabName == constant.WorkWeiXin {
		vars["type"] = constant.WorkWeiXin
		vars["CONTENT"] = "此邮件由 KubeOperator 发送，用于测试消息发送"
		vars["RECEIVERS"] = vars["WORK_WEIXIN_TEST_USER"]
	}
	c, err := message.NewMessageClient(vars)
	if err != nil {
		return err
	}
	if tabName == constant.WorkWeiXin {
		token, err := client.GetToken(vars)
		if err != nil {
			return err
		}
		vars["TOKEN"] = token
	}
	err = c.SendMessage(vars)
	if err != nil {
		return err
	}
	return nil
}

func (s systemSettingService) ListRegistry(conditions condition.Conditions) ([]dto.SystemRegistry, error) {
	var systemRegistryDto []dto.SystemRegistry
	var mos []model.SystemRegistry
	d := db.DB.Model(model.SystemRegistry{})
	if err := dbUtil.WithConditions(&d, model.User{}, conditions); err != nil {
		return nil, err
	}
	if err := d.Order("architecture").
		Find(&mos).Error; err != nil {
		return nil, err
	}
	for _, mo := range mos {
		systemRegistryDto = append(systemRegistryDto, dto.SystemRegistry{
			SystemRegistry: mo,
		})
	}
	return systemRegistryDto, nil
}

func (s systemSettingService) GetRegistryByID(id string) (dto.SystemRegistry, error) {
	r, err := s.systemRegistryRepo.Get(id)
	if err != nil {
		return dto.SystemRegistry{}, err
	}
	pass, _ := encrypt.StringDecrypt(r.NexusPassword)
	systemRegistryDto := dto.SystemRegistry{
		SystemRegistry: model.SystemRegistry{
			ID:                 r.ID,
			Hostname:           r.Hostname,
			Protocol:           r.Protocol,
			Architecture:       r.Architecture,
			RepoPort:           r.RepoPort,
			RegistryPort:       r.RegistryPort,
			RegistryHostedPort: r.RegistryHostedPort,
			NexusUser:          r.NexusUser,
			NexusPassword:      pass,
		},
	}
	return systemRegistryDto, nil
}

func (s systemSettingService) GetRegistryByArch(arch string) (dto.SystemRegistry, error) {
	r, err := s.systemRegistryRepo.GetByArch(arch)
	if err != nil {
		return dto.SystemRegistry{}, err
	}
	systemRegistryDto := dto.SystemRegistry{
		SystemRegistry: model.SystemRegistry{
			ID:           r.ID,
			Hostname:     r.Hostname,
			Protocol:     r.Protocol,
			Architecture: r.Architecture,
		},
	}
	return systemRegistryDto, nil
}

func (s systemSettingService) PageRegistry(num, size int, conditions condition.Conditions) (*page.Page, error) {
	var (
		p                 page.Page
		systemRegistryDto []dto.SystemRegistry
		mos               []model.SystemRegistry
	)

	d := db.DB.Model(model.SystemRegistry{})
	if err := dbUtil.WithConditions(&d, model.SystemRegistry{}, conditions); err != nil {
		return nil, err
	}
	if err := d.
		Count(&p.Total).
		Order("architecture").
		Offset((num - 1) * size).
		Limit(size).
		Find(&mos).Error; err != nil {
		return nil, err
	}
	var wg sync.WaitGroup
	for i := 0; i < len(mos); i++ {
		wg.Add(1)
		itemDto := dto.SystemRegistry{SystemRegistry: mos[i]}
		go func(repo model.SystemRegistry) {
			pass, err := encrypt.StringDecrypt(repo.NexusPassword)
			if err != nil {
				itemDto.Status = constant.StatusFailed
				itemDto.Message = err.Error()
				systemRegistryDto = append(systemRegistryDto, itemDto)
				wg.Done()
				return
			}
			if err := nexus.CheckConn(
				repo.NexusUser,
				pass,
				fmt.Sprintf("%s://%s:%d", repo.Protocol, repo.Hostname, repo.RepoPort),
			); err != nil {
				itemDto.Status = constant.StatusFailed
				itemDto.Message = err.Error()
			} else {
				itemDto.Status = constant.StatusSuccess
			}
			systemRegistryDto = append(systemRegistryDto, itemDto)
			wg.Done()
		}(mos[i])
	}
	wg.Wait()
	p.Items = systemRegistryDto
	return &p, nil
}

func (s systemSettingService) CreateRegistry(creation dto.SystemRegistryCreate) (*dto.SystemRegistry, error) {
	password, err := encrypt.StringEncrypt(creation.NexusPassword)
	if err != nil {
		return nil, err
	}
	systemRegistry := model.SystemRegistry{
		Architecture:       creation.Architecture,
		Protocol:           creation.Protocol,
		Hostname:           creation.Hostname,
		RepoPort:           creation.RepoPort,
		RegistryPort:       creation.RegistryPort,
		RegistryHostedPort: creation.RegistryHostedPort,
		NexusUser:          creation.NexusUser,
		NexusPassword:      password,
	}
	if err := s.systemRegistryRepo.Save(&systemRegistry); err != nil {
		return nil, err
	}
	return &dto.SystemRegistry{SystemRegistry: systemRegistry}, nil
}

func (s systemSettingService) UpdateRegistry(arch string, creation dto.SystemRegistryUpdate) (*dto.SystemRegistry, error) {
	systemRegistry := model.SystemRegistry{
		ID:                 creation.ID,
		Architecture:       arch,
		Protocol:           creation.Protocol,
		Hostname:           creation.Hostname,
		RepoPort:           creation.RepoPort,
		RegistryPort:       creation.RegistryPort,
		RegistryHostedPort: creation.RegistryHostedPort,
	}
	if err := db.DB.Model(&model.SystemRegistry{}).Update(&systemRegistry).Error; err != nil {
		return nil, err
	}
	return &dto.SystemRegistry{SystemRegistry: systemRegistry}, nil
}

func (s systemSettingService) BatchRegistry(op dto.SystemRegistryBatchOp) error {
	var deleteItems []model.SystemRegistry
	for _, item := range op.Items {
		deleteItems = append(deleteItems, model.SystemRegistry{
			BaseModel:    common.BaseModel{},
			ID:           item.ID,
			Architecture: item.Architecture,
		})
	}
	err := s.systemRegistryRepo.Batch(op.Operation, deleteItems)
	if err != nil {
		return err
	}
	return nil
}

func (s systemSettingService) DeleteRegistry(id string) error {
	err := s.systemRegistryRepo.Delete(id)
	if err != nil {
		return err
	}
	return nil
}

func (u *systemSettingService) ChangePassword(ch dto.RepoChangePassword) error {
	password, err := encrypt.StringEncrypt(ch.NexusPassword)
	if err != nil {
		return err
	}
	if err := db.DB.Model(&model.SystemRegistry{}).Where("id = ?", ch.ID).
		Update(map[string]interface{}{"nexus_password": password, "nexus_user": ch.NexusUser}).Error; err != nil {
		return err
	}
	return nil
}
