package mapper

import "reflect"

type converter = func(reflect.Value, reflect.Value) error

func (m *mapper) getConverter(from reflect.Type, to reflect.Type) (converter, error) {
	return nil, nil
}
