package plugins

type Input interface {
	SampleConfig() string

	Description() string

	Gather(Accumulator) error
}
