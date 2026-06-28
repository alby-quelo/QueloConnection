package agent

import "os"

func osHostLookup() (string, error) {
	return os.Hostname()
}
