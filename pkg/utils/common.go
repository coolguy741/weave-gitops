package utils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	wego "github.com/weaveworks/weave-gitops/api/v1alpha1"
	validation "k8s.io/apimachinery/pkg/api/validation"
)

var commitMessage string

func Exists(filePath string) bool {
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// WaitUntil runs checkDone until a timeout is reached
func WaitUntil(out io.Writer, poll, timeout time.Duration, checkDone func() error) error {
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		err := checkDone()
		if err == nil {
			return nil
		}
		fmt.Fprintf(out, "error occurred %s, retrying in %s\n", err, poll.String())
	}
	return fmt.Errorf("timeout reached %s", timeout.String())
}

type callback func()

func CaptureStdout(c callback) string {
	r, w, _ := os.Pipe()
	tmp := os.Stdout
	defer func() {
		os.Stdout = tmp
	}()
	os.Stdout = w
	c()
	w.Close()
	stdout, _ := ioutil.ReadAll(r)

	return string(stdout)
}

func SetCommmitMessageFromArgs(cmd string, url, path, name string) {
	commitMessage = fmt.Sprintf("%s %s %s %s", cmd, url, path, name)
}

func SetCommmitMessage(msg string) {
	commitMessage = msg
}

func GetCommitMessage() string {
	return commitMessage
}

func UrlToRepoName(url string) string {
	return strings.TrimSuffix(filepath.Base(url), ".git")
}

func GetOwnerFromUrl(url string) (string, error) {
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("could not get owner from url %s", url)
	}
	return parts[len(parts)-2], nil
}

func ValidateNamespace(ns string) error {
	if errList := validation.ValidateNamespaceName(ns, false); len(errList) != 0 {
		return fmt.Errorf("invalid namespace: %s", strings.Join(errList, ", "))
	}

	return nil
}

// SanitizeRepoUrl accepts a url like git@github.com:someuser/podinfo.git and converts it into
// a string like ssh://git@github.com/someuser/podinfo.git. This helps standardize the different
// user inputs that might be provided.
func SanitizeRepoUrl(url string) string {
	trimmed := ""

	if !strings.HasSuffix(url, ".git") {
		url = url + ".git"
	}

	sshPrefix := "git@github.com:"
	if strings.HasPrefix(url, sshPrefix) {
		trimmed = strings.TrimPrefix(url, sshPrefix)
	}

	httpsPrefix := "https://github.com/"
	if strings.HasPrefix(url, httpsPrefix) {
		trimmed = strings.TrimPrefix(url, httpsPrefix)
	}

	if trimmed != "" {
		return "ssh://git@github.com/" + trimmed
	}

	return url
}

func GetAppHash(app wego.Application) (string, error) {
	var appHash string
	var err error

	var getHash = func(inputs ...string) (string, error) {
		h := md5.New()
		final := ""
		for _, input := range inputs {
			final += input
		}
		_, err := h.Write([]byte(final))
		if err != nil {
			return "", fmt.Errorf("error generating app hash %s", err)
		}
		return hex.EncodeToString(h.Sum(nil)), nil
	}

	if app.Spec.DeploymentType == wego.DeploymentTypeHelm {
		appHash, err = getHash(app.Spec.URL, app.Name, app.Spec.Branch)
		if err != nil {
			return "", err
		}
	} else {
		appHash, err = getHash(app.Spec.URL, app.Spec.Path, app.Spec.Branch)
		if err != nil {
			return "", err
		}
	}
	return "wego-" + appHash, nil
}

func PrintTable(writer io.Writer, header []string, rows [][]string) {
	table := tablewriter.NewWriter(writer)
	table.SetHeader(header)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)
	table.AppendBulk(rows)
	table.Render()
}

func CleanCommitMessage(msg string) string {
	str := strings.ReplaceAll(msg, "\n", " ")
	if len(str) > 50 {
		str = str[:49] + "..."

	}
	return str
}

func CleanCommitCreatedAt(createdAt time.Time) string {
	return strings.ReplaceAll(createdAt.String(), " +0000", "")
}

func ConvertCommitHashToShort(hash string) string {
	return hash[:7]
}
