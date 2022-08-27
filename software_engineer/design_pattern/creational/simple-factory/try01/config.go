package try01

type CloudType int

const (
	GCP CloudType = iota
	DO
	Vultr
)

type CloudConfig struct {
	CloudType CloudType
	Token     string
	AuthFile  string
}
