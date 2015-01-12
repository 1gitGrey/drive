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
	"sync"
	"time"

	drive "github.com/google/google-api-go-client/drive/v2"
)

const (
	CopyNone = 1 << iota
	CopyAllowDuplicates
)

const (
	RatePerSecond = 20
)

func allowDuplicates(mask int) bool {
	return (mask & CopyAllowDuplicates) != 0
}

func (g *Commands) Copy() (err error) {
	srcLen := len(g.opts.Sources)
	if srcLen < 2 {
		return fmt.Errorf("expecting <src> [src...] <dest>")
	}

	nestingCount := 0
	var srcFiles []*File
	rest, dest := g.opts.Sources[:srcLen-1], g.opts.Sources[srcLen-1]

	if len(rest) >= 2 {
		nestingCount += 1
	}
	for _, src := range rest {
		srcFile, err := g.rem.FindByPath(src)
		if err != nil {
			return err
		}
		if srcFile == nil {
			return fmt.Errorf("%s: source doesn't exist", src)
		}

		if srcFile.IsDir {
			nestingCount += 1
		}

		if !srcFile.Copyable && !srcFile.IsDir {
			return fmt.Errorf("%s: not copyable", src)
		}
		if srcFile.IsDir && !g.opts.Recursive {
			return fmt.Errorf("copy: %s is a directory", src)
		}
		srcFiles = append(srcFiles, srcFile)
	}

	destFile, destErr := g.rem.FindByPath(dest)
	if destErr != nil {
		if destErr != ErrPathNotExists {
			return destErr
		}
	}
	if nestingCount > 1 && destFile != nil && !destFile.IsDir {
		return fmt.Errorf("%s: is not a directory yet multiple paths are to be copied to it")
	}

	var wg sync.WaitGroup
	wg.Add(len(srcFiles))

	for _, srcFile := range srcFiles {
		go func(f *File, wg *sync.WaitGroup) {
			defer wg.Done()
			if _, err = g.copy(f, dest); err != nil {
				fmt.Println(err)
			}
		}(srcFile, &wg)
	}
	wg.Wait()
	return
}

func (g *Commands) copy(srcFile *File, dest string) (*File, error) {
	parent, child := parentChild(dest)
	// fmt.Printf("parent %s child %s srcFile %s\n", parent, child, srcFile.Name)
	destParent, destErr := g.rem.FindByPath(parent)
	if destErr != nil {
		return nil, destErr
	}

	destFile, destErr := g.rem.FindByPath(dest)
	if !allowDuplicates(g.opts.TypeMask) {
		if destErr != nil && destErr != ErrPathNotExists {
			return nil, destErr
		}
		if destFile != nil {
			if !destFile.IsDir {
				return nil, fmt.Errorf("copy [%s]: No duplicates allowed when CopyDuplicates is not set", dest)
			}
			child = destFile.Name + "/" + child
		}
	}

	parentId := ""
	if destParent != nil {
		parentId = destParent.Id
	}

	if destFile != nil && destFile.IsDir && !srcFile.IsDir {
		child = srcFile.Name
		parentId = destFile.Id
	}

	if !srcFile.IsDir {
		if parentId == "" {
			return nil, fmt.Errorf("cannot copy to a non existant parent: %s", parent)
		}
		return g.rem.Copy(srcFile, parentId, child)
	}

	if !g.opts.Recursive {
		return nil, fmt.Errorf("%s is a directory", srcFile.Name)
	}

	dupdFile := srcFile.DupFile()

	// Note explicitly clear dupFile's Id to register this as a first time creation
	dupdFile.Id = ""

	_, pErr := g.rem.Upsert(parentId, dupdFile, destFile, nil)
	if pErr != nil {
		return nil, pErr
	}

	searchExpr := buildExpression(srcFile.Id, 0, false)

	req := g.rem.service.Files.List()
	req.Q(searchExpr)

	// TODO: Get pageSize from g.opts
	req.MaxResults(g.opts.PageSize)
	pageToken := ""
	throttle := time.Tick(1e9 / RatePerSecond)
	for {
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}
		<-throttle
		res, childErr := req.Do()
		if childErr != nil {
			return nil, childErr
		}

		var wg sync.WaitGroup

		newG := New(g.context, g.opts)
		newG.taskStart(len(res.Items))
		wg.Add(len(res.Items))
		for _, file := range res.Items {
			go func(f *drive.File, dest string, wgg *sync.WaitGroup) {
				defer wgg.Done()
				defer newG.taskDone()
				if isHidden(file.Title, newG.opts.Hidden) {
					return
				}
				rem := NewRemoteFile(f)

				<-throttle
				// Exponential back-off in case
				_, err := newG.copy(rem, dest+"/"+rem.Name)
				if err != nil {
					fmt.Println(err)
				}
			}(file, dest, &wg)
		}
		wg.Wait()
		newG.taskFinish()

		pageToken = res.NextPageToken
		if pageToken == "" {
			break
		}
		if !nextPage() {
			return nil, nil
		}
	}
	return nil, nil
}
