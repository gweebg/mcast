package server

import (
	"os"
)

type ConfigItem struct {
	Name   string
	Width  uint
	Height uint
	FPS    uint
}

type Config struct {
	Content    []ConfigItem `json:"content"`
	Throughput uint         `json:"throughput"`
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return false
	}

	return true
}

func ValidateConfig(obj Config) bool {

	for _, val := range obj.Content {
		if !fileExists(val.Name) {
			return false
		}
	}
	return true
}
