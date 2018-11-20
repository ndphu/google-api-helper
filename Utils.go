package google_api_helper

import "log"

func FailOnError(msg string, err error) {
	if err != nil {
		log.Fatalf("%s: %v", msg, err)
	}
}
