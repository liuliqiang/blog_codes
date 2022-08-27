package try01

import (
	"github.com/liuliqiang/blog-demos/design-pattern/creational/simple-factory/try01/service"
)

func main() {
	cfg := CloudConfig{}
	cloudManager := service.NewManager(cfg)

	cloudManager.DeleteVM("vm-1")
}
