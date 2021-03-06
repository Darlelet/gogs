// Copyright 2015 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package git

import (
	"container/list"
	"fmt"
	"strings"
	"time"

	"github.com/Unknwon/com"
)

type PullRequestInfo struct {
	MergeBase string
	Commits   *list.List
	// Diff      *Diff
	NumFiles int
}

// GetMergeBase checks and returns merge base of two branches.
func (repo *Repository) GetMergeBase(remoteBranch, headBranch string) (string, error) {
	// Get merge base commit.
	stdout, stderr, err := com.ExecCmdDir(repo.Path, "git", "merge-base", remoteBranch, headBranch)
	if err != nil {
		return "", fmt.Errorf("get merge base: %v", concatenateError(err, stderr))
	}
	return strings.TrimSpace(stdout), nil
}

// AddRemote adds a remote to repository.
func (repo *Repository) AddRemote(name, path string) error {
	_, stderr, err := com.ExecCmdDir(repo.Path, "git", "remote", "add", "-f", name, path)
	if err != nil {
		return fmt.Errorf("add remote(%s - %s): %v", name, path, concatenateError(err, stderr))
	}
	return nil
}

// RemoveRemote removes a remote from repository.
func (repo *Repository) RemoveRemote(name string) error {
	_, stderr, err := com.ExecCmdDir(repo.Path, "git", "remote", "remove", name)
	if err != nil {
		return fmt.Errorf("remove remote(%s): %v", name, concatenateError(err, stderr))
	}
	return nil
}

// GetPullRequestInfo generates and returns pull request information
// between base and head branches of repositories.
func (repo *Repository) GetPullRequestInfo(basePath, baseBranch, headBranch string) (_ *PullRequestInfo, err error) {
	// Add a temporary remote.
	tmpRemote := com.ToStr(time.Now().UnixNano())
	if err = repo.AddRemote(tmpRemote, basePath); err != nil {
		return nil, fmt.Errorf("AddRemote: %v", err)
	}
	defer func() {
		repo.RemoveRemote(tmpRemote)
	}()

	remoteBranch := "remotes/" + tmpRemote + "/" + baseBranch

	prInfo := new(PullRequestInfo)
	prInfo.MergeBase, err = repo.GetMergeBase(remoteBranch, headBranch)
	if err != nil {
		return nil, fmt.Errorf("GetMergeBase: %v", err)
	}

	stdout, stderr, err := com.ExecCmdDir(repo.Path, "git", "log", prInfo.MergeBase+"..."+headBranch, prettyLogFormat)
	if err != nil {
		return nil, fmt.Errorf("list diff logs: %v", concatenateError(err, stderr))
	}
	prInfo.Commits, err = parsePrettyFormatLog(repo, []byte(stdout))
	if err != nil {
		return nil, fmt.Errorf("parsePrettyFormatLog: %v", err)
	}

	// Count number of changed files.
	stdout, stderr, err = com.ExecCmdDir(repo.Path, "git", "diff", "--name-only", remoteBranch+"..."+headBranch)
	if err != nil {
		return nil, fmt.Errorf("list changed files: %v", concatenateError(err, stderr))
	}
	prInfo.NumFiles = len(strings.Split(stdout, "\n")) - 1

	return prInfo, nil
}

// GetPatch generates and returns patch data between given branches.
func (repo *Repository) GetPatch(mergeBase, headBranch string) ([]byte, error) {
	stdout, stderr, err := com.ExecCmdDirBytes(repo.Path, "git", "diff", "-p", mergeBase, headBranch)
	if err != nil {
		return nil, concatenateError(err, string(stderr))
	}

	return stdout, nil
}

// Merge merges pull request from head repository and branch.
func (repo *Repository) Merge(headRepoPath string, baseBranch, headBranch string) error {

	return nil
}
