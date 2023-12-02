package rendezvous

import "github.com/gweebg/mcast/internal/server"

func Contains(s []server.ConfigItem, str string) bool {
	for _, v := range s {
		if v.Name == str {
			return true
		}
	}
	return false
}
