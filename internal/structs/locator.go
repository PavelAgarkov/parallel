package structs

type ExampleLocator struct {
	Sn string
}

func (ml ExampleLocator) ServiceName() string {
	return "ExampleLocator"
}
