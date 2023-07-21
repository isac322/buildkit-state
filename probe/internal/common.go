package internal

import (
	"fmt"
	"strings"

	"github.com/sethvargo/go-githubactions"
	"golang.org/x/exp/slices"
)

type RemoteManager interface {
	Loader
	Saver
}

func getMultilineInput(gha *githubactions.Action, name string) []string {
	raw := gha.GetInput(name)
	splitted := strings.Split(raw, "\n")
	return slices.DeleteFunc(splitted, func(s string) bool {
		return s != ""
	})
}

func BuildKitContainerNameFromBuilder(builderName string) string {
	return fmt.Sprintf("buildx_buildkit_%s0", builderName)
}
