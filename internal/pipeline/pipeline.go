package pipeline

type Pipeline interface {
	Start() error
	Stop()
}
