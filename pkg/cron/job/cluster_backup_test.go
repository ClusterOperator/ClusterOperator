package job

import (
	"testing"

	"github.com/ClusterOperator/ClusterOperator/pkg/config"
	"github.com/ClusterOperator/ClusterOperator/pkg/db"
	"github.com/ClusterOperator/ClusterOperator/pkg/logger"
	"github.com/spf13/viper"
)

func TestCLusterBackup(t *testing.T) {
	config.Init()
	dbi := db.InitDBPhase{
		Host:     viper.GetString("db.host"),
		Port:     viper.GetInt("db.port"),
		Name:     viper.GetString("db.name"),
		User:     viper.GetString("db.user"),
		Password: viper.GetString("db.password"),
	}
	err := dbi.Init()
	if err != nil {
		logger.Log.Fatal(err)
	}
	j := NewClusterBackup()
	j.Run()
	select {}
}
