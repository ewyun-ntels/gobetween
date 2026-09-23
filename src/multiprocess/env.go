package multiprocess

import "os"

var getenv = os.Getenv

func replaceEnv(env []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(env)+1)
	for _, entry := range env {
		if len(entry) >= len(prefix) && entry[:len(prefix)] == prefix {
			continue
		}
		result = append(result, entry)
	}
	if value != "" {
		result = append(result, prefix+value)
	}
	return result
}
