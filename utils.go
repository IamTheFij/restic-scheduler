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
