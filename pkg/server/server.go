package server

import (
	"fmt"

	"github.com/ClusterOperator/ClusterOperator/pkg/config"
	"github.com/ClusterOperator/ClusterOperator/pkg/cron"
	"github.com/ClusterOperator/ClusterOperator/pkg/data"
	"github.com/ClusterOperator/ClusterOperator/pkg/db"
	"github.com/ClusterOperator/ClusterOperator/pkg/logger"
	"github.com/ClusterOperator/ClusterOperator/pkg/migrate"
	"github.com/ClusterOperator/ClusterOperator/pkg/plugin"
	"github.com/ClusterOperator/ClusterOperator/pkg/plugin/xpack"
	"github.com/ClusterOperator/ClusterOperator/pkg/router"
	"github.com/ClusterOperator/ClusterOperator/pkg/server/hook"
	"github.com/kataras/iris/v12"
	"github.com/spf13/viper"
)

type Phase interface {
	Init() error
	PhaseName() string
}

func Phases() []Phase {
	return []Phase{
		&db.InitDBPhase{
			Host:         viper.GetString("db.host"),
			Port:         viper.GetInt("db.port"),
			Name:         viper.GetString("db.name"),
			User:         viper.GetString("db.user"),
			Password:     viper.GetString("db.password"),
			MaxOpenConns: viper.GetInt("db.max_open_conns"),
			MaxIdleConns: viper.GetInt("db.max_idle_conns"),
		},
		&migrate.InitMigrateDBPhase{
			Host:     viper.GetString("db.host"),
			Port:     viper.GetInt("db.port"),
			Name:     viper.GetString("db.name"),
			User:     viper.GetString("db.user"),
			Password: viper.GetString("db.password"),
		},
		&data.InitDataPhase{},
		&plugin.InitPluginDBPhase{},
		&cron.InitCronPhase{
			Enable: viper.GetBool("cron.enable"),
		},
	}
}

func Start() error {
	config.Init()
	logger.Init()
	phases := Phases()
	for _, phase := range phases {
		if err := phase.Init(); err != nil {
			logger.Log.Errorf("start phase [%s] failed reason: %s",
				phase.PhaseName(), err.Error())
			return err
		}
		logger.Log.Infof("start phase [%s] success", phase.PhaseName())
	}
	s := router.Server()
	// load xpack plugin must behead router init,so can not create an phase for it.
	if err := xpack.LoadXpackPlugin(); err != nil {
		logger.Log.Error("xpack load failed, xpack can not be registered")
	}
	bind := fmt.Sprintf("%s:%d",
		viper.GetString("bind.host"),
		viper.GetInt("bind.port"))

	if err := hook.BeforeApplicationStart.Run(); err != nil {
		return err
	}
	return s.Run(iris.Addr(bind))
}
