// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package base

import (
	"container/list"
	"encoding/json"
	"fmt"
	"html/template"
	"runtime"
	"strings"
	"time"

	"golang.org/x/net/html/charset"
	"golang.org/x/text/transform"

	"github.com/gogits/chardet"
	"github.com/gogits/gogs/modules/setting"
)

func Safe(raw string) template.HTML {
	return template.HTML(raw)
}

func Str2html(raw string) template.HTML {
	return template.HTML(Sanitizer.Sanitize(raw))
}

func Range(l int) []int {
	return make([]int, l)
}

func List(l *list.List) chan interface{} {
	e := l.Front()
	c := make(chan interface{})
	go func() {
		for e != nil {
			c <- e.Value
			e = e.Next()
		}
		close(c)
	}()
	return c
}

func Sha1(str string) string {
	return EncodeSha1(str)
}

func ShortSha(sha1 string) string {
	if len(sha1) == 40 {
		return sha1[:10]
	}
	return sha1
}

func DetectEncoding(content []byte) (string, error) {
	detector := chardet.NewTextDetector()
	result, err := detector.DetectBest(content)
	if result.Charset != "UTF-8" && len(setting.Repository.AnsiCharset) > 0 {
		return setting.Repository.AnsiCharset, err
	}
	return result.Charset, err
}

func ToUtf8WithErr(content []byte) (error, string) {
	charsetLabel, err := DetectEncoding(content)
	if err != nil {
		return err, ""
	}

	if charsetLabel == "UTF-8" {
		return nil, string(content)
	}

	encoding, _ := charset.Lookup(charsetLabel)
	if encoding == nil {
		return fmt.Errorf("unknown char decoder %s", charsetLabel), string(content)
	}

	result, n, err := transform.String(encoding.NewDecoder(), string(content))

	// If there is an error, we concatenate the nicely decoded part and the
	// original left over. This way we won't loose data.
	if err != nil {
		result = result + string(content[n:])
	}

	return err, result
}

func ToUtf8(content string) string {
	_, res := ToUtf8WithErr([]byte(content))
	return res
}

// Replaces all prefixes 'old' in 's' with 'new'.
func ReplaceLeft(s, old, new string) string {
	old_len, new_len, i, n := len(old), len(new), 0, 0
	for ; i < len(s) && strings.HasPrefix(s[i:], old); n += 1 {
		i += old_len
	}

	// simple optimization
	if n == 0 {
		return s
	}

	// allocating space for the new string
	newLen := n*new_len + len(s[i:])
	replacement := make([]byte, newLen, newLen)

	j := 0
	for ; j < n*new_len; j += new_len {
		copy(replacement[j:j+new_len], new)
	}

	copy(replacement[j:], s[i:])
	return string(replacement)
}

// RenderCommitMessage renders commit message with XSS-safe and special links.
func RenderCommitMessage(msg, urlPrefix string) template.HTML {
	cleanMsg := template.HTMLEscapeString(msg)
	fullMessage := string(RenderIssueIndexPattern([]byte(cleanMsg), urlPrefix))
	msgLines := strings.Split(strings.TrimSpace(fullMessage), "\n")
	for i := range msgLines {
		msgLines[i] = ReplaceLeft(msgLines[i], " ", "&nbsp;")
	}

	fullMessage = strings.Join(msgLines, "<br>")
	return template.HTML(fullMessage)
}

var TemplateFuncs template.FuncMap = map[string]interface{}{
	"GoVer": func() string {
		return strings.Title(runtime.Version())
	},
	"AppName": func() string {
		return setting.AppName
	},
	"AppSubUrl": func() string {
		return setting.AppSubUrl
	},
	"AppVer": func() string {
		return setting.AppVer
	},
	"AppDomain": func() string {
		return setting.Domain
	},
	"DisableGravatar": func() bool {
		return setting.DisableGravatar
	},
	"LoadTimes": func(startTime time.Time) string {
		return fmt.Sprint(time.Since(startTime).Nanoseconds()/1e6) + "ms"
	},
	"AvatarLink":   AvatarLink,
	"Safe":         Safe,
	"Str2html":     Str2html,
	"TimeSince":    TimeSince,
	"RawTimeSince": RawTimeSince,
	"FileSize":     FileSize,
	"Subtract":     Subtract,
	"Add": func(a, b int) int {
		return a + b
	},
	"ActionIcon": ActionIcon,
	"DateFmtLong": func(t time.Time) string {
		return t.Format(time.RFC1123Z)
	},
	"DateFmtShort": func(t time.Time) string {
		return t.Format("Jan 02, 2006")
	},
	"List": List,
	"Mail2Domain": func(mail string) string {
		if !strings.Contains(mail, "@") {
			return "try.gogs.io"
		}

		return strings.SplitN(mail, "@", 2)[1]
	},
	"SubStr": func(str string, start, length int) string {
		if len(str) == 0 {
			return ""
		}
		end := start + length
		if length == -1 {
			end = len(str)
		}
		if len(str) < end {
			return str
		}
		return str[start:end]
	},
	"DiffTypeToStr":     DiffTypeToStr,
	"DiffLineTypeToStr": DiffLineTypeToStr,
	"Sha1":              Sha1,
	"ShortSha":          ShortSha,
	"Md5":               EncodeMd5,
	"ActionContent2Commits": ActionContent2Commits,
	"Oauth2Icon":            Oauth2Icon,
	"Oauth2Name":            Oauth2Name,
	"ToUtf8":                ToUtf8,
	"EscapePound": func(str string) string {
		return strings.Replace(strings.Replace(str, "%", "%25", -1), "#", "%23", -1)
	},
	"RenderCommitMessage": RenderCommitMessage,
}

type Actioner interface {
	GetOpType() int
	GetActUserName() string
	GetActEmail() string
	GetRepoUserName() string
	GetRepoName() string
	GetRepoPath() string
	GetRepoLink() string
	GetBranch() string
	GetContent() string
	GetCreate() time.Time
	GetIssueInfos() []string
}

// ActionIcon accepts a int that represents action operation type
// and returns a icon class name.
func ActionIcon(opType int) string {
	switch opType {
	case 1, 8: // Create, transfer repository.
		return "repo"
	case 5, 9: // Commit repository.
		return "git-commit"
	case 6: // Create issue.
		return "issue-opened"
	case 10: // Comment issue.
		return "comment"
	default:
		return "invalid type"
	}
}

type PushCommit struct {
	Sha1        string
	Message     string
	AuthorEmail string
	AuthorName  string
}

type PushCommits struct {
	Len        int
	Commits    []*PushCommit
	CompareUrl string
}

func ActionContent2Commits(act Actioner) *PushCommits {
	var push *PushCommits
	if err := json.Unmarshal([]byte(act.GetContent()), &push); err != nil {
		return nil
	}
	return push
}

func DiffTypeToStr(diffType int) string {
	diffTypes := map[int]string{
		1: "add", 2: "modify", 3: "del", 4: "rename",
	}
	return diffTypes[diffType]
}

func DiffLineTypeToStr(diffType int) string {
	switch diffType {
	case 2:
		return "add"
	case 3:
		return "del"
	case 4:
		return "tag"
	}
	return "same"
}

func Oauth2Icon(t int) string {
	switch t {
	case 1:
		return "fa-github-square"
	case 2:
		return "fa-google-plus-square"
	case 3:
		return "fa-twitter-square"
	case 4:
		return "fa-qq"
	case 5:
		return "fa-weibo"
	}
	return ""
}

func Oauth2Name(t int) string {
	switch t {
	case 1:
		return "GitHub"
	case 2:
		return "Google+"
	case 3:
		return "Twitter"
	case 4:
		return "腾讯 QQ"
	case 5:
		return "Weibo"
	}
	return ""
}
