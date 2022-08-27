package try01

type Vm struct {
}

type Manager interface {
	CreateVm(vm Vm) (Vm, error)
	DeleteVM(vmID string) error
}
