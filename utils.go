package main

func MergeEnv(parent, child map[string]string) (result map[string]string) {
	for key, value := range parent {
		result[key] = value
	}

	for key, value := range child {
		result[key] = value
	}

	return
}
