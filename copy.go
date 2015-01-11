// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package drive

import (
	"fmt"
)

const (
	CopyNone = 1 << iota
	CopyAllowDuplicates
)

func allowDuplicates(mask int) bool {
	return (mask & CopyAllowDuplicates) != 0
}

func (g *Commands) Copy() (err error) {
	srcLen := len(g.opts.Sources)
	if srcLen < 2 {
		return fmt.Errorf("expecting <src> <dest>")
	}

	src, dest := g.opts.Sources[0], g.opts.Sources[1]
	srcFile, err := g.rem.FindByPath(src)
	if err != nil {
		return err
	}
	if srcFile == nil {
		return fmt.Errorf("%s: source doesn't exist", src)
	}
	if !srcFile.Copyable {
		return fmt.Errorf("%s: not copyable", src)
	}
	_, err = g.copy(srcFile, dest)
	return
}

func (g *Commands) copy(srcFile *File, dest string) (*File, error) {
	// Noop for now for directory copying.
	if srcFile.IsDir {
		// && !g.opts.Recursive
		return nil, fmt.Errorf("%s is a directory", srcFile.Name)
	}
	parent, child := parentChild(dest)
	destParent, destErr := g.rem.FindByPath(parent)
	if destErr != nil {
		if destErr != ErrPathNotExists {
			return nil, destErr
		}
	}

	if !allowDuplicates(g.opts.TypeMask) {
		destFile, destErr := g.rem.FindByPath(dest)
		if destErr != nil && destErr != ErrPathNotExists {
			return nil, destErr
		}
		if destFile != nil {
			return nil, fmt.Errorf("copy [%s]: No duplicates allowed when CopyDuplicates is not set", dest)
		}
	}
	return g.rem.Copy(srcFile, destParent.Id, child)
}
