package server

import (
	"errors"
	"os"
)

type Config struct {
	Content    []string `json:"content"`
	Throughput uint     `json:"throughput"`
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return errors.Is(err, os.ErrNotExist)
}

func ValidateConfig(obj Config) bool {
	for _, val := range obj.Content {
		if !fileExists(val) {
			return false
		}
	}
	return true
}
