package job

import (
	"sync"

	"github.com/ClusterOperator/ClusterOperator/pkg/constant"
	"github.com/ClusterOperator/ClusterOperator/pkg/db"
	"github.com/ClusterOperator/ClusterOperator/pkg/logger"
	"github.com/ClusterOperator/ClusterOperator/pkg/model"
	"github.com/ClusterOperator/ClusterOperator/pkg/service"
)

type RefreshHostInfo struct {
	hostService service.HostService
}

func NewRefreshHostInfo() *RefreshHostInfo {
	return &RefreshHostInfo{
		hostService: service.NewHostService(),
	}
}

func (r *RefreshHostInfo) Run() {
	var hosts []model.Host
	var wg sync.WaitGroup
	sem := make(chan struct{}, 2) // 信号量
	db.DB.Find(&hosts)
	for _, host := range hosts {
		if host.Status == constant.StatusCreating || host.Status == constant.StatusInitializing || host.Status == constant.StatusSynchronizing {
			continue
		}
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			_, err := r.hostService.Sync(name)
			if err != nil {
				logger.Log.Errorf("gather host info error: %s", err.Error())
			}
		}(host.Name)
	}
	wg.Wait()
}
