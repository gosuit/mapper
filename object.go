package mapper

import (
	"fmt"
	"maps"
	"net/url"
	"reflect"
	"slices"
	"time"
)

const (
	mapTag     = "map"
	mapFromTag = "map-from"
	mapToTag   = "map-to"
)

var specificTypes = []reflect.Type{
	reflect.TypeFor[time.Time](),
	reflect.TypeFor[time.Location](),
	reflect.TypeFor[time.Duration](),
	reflect.TypeFor[url.URL](),
}

type parsedObject struct {
	setters map[string]setter
	getters map[string]getter
}

func parseObject(obj reflect.Type) *parsedObject {
	from, to := getPaths(obj, make([]int, 0), "")

	parsed := &parsedObject{
		setters: make(map[string]setter),
		getters: make(map[string]getter),
	}

	for k, v := range from {
		parsed.setters[k] = getSetter(v)
	}

	for k, v := range to {
		parsed.getters[k] = getGetter(v)
	}

	return parsed
}

func getPaths(obj reflect.Type, basePath []int, baseKey string) (map[string][]int, map[string][]int) {
	from := make(map[string][]int)
	to := make(map[string][]int)

	for i := range obj.NumField() {
		field := obj.Field(i)
		var fromTag string
		var toTag string

		commonTag, ok := field.Tag.Lookup(mapTag)
		if !ok {
			fromTag, ok = field.Tag.Lookup(mapFromTag)
			if !ok {
				fromTag = "-"
			}

			toTag, ok = field.Tag.Lookup(mapToTag)
			if !ok {
				toTag = "-"
			}

			if fromTag == "-" && toTag == "-" {
				continue
			}
		} else if commonTag != "-" {
			fromTag = commonTag
			toTag = commonTag
		} else {
			continue
		}

		path := make([]int, 0)

		if len(basePath) != 0 {
			path = append(basePath, i)
		} else {
			path = append(path, i)
		}

		if fromTag != "-" {
			if baseKey != "" {
				fromTag = baseKey + "." + fromTag
			}

			if field.Type.Kind() != reflect.Struct || slices.Contains(specificTypes, field.Type) {
				from[fromTag] = path
			} else {
				toAdd, _ := getPaths(field.Type, path, fromTag)

				maps.Copy(from, toAdd)
			}
		}

		if toTag != "-" {
			if baseKey != "" {
				toTag = baseKey + "." + toTag
			}

			if field.Type.Kind() != reflect.Struct || slices.Contains(specificTypes, field.Type) {
				to[toTag] = path
			} else {
				_, toAdd := getPaths(field.Type, path, toTag)

				maps.Copy(to, toAdd)
			}
		}
	}

	return from, to
}

type setter = func(model reflect.Value, value reflect.Value) error
type getter = func(reflect.Value) reflect.Value

func getSetter(indexPath []int) setter {
	base := getSetterBase(indexPath)

	setterIn := []reflect.Type{reflect.TypeFor[reflect.Value](), reflect.TypeFor[reflect.Value]()}
	setterOut := []reflect.Type{reflect.TypeFor[error]()}
	setterType := reflect.FuncOf(setterIn, setterOut, false)

	return reflect.MakeFunc(setterType, base).Interface().(setter)
}

func getGetter(indexPath []int) getter {
	base := getGetterBase(indexPath)

	getterIn := []reflect.Type{reflect.TypeFor[reflect.Value]()}
	getterOut := []reflect.Type{reflect.TypeFor[reflect.Value]()}
	getterType := reflect.FuncOf(getterIn, getterOut, false)

	return reflect.MakeFunc(getterType, base).Interface().(getter)
}

func getSetterBase(indexPath []int) fnBase {
	return func(args []reflect.Value) (results []reflect.Value) {
		model := args[0].Interface().(reflect.Value)
		value := args[1].Interface().(reflect.Value)

		var field reflect.Value
		var err error
		name := ""

		for i := range indexPath {
			if i == 0 {
				field = model.Field(indexPath[i])
				name = model.Type().Field(indexPath[i]).Name
			} else {
				name += "." + field.Type().Field(indexPath[i]).Name
				field = field.Field(indexPath[i])
			}
		}

		if value.IsValid() && value.CanConvert(field.Type()) && value.Kind() == field.Kind() {
			field.Set(value.Convert(field.Type()))
		} else {
			err = fmt.Errorf("invalid value for field '%s'. ", name)
		}

		if err != nil {
			results = append(results, reflect.ValueOf(err))
		} else {
			results = append(results, reflect.Zero(reflect.TypeOf((*error)(nil)).Elem()))
		}

		return results
	}
}

func getGetterBase(indexPath []int) fnBase {
	return func(args []reflect.Value) (results []reflect.Value) {
		model := args[0].Interface().(reflect.Value)

		var result reflect.Value

		for i := range indexPath {
			if i == 0 {
				result = model.Field(indexPath[i])
			} else {
				result = result.Field(indexPath[i])
			}
		}

		results = append(results, reflect.ValueOf(result))

		return results
	}
}
