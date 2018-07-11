package diff

import (
	"fmt"

	"github.com/blang/semver"
	"github.com/containerum/kube-client/pkg/model"
	"github.com/docker/distribution/reference"
)

func NewVersion(oldDepl, newDepl model.Deployment) semver.Version {
	var version = oldDepl.Version
	var containers = diff(oldDepl.Containers, newDepl.Containers)
	var stats = containers.Stats()

	if stats.Deleted > 0 {
		version.Major++
		return version
	}

	if containers.Filter(func(change versionChange) bool {
		return (!change.New.IsSemver() || !change.Old.IsSemver()) && change.Type == Change
	}).Len() > 0 {
		version.Major++
		return version
	}

	var onlySemver = containers.Filter(func(change versionChange) bool {
		return change.New.IsSemver() && change.Old.IsSemver()
	})
	for _, v := range onlySemver {
		var semverChange = v.Diff()
		switch {
		case semverChange.Major() != 0:
			version.Major++
			return version
		case semverChange.Minor() != 0:
			version.Minor++
			return version
		case semverChange.Patch() != 0:
			version.Patch++
			return version
		}
	}

	if stats.Created > 0 {
		version.Minor++
		return version
	}

	// nothing ever changes
	return oldDepl.Version
}

func ContainerSemver(container model.Container) (semver.Version, bool) {
	var v = container.Version()
	if v == "" {
		return semver.Version{}, false
	}
	var semversion, err = semver.ParseTolerant(v)
	if err != nil {
		return semver.Version{}, false
	}
	return semversion, true
}

type ComparableContainer struct {
	Name    string
	Image   string
	Version TriVersion // zero means latest
}

func (c ComparableContainer) String() string {
	var img = c.Image
	if c.Version.String() != "" {
		img += ":" + c.Version.String()
	}
	return fmt.Sprintf("%s [%s]", c.Name, img)
}

func (c ComparableContainer) IsLatest() bool {
	return c.Version == TriVersion{}
}

func FromContainer(container model.Container) ComparableContainer {
	var imageName string
	var namedImage, err = reference.ParseNamed(container.Image)
	if err != nil {
		imageName = container.Image
	} else {
		imageName = namedImage.Name()
	}
	var version, ok = ContainerSemver(container)
	if !ok {
		var tagged, isTagged = namedImage.(reference.NamedTagged)
		if isTagged {
			version.Build = append(version.Build, tagged.Tag())
		}
	}
	return ComparableContainer{
		Name:    container.Name,
		Image:   imageName,
		Version: FromVersion(version),
	}
}
