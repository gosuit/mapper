package mapper

import (
	"errors"
	"reflect"
	"sync"
)

type Builder[T any] interface {
	Build(from any) (T, error)
}

func NewBuilder[T any](constructor any) Builder[T] {
	return &builder[T]{
		objects: make(map[reflect.Type]*parsedBuildObject),
	}
}

type builder[T any] struct {
	objects     map[reflect.Type]*parsedBuildObject
	constructor reflect.Value
	mu          sync.Mutex
}

func (b *builder[T]) Build(from any) (T, error) {
	var zeroT T

	fromValue, err := b.parseInput(from)
	if err != nil {
		return zeroT, err
	}

	b.registerObject(fromValue.Type())

	return b.build(fromValue)
}

func (b *builder[T]) parseInput(from any) (reflect.Value, error) {
	fromValue := reflect.ValueOf(from)

	if fromValue.Kind() != reflect.Pointer {
		return reflect.Value{}, errors.New("from must be pointer to struct")
	}

	fromValue = fromValue.Elem()

	if fromValue.Kind() != reflect.Struct {
		return reflect.Value{}, errors.New("from must be pointer to struct")
	}

	return fromValue, nil
}

func (b *builder[T]) registerObject(obj reflect.Type) {
	_, ok := b.objects[obj]
	if ok {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	_, ok = b.objects[obj]
	if ok {
		return
	}

	b.objects[obj] = parseBuildObject(obj)
}

func (b *builder[T]) build(from reflect.Value) (T, error) {
	var zeroT T

	return zeroT, nil
}
