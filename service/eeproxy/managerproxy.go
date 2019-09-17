package eeproxy

const (
	managerVERSION = 100
	managerRUN     = 101
	managerKILL    = 102
	managerEND     = 103
)

type managerVersion struct {
	Version uint16
	Type    string
}
