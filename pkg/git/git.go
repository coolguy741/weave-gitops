/*
Copyright 2021 The Flux authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package git

import (
	"context"
	"errors"

	gogit "github.com/go-git/go-git/v5"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

var (
	ErrNoGitRepository = errors.New("no git repository")
	ErrNoStagedFiles   = errors.New("no staged files")
)

type Author struct {
	Name  string
	Email string
}

type Commit struct {
	Author
	Hash    string
	Message string
}

// Git is an interface for basic Git operations on a single branch of a
// remote repository.
//counterfeiter:generate . Git
type Git interface {
	Open(path string) (*gogit.Repository, error)
	Init(path, url, branch string) (bool, error)
	Clone(ctx context.Context, path, url, branch string) (bool, error)
	Write(path string, content []byte) error
	Remove(path string) error
	Commit(message Commit, filters ...func(string) bool) (string, error)
	Push(ctx context.Context) error
	Status() (bool, error)
	Head() (string, error)
	GetRemoteUrl(dir string, remoteName string) (string, error)
	ValidateAccess(ctx context.Context, url string, branch string) error
}
