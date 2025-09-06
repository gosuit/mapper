package mapper

import (
	"errors"
	"reflect"
	"sync"
)

type Mapper interface {
	Map(from any, to any) error
}

func New() Mapper {
	return &mapper{
		converters: make(map[reflect.Type]map[reflect.Type]converter),
		objects:    make(map[reflect.Type]*parsedObject),
	}
}

type mapper struct {
	converters map[reflect.Type]map[reflect.Type]converter
	objects    map[reflect.Type]*parsedObject
	mu         sync.Mutex
}

func (m *mapper) Map(from any, to any) error {
	fromValue, toValue, err := m.parseInput(from, to)
	if err != nil {
		return err
	}

	m.registerObject(fromValue.Type())
	m.registerObject(toValue.Type())

	if err := m.registerPair(fromValue.Type(), toValue.Type()); err != nil {
		return err
	}

	return m.mapObjects(fromValue, toValue)
}

func (m *mapper) parseInput(from any, to any) (reflect.Value, reflect.Value, error) {
	fromValue := reflect.ValueOf(from)

	if fromValue.Kind() != reflect.Pointer {
		return reflect.Value{}, reflect.Value{}, errors.New("from must be pointer to struct")
	}

	fromValue = fromValue.Elem()

	if fromValue.Kind() != reflect.Struct {
		return reflect.Value{}, reflect.Value{}, errors.New("from must be pointer to struct")
	}

	toValue := reflect.ValueOf(to)

	if toValue.Kind() != reflect.Pointer {
		return reflect.Value{}, reflect.Value{}, errors.New("to must be pointer to struct")
	}

	toValue = toValue.Elem()

	if toValue.Kind() != reflect.Struct {
		return reflect.Value{}, reflect.Value{}, errors.New("to must be pointer to struct")
	}

	return fromValue, toValue, nil
}

func (m *mapper) registerObject(obj reflect.Type) {
	_, ok := m.objects[obj]
	if ok {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok = m.objects[obj]
	if ok {
		return
	}

	m.objects[obj] = parseObject(obj)
}

func (m *mapper) registerPair(from reflect.Type, to reflect.Type) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.converters[from]
	if !ok {
		m.converters[from] = make(map[reflect.Type]converter)
	}

	if _, ok := m.converters[from][to]; ok {
		return nil
	}

	fn, err := m.getConverter(from, to)
	if err != nil {
		return err
	}

	m.converters[from][to] = fn

	return nil
}

func (m *mapper) mapObjects(from reflect.Value, to reflect.Value) error {
	return m.converters[from.Type()][to.Type()](from, to)
}
