package main

import "fmt"

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

	for key, value := range parent {
		result[key] = value
	}

	for key, value := range child {
		result[key] = value
	}

	return result
}

func EnvMapToList(envMap map[string]string) []string {
	envList := []string{}
	for name, value := range envMap {
		envList = append(envList, fmt.Sprintf("%s=%s", name, value))
	}

	return envList
}

func maybeAddArgString(args []string, name, value string) []string {
	if value != "" {
		return append(args, name, value)
	}

	return args
}

func maybeAddArgInt(args []string, name string, value int) []string {
	if value > 0 {
		return append(args, name, fmt.Sprint(value))
	}

	return args
}

func maybeAddArgBool(args []string, name string, value bool) []string {
	if value {
		return append(args, name)
	}

	return args
}

func maybeAddArgsList(args []string, name string, value []string) []string {
	for _, v := range value {
		args = append(args, name, v)
	}

	return args
}
