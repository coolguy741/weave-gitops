package osys

import (
	"os"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . Osys
type Osys interface {
	UserHomeDir() (string, error)
	Getenv(envVar string) string
	Setenv(envVar, value string) error
	Unsetenv(envVar string) error
	Exit(code int)
	Stdin() *os.File
	Stdout() *os.File
	Stderr() *os.File
}

type OsysClient struct{}

func New() Osys {
	return &OsysClient{}
}

func (o *OsysClient) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

func (o *OsysClient) Getenv(envVar string) string {
	return os.Getenv(envVar)
}

func (o *OsysClient) Setenv(envVar, value string) error {
	return os.Setenv(envVar, value)
}

func (o *OsysClient) Unsetenv(envVar string) error {
	return os.Unsetenv(envVar)
}

func (o *OsysClient) Exit(code int) {
	os.Exit(code)
}

func (o *OsysClient) Stdin() *os.File {
	return os.Stdin
}

func (o *OsysClient) Stdout() *os.File {
	return os.Stdout
}

func (o *OsysClient) Stderr() *os.File {
	return os.Stderr
}
