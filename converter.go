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

	for k, mapSource := range parsedTo.mapFrom {
		getter, ok := parsedFrom.getters[k]
		if !ok {
			methodGetter, ok := parsedFrom.methods[mapSource.funcGetter]
			if !ok {
				return nil, errors.New("getter not found")
			}

			conversions = append(conversions, func(from reflect.Value, to reflect.Value) error {
				getter := methodGetter(from)
				values := getter.Call(make([]reflect.Value, 0))

				if len(values) < 1 {
					return errors.New("invalid output")
				}

				return mapSource.setter(to, values[0])
			})

			continue
		}

		conversions = append(conversions, func(from reflect.Value, to reflect.Value) error {
			return mapSource.setter(to, getter(from))
		})
	}

	for k, mapDestination := range parsedFrom.mapTo {
		setter, ok := parsedTo.setters[k]
		if !ok {
			methodSetter, ok := parsedTo.methods[mapDestination.funcSetter]
			if !ok {
				return nil, errors.New("setter not found")
			}

			conversions = append(conversions, func(from reflect.Value, to reflect.Value) error {
				setter := methodSetter(to)
				value := mapDestination.getter(from)

				setter.Call([]reflect.Value{value})

				return nil
			})

			continue
		}

		conversions = append(conversions, func(from reflect.Value, to reflect.Value) error {
			return setter(to, mapDestination.getter(from))
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
