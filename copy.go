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
	"strings"
)

func (g *Commands) Copy() (err error) {
	srcLen := len(g.opts.Sources)
	if srcLen != 2 {
		return fmt.Errorf("expecting <src> <dest>")
	}

	var srcFile *File
	src, dest := g.opts.Sources[0], g.opts.Sources[1]

	srcFile, err = g.rem.FindByPath(src)
	if err != nil {
		return
	}
	if srcFile == nil {
		return fmt.Errorf("%s: source doesn't exist\n", src)
	}

	// TODO: Incorporate directory copying
	if !g.opts.Recursive && srcFile.IsDir {
		return fmt.Errorf("%s is a directory", src)
	}
	_, err = g.rem.Copy(srcFile, strings.TrimLeft(dest, "/"))
	return
}
