package service

import (
	"github.com/liuliqiang/blog-demos/design-pattern/creational/simple-factory/try01"
)

type gcpManger struct {
	cfg try01.CloudConfig
}

func NewGcpManager(cfg try01.CloudConfig) Manager {
	return &gcpManger{
		cfg: cfg,
	}
}

func (g *gcpManger) CreateVm(vm try01.Vm) (try01.Vm, error) {
	panic("implement me")
}

func (g *gcpManger) DeleteVM(vmID string) error {
	panic("implement me")
}
