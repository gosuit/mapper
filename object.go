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
	tagPlug    = "!plug!"
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
	methods map[string]methodGetter

	mapFrom map[string]mapSource
	mapTo   map[string]mapDestination
}

type mapSource struct {
	set        setter
	funcGetter string
}

type mapDestination struct {
	get        getter
	funcSetter string
}

func parseObject(obj reflect.Type) *parsedObject {
	parsed := &parsedObject{
		setters: make(map[string]setter),
		getters: make(map[string]getter),
		methods: make(map[string]methodGetter),

		mapFrom: make(map[string]mapSource),
		mapTo:   make(map[string]mapDestination),
	}

	fields := getPaths(obj, make([]int, 0), "")

	for k, v := range fields {
		parsed.setters[k] = getSetter(v)
		parsed.getters[k] = getGetter(v)
	}

	methods := getMethods(obj)

	for k, v := range methods {
		parsed.methods[k] = getMethodGetter(v)
	}

	mapSrc, mapDst := getMappingMeta(obj, []int{}, "")

	for _, src := range mapSrc {
		parsed.mapFrom[src.fieldSrc] = mapSource{
			set:        getSetter(src.indexPath),
			funcGetter: src.funcSrc,
		}
	}

	for _, src := range mapDst {
		parsed.mapTo[src.fieldDst] = mapDestination{
			get:        getGetter(src.indexPath),
			funcSetter: src.funcDst,
		}
	}

	return parsed
}

func getPaths(obj reflect.Type, basePath []int, baseKey string) map[string][]int {
	result := make(map[string][]int)

	for i := range obj.NumField() {
		field := obj.Field(i)

		key := field.Name

		if baseKey != "" {
			key = baseKey + "." + key
		}

		path := make([]int, 0)

		if len(basePath) != 0 {
			path = append(basePath, i)
		} else {
			path = append(path, i)
		}

		if field.Type.Kind() != reflect.Struct || slices.Contains(specificTypes, field.Type) {
			result[key] = path
		} else {
			toAdd := getPaths(field.Type, path, key)

			maps.Copy(result, toAdd)
		}
	}

	return result
}

func getMethods(obj reflect.Type) map[string]int {
	result := make(map[string]int)

	for i := range obj.NumMethod() {
		result[obj.Method(i).Name] = obj.Method(i).Index
	}

	return result
}

func getMappingMeta(obj reflect.Type, basePath []int, baseKey string) ([]rawMapSource, []rawMapDestination) {
	src := make([]rawMapSource, 0)
	dst := make([]rawMapDestination, 0)

	for i := range obj.NumField() {
		field := obj.Field(i)

		var fromTag string
		var toTag string

		commonTag, ok := field.Tag.Lookup(mapTag)
		if !ok {
			fromTag, ok = field.Tag.Lookup(mapFromTag)
			if !ok {
				fromTag = tagPlug
			}

			toTag, ok = field.Tag.Lookup(mapToTag)
			if !ok {
				toTag = tagPlug
			}
		} else if commonTag == "-" {
			continue
		} else {
			fromTag = commonTag
			toTag = commonTag
		}

		path := make([]int, 0)

		if len(basePath) != 0 {
			path = append(basePath, i)
		} else {
			path = append(path, i)
		}

		if fromTag != "-" {
			funcTag := fromTag

			if fromTag != tagPlug && baseKey != "" {
				fromTag = baseKey + "." + fromTag
			}

			if field.Type.Kind() != reflect.Struct || slices.Contains(specificTypes, field.Type) {
				if fromTag != tagPlug {
					src = append(src, rawMapSource{
						indexPath: path,
						fieldSrc:  fromTag,
						funcSrc:   funcTag,
					})
				}
			} else {
				var baseFromTag string

				if fromTag == tagPlug {
					baseFromTag = ""
				} else {
					baseFromTag = fromTag
				}

				from, _ := getMappingMeta(field.Type, path, baseFromTag)

				src = append(src, from...)
			}

			if toTag != "-" {
				funcTag := toTag

				if toTag != tagPlug && baseKey != "" {
					toTag = baseKey + "." + toTag
				}

				if field.Type.Kind() != reflect.Struct || slices.Contains(specificTypes, field.Type) {
					if toTag != tagPlug {
						dst = append(dst, rawMapDestination{
							indexPath: path,
							fieldDst:  toTag,
							funcDst:   funcTag,
						})
					}
				} else {
					var baseToTag string

					if fromTag == tagPlug {
						baseToTag = ""
					} else {
						baseToTag = toTag
					}

					_, to := getMappingMeta(field.Type, path, baseToTag)

					dst = append(dst, to...)
				}
			}
		}
	}

	return src, dst
}

type rawMapSource struct {
	indexPath []int
	fieldSrc  string
	funcSrc   string
}

type rawMapDestination struct {
	indexPath []int
	fieldDst  string
	funcDst   string
}

type setter = func(model reflect.Value, value reflect.Value) error
type getter = func(reflect.Value) reflect.Value
type methodGetter = func(reflect.Value) reflect.Value

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

func getMethodGetter(index int) methodGetter {
	base := getMethodGetterBase(index)

	getterIn := []reflect.Type{reflect.TypeFor[reflect.Value]()}
	getterOut := []reflect.Type{reflect.TypeFor[reflect.Value]()}
	getterType := reflect.FuncOf(getterIn, getterOut, false)

	return reflect.MakeFunc(getterType, base).Interface().(methodGetter)
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

func getMethodGetterBase(index int) fnBase {
	return func(args []reflect.Value) (results []reflect.Value) {
		model := args[0].Interface().(reflect.Value)

		results = append(results, reflect.ValueOf(model.Method(index)))

		return results
	}
}
