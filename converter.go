package mapper

import (
	"errors"
	"reflect"
)

type converter = func(from reflect.Value, to reflect.Value) error

func (m *mapper) getConverter(from reflect.Type, to reflect.Type) (converter, error) {
	conversions, err := m.getPairConversions(from, to)
	if err != nil {
		return nil, err
	}

	base := getConverterBase(conversions)

	converterIn := []reflect.Type{reflect.TypeFor[reflect.Value](), reflect.TypeFor[reflect.Value]()}
	converterOut := []reflect.Type{reflect.TypeFor[error]()}
	converterType := reflect.FuncOf(converterIn, converterOut, false)

	return reflect.MakeFunc(converterType, base).Interface().(converter), nil
}

func (m *mapper) getPairConversions(from reflect.Type, to reflect.Type) ([]converter, error) {
	parsedFrom := m.objects[from]
	parsedTo := m.objects[to]

	conversions := make([]converter, 0)

	for k, setter := range parsedTo.mapFrom {
		getter, ok := parsedFrom.getters[k]
		if !ok {
			return nil, errors.New("getter not found")
		}

		conversions = append(conversions, func(from reflect.Value, to reflect.Value) error {
			return setter(to, getter(from))
		})
	}

	for k, getter := range parsedFrom.mapTo {
		setter, ok := parsedTo.setters[k]
		if !ok {
			return nil, errors.New("setter not found")
		}

		conversions = append(conversions, func(from reflect.Value, to reflect.Value) error {
			return setter(to, getter(from))
		})
	}

	return conversions, nil
}

func getConverterBase(conversions []converter) fnBase {
	return func(args []reflect.Value) (results []reflect.Value) {
		from := args[0].Interface().(reflect.Value)
		to := args[1].Interface().(reflect.Value)

		var err error

		for _, conv := range conversions {
			if convErr := conv(from, to); convErr != nil {
				err = convErr
				break
			}
		}

		if err != nil {
			results = append(results, reflect.ValueOf(err))
		} else {
			results = append(results, reflect.Zero(reflect.TypeOf((*error)(nil)).Elem()))
		}

		return results
	}
}
