package controller

import (
	"github.com/ClusterOperator/ClusterOperator/pkg/dto"
	"github.com/ClusterOperator/ClusterOperator/pkg/service"
	"github.com/kataras/iris/v12/context"
)

type GradeController struct {
	Ctx          context.Context
	GradeService service.GradeService
}

func NewGradeController() *GradeController {
	return &GradeController{
		GradeService: service.NewGradeService(),
	}
}

func (g GradeController) GetBy(clusterName string) (*dto.ClusterGrade, error) {
	return g.GradeService.GetGrade(clusterName)
}
