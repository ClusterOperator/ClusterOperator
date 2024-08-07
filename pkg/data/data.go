package data

import (
	"os"

	"github.com/ClusterOperator/ClusterOperator/pkg/constant"
	"github.com/ClusterOperator/ClusterOperator/pkg/util/file"
)

var initDirs = []string{
	constant.DefaultDataDir,
}

const phaseName = "create data dir"

type InitDataPhase struct{}

func (i *InitDataPhase) Init() error {
	for _, d := range initDirs {
		if !file.Exists(d) {
			err := os.MkdirAll(d, 0755)
			if err != nil {
				return err
			}
		}
	}
	return nil

}

func (i *InitDataPhase) PhaseName() string {
	return phaseName
}
