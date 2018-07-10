package diff

import (
	"fmt"

	"github.com/blang/semver"
	"github.com/containerum/kube-client/pkg/model"
	"github.com/docker/distribution/reference"
	"github.com/ninedraft/boxofstuff/strset"
)

func NewVersion(oldDepl, newDepl model.Deployment) semver.Version {
	var oldContainersNames = strset.NewSet(oldDepl.ContainersNames())
	var newContainersNames = strset.NewSet(newDepl.ContainersNames())
	var oldContainers = NewContainerSet(oldDepl.Containers)
	var newContainers = NewContainerSet(newDepl.Containers)
	var oldContainersVersions = func() map[string]ComparableContainer {
		var versions = make(map[string]ComparableContainer, len(oldDepl.Containers))
		for _, container := range oldDepl.Containers {
			var cmpCont = FromContainer(container)
			versions[container.Name] = cmpCont
		}
		return versions
	}()

	if len(oldContainersNames.Sub(newContainersNames)) > 0 {
		// some containers have been deleted
		var newVersion = oldDepl.Version
		newVersion.Major++
		newVersion.Minor = 0
		newVersion.Patch = 0
		return newVersion
	}

	var addedOrChangedContainers = newContainers.Sub(oldContainers)
	//var addedOrChangedContainersNames = addedOrChangedContainers.NamesSet()
	var changedContainers = addedOrChangedContainers.Filter(func(container ComparableContainer) bool {
		return oldContainersNames.Have(container.Name)
	})
	if changedContainers.OnlyLatest().Len() != changedContainers.Len() {
		// if some containers now use not semver
		var newVersion = oldDepl.Version
		newVersion.Major++
		newVersion.Minor = 0
		newVersion.Patch = 0
		return newVersion
	}

	var changedContainersWithSemvers = changedContainers.Sub(changedContainers.OnlyLatest())
	if changedContainersWithSemvers.Filter(func(container ComparableContainer) bool {
		return container.Version.Major() != oldContainersVersions[container.Name].Version.Major()
	}).Len() > 0 {
		// some images have major updates, so
		var newVersion = oldDepl.Version
		newVersion.Major++
		newVersion.Minor = 0
		newVersion.Patch = 0
		return newVersion
	}

	if changedContainersWithSemvers.Filter(func(container ComparableContainer) bool {
		return container.Version.Minor() != oldContainersVersions[container.Name].Version.Minor()
	}).Len() > 0 {
		// some images have minor updates, so
		var newVersion = oldDepl.Version
		newVersion.Minor++
		newVersion.Patch = 0
		return newVersion
	}

	if changedContainersWithSemvers.Filter(func(container ComparableContainer) bool {
		return container.Version.Patch() != oldContainersVersions[container.Name].Version.Patch()
	}).Len() > 0 {
		// some images have minor updates, so
		var newVersion = oldDepl.Version
		newVersion.Patch++
		return newVersion
	}

	if len(newContainersNames.Sub(oldContainersNames)) > 0 {
		// only new containers have been added
		var newVersion = oldDepl.Version
		newVersion.Major++
		newVersion.Minor = 0
		newVersion.Patch = 0
		return newVersion
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

func ComparableContainers(depl model.Deployment) []ComparableContainer {
	var containers = make([]ComparableContainer, 0, len(depl.Containers))
	for _, container := range depl.Containers {
		containers = append(containers, FromContainer(container))
	}
	return containers
}
