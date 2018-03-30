package main

import (
	"bytes"
	"context"
	"fmt"
	"go/format"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/google/go-github/github"
	"github.com/weppos/publicsuffix-go/publicsuffix"
)

const (
	goSrc = `// This file is automatically generated
// Run "go run cmd/gen/gen.go" to update the list.

package publicsuffix

const defaultListVersion = "PSL version {{.VersionSHA}} ({{.VersionDate}})"

func init() {
	r := [{{len .Rules}}]Rule{
		{{range $r := .Rules}} \
		{ {{$r.Type}}, "{{$r.Value}}", {{$r.Length}}, {{$r.Private}} },
		{{end}}
	}
	DefaultList.rules = r[:]
}

`
)

var (
	goTemplate = template.Must(template.New("").Parse(cont(goSrc)))
)

// https://github.com/golang/go/issues/9969
func cont(s string) string {
	return strings.Replace(s, "\\\n", "", -1)
}

func main() {
	sha, datetime := extractHeadInfo()

	resp, err := http.Get(fmt.Sprintf("https://raw.githubusercontent.com/publicsuffix/list/%s/public_suffix_list.dat", sha))
	if err != nil {
		fatal(err)
	}
	defer resp.Body.Close()

	list := publicsuffix.NewList()
	rules, err := list.Load(resp.Body, nil)
	if err != nil {
		fatal(err)
	}

	data := struct {
		VersionSHA  string
		VersionDate string
		Rules       []publicsuffix.Rule
	}{
		sha[:6],
		datetime.Format(time.ANSIC),
		rules,
	}

	buf := new(bytes.Buffer)
	err = goTemplate.Execute(buf, &data)
	if err != nil {
		fatal(err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		fatal(err)
	}
	_, err = os.Stdout.Write(formatted)
	//_, err = os.Stdout.Write(buf.Bytes())
}

func extractHeadInfo() (sha string, datetime time.Time) {
	client := github.NewClient(nil)

	commits, _, err := client.Repositories.ListCommits(context.Background(), "publicsuffix", "list", nil)
	if err != nil {
		fatal(err)
	}

	lastCommit := commits[0]
	return lastCommit.GetSHA(), lastCommit.GetCommit().GetCommitter().GetDate()
}

func fatal(err error) {
	fmt.Println(err)
	os.Exit(1)
}