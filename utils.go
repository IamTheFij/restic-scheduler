package main

import "fmt"

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
