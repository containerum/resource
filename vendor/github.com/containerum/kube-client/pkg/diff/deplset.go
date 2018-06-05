package diff

import (
	"github.com/containerum/kube-client/pkg/model"
	"github.com/ninedraft/boxofstuff/strset"
)

type ContainerSet map[ComparableContainer]model.Container

func NewContainerSet(containers []model.Container) ContainerSet {
	var set = make(ContainerSet, len(containers))
	for _, container := range containers {
		set[FromContainer(container)] = container
	}
	return set
}

func (set ContainerSet) Copy() ContainerSet {
	var cp = make(ContainerSet, len(set))
	for k, v := range set {
		cp[k] = v
	}
	return cp
}

func (set ContainerSet) Have(container model.Container) bool {
	var _, ok = set[FromContainer(container)]
	return ok
}

func (set ContainerSet) Put(container model.Container) ContainerSet {
	set = set.Copy()
	set[FromContainer(container)] = container
	return set
}

func (set ContainerSet) New() ContainerSet {
	return make(ContainerSet, len(set))
}

func (set ContainerSet) Len() int {
	return len(set)
}

func (set ContainerSet) Filter(pred func(container ComparableContainer) bool) ContainerSet {
	var filtered = set.New()
	for k, v := range set {
		if pred(k) {
			filtered[k] = v
		}
	}
	return filtered
}

func (set ContainerSet) Keys() []ComparableContainer {
	var containers = make([]ComparableContainer, 0, set.Len())
	for container := range set {
		containers = append(containers, container)
	}
	return containers
}

func (set ContainerSet) Values() []model.Container {
	var containers = make([]model.Container, 0, set.Len())
	for _, container := range set {
		containers = append(containers, container)
	}
	return containers
}

func (set ContainerSet) Sub(x ContainerSet) ContainerSet {
	set = set.Copy()
	for k := range x {
		delete(set, k)
	}
	return set
}

func (set ContainerSet) Names() []string {
	var names = make([]string, 0, set.Len())
	for k := range set {
		names = append(names, k.Name)
	}
	return names
}

func (set ContainerSet) NamesSet() strset.Set {
	return strset.NewSet(set.Names())
}

func (set ContainerSet) OnlyLatest() ContainerSet {
	return set.Filter(func(container ComparableContainer) bool {
		return container.IsLatest()
	})
}
