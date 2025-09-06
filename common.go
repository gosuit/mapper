package mapper

import "reflect"

type fnBase func(args []reflect.Value) (results []reflect.Value)
