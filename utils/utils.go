package utils

import (
	"fmt"
	"maps"
	"time"
)

type Set map[string]bool

func (s Set) Contains(key string) bool {
	_, contains := s[key]

	return contains
}

func NewSetFrom(l []string) Set {
	s := make(Set)
	for _, l := range l {
		s[l] = true
	}

	return s
}

func MergeEnvMap(parent, child map[string]string) map[string]string {
	result := map[string]string{}

	maps.Copy(result, parent)
	maps.Copy(result, child)

	return result
}

func MapPop[K comparable, T any](m map[K]T, key K) T {
	var zeroValue T

	if val, ok := m[key]; ok {
		delete(m, key)
		return val
	}

	return zeroValue
}

func EnvMapToList(envMap map[string]string) []string {
	envList := []string{}
	for name, value := range envMap {
		envList = append(envList, fmt.Sprintf("%s=%s", name, value))
	}

	return envList
}

func MaybeAddArgString(args []string, name, value string) []string {
	if value != "" {
		return append(args, name, value)
	}

	return args
}

func MaybeAddArgInt(args []string, name string, value int) []string {
	if value > 0 {
		return append(args, name, fmt.Sprint(value))
	}

	return args
}

func MaybeAddArgDuration(args []string, name string, value time.Duration) []string {
	if value > 0 {
		return append(args, name, value.String())
	}

	return args
}

func MaybeAddArgBool(args []string, name string, value bool) []string {
	if value {
		return append(args, name)
	}

	return args
}

func MaybeAddArgsList(args []string, name string, value []string) []string {
	for _, v := range value {
		args = append(args, name, v)
	}

	return args
}
