package service

import (
	"github.com/liuliqiang/blog-demos/design-pattern/creational/simple-factory/try01"
)

type ManagerFactory interface {
	NewManager(config try01.CloudConfig) try01.Manager
}

func (f *ManagerFactory) NewManager(config try01.CloudConfig) try01.Manager {
	switch config.CloudType {
	case config.GCP:
		return NewGcpManager(config)
	case config.DO:
		return NewDoManager(config)
	case config.Vultr:
		return NewVultrManager(config)
	}
}
