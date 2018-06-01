package diff

import (
	"sort"

	"github.com/containerum/kube-client/pkg/model"
	"github.com/pmezard/go-difflib/difflib"
)

func Diff(newDepl, oldDepl model.Deployment) string {
	var newImages = newDepl.ContainersAndImages()
	var oldImages = oldDepl.ContainersAndImages()
	sort.Strings(newImages)
	sort.Strings(oldImages)
	for i, image := range oldImages {
		oldImages[i] = image + "\n"
	}
	for i, image := range newImages {
		newImages[i] = image + "\n"
	}
	var oldTimestamp, newTimestamp string
	if oldDepl.Status != nil {
		oldTimestamp = oldDepl.Status.CreatedAt
	}
	if newDepl.Status != nil {
		newTimestamp = oldDepl.Status.CreatedAt
	}
	var diff = difflib.UnifiedDiff{
		A:        oldImages,
		B:        newImages,
		FromFile: oldDepl.Name,
		FromDate: oldTimestamp,
		ToDate:   newTimestamp,
		ToFile:   newDepl.Name,
		Context:  3,
	}
	var diffString, err = difflib.GetUnifiedDiffString(diff)
	if err != nil {
		return err.Error()
	}
	return diffString
}
