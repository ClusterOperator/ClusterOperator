package controller

import (
	"github.com/ClusterOperator/ClusterOperator/pkg/constant"
	"github.com/ClusterOperator/ClusterOperator/pkg/controller/condition"
	"github.com/ClusterOperator/ClusterOperator/pkg/dto"
	"github.com/ClusterOperator/ClusterOperator/pkg/service"
	"github.com/kataras/iris/v12/context"
)

type UserMsgController struct {
	Ctx            context.Context
	UserMsgService service.UserMsgService
}

func NewUserMsgController() *UserMsgController {
	return &UserMsgController{
		UserMsgService: service.NewUserMsgService(),
	}
}

func (u *UserMsgController) Get() (dto.UserMsgResponse, error) {
	p, _ := u.Ctx.Values().GetBool("page")
	sessionUser := u.Ctx.Values().Get("user")
	user, _ := sessionUser.(dto.SessionUser)
	if p {
		num, _ := u.Ctx.Values().GetInt(constant.PageNumQueryKey)
		size, _ := u.Ctx.Values().GetInt(constant.PageSizeQueryKey)
		return u.UserMsgService.PageLocalMsg(num, size, user, condition.TODO())
	}
	return dto.UserMsgResponse{}, nil
}

func (u *UserMsgController) PostReadBy(msgID string) error {
	sessionUser := u.Ctx.Values().Get("user")
	user, _ := sessionUser.(dto.SessionUser)
	return u.UserMsgService.UpdateLocalMsg(msgID, user)
}

func (u *UserMsgController) PostReadAll() error {
	sessionUser := u.Ctx.Values().Get("user")
	user, _ := sessionUser.(dto.SessionUser)
	return u.UserMsgService.MarkAllRead(user)
}
