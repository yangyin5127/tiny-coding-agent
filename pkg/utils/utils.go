package utils

import "os"

func Getwd() string {
	wd, _ := os.Getwd()
	return wd
}
