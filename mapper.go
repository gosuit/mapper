package mapper

type Mapper interface {
	Map(from any, to any) error
}

func New() Mapper {
	return &mapper{}
}

type mapper struct{}

func (m *mapper) Map(from any, to any) error {
	return nil
}
