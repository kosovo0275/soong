package android

import (
	"fmt"
	"sync"
)

type OncePer struct {
	values     sync.Map
	valuesLock sync.Mutex
}

type valueMap map[interface{}]interface{}

func (once *OncePer) Once(key interface{}, value func() interface{}) interface{} {
	if v, ok := once.values.Load(key); ok {
		return v
	}

	once.valuesLock.Lock()
	defer once.valuesLock.Unlock()

	if v, ok := once.values.Load(key); ok {
		return v
	}

	v := value()
	once.values.Store(key, v)

	return v
}

func (once *OncePer) Get(key interface{}) interface{} {
	v, ok := once.values.Load(key)
	if !ok {
		panic(fmt.Errorf("Get() called before Once()"))
	}

	return v
}

func (once *OncePer) OnceStringSlice(key interface{}, value func() []string) []string {
	return once.Once(key, func() interface{} { return value() }).([]string)
}

func (once *OncePer) Once2StringSlice(key interface{}, value func() ([]string, []string)) ([]string, []string) {
	type twoStringSlice [2][]string
	s := once.Once(key, func() interface{} {
		var s twoStringSlice
		s[0], s[1] = value()
		return s
	}).(twoStringSlice)
	return s[0], s[1]
}
