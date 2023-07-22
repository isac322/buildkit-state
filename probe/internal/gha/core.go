package gha

import (
	"strings"

	"github.com/sethvargo/go-githubactions"
	"golang.org/x/exp/slices"
)

func GetMultilineInput(gha *githubactions.Action, name string) []string {
	raw := gha.GetInput(name)
	if raw == "" {
		return nil
	}
	splitted := strings.Split(raw, "\n")
	return slices.DeleteFunc(splitted, func(s string) bool {
		return s == ""
	})
}
