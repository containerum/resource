package diff

import (
	"github.com/containerum/kube-client/pkg/model"
)

type ChangeType byte

const (
	Delete ChangeType = iota
	Change
	Create
)

type versionChange struct {
	Type ChangeType
	Old  TriVersion
	New  TriVersion
}

func (change versionChange) Diff() TriVersion {
	return TriVersion{
		Index: [...]uint64{
			change.New.Index[0] - change.Old.Index[0],
			change.New.Index[1] - change.Old.Index[1],
			change.New.Index[2] - change.Old.Index[2],
		},
	}
}

type Set map[string]versionChange

func diff(old, new []model.Container) Set {
	var set = make(Set, len(old)+len(new))
	for _, cont := range old {
		var change = versionChange{
			Type: Delete,
		}
		change.Old = FromContainer(cont).Version
		set[cont.Name] = change
	}
	for _, cont := range new {
		var change, ok = set[cont.Name]
		if !ok {
			change.Type = Create
			change.New = FromContainer(cont).Version
		} else {
			change.Type = Change
			change.New = FromContainer(cont).Version
		}
		set[cont.Name] = change
	}
	return set
}

type Stats struct {
	Changed int
	Deleted int
	Created int
}

func (set Set) Stats() Stats {
	var stats Stats
	for _, version := range set {
		switch version.Type {
		case Change:
			stats.Changed++
		case Create:
			stats.Created++
		case Delete:
			stats.Deleted++
		}
	}
	return stats
}

func (set Set) Filter(pred func(change versionChange) bool) Set {
	var filtered = make(Set, len(set))
	for name, version := range set {
		if pred(version) {
			filtered[name] = version
		}
	}
	return filtered
}

func (set Set) Changed() Set {
	return set.Filter(func(change versionChange) bool {
		return change.Type == Change
	})
}

func (set Set) Deleted() Set {
	return set.Filter(func(change versionChange) bool {
		return change.Type == Delete
	})
}

func (set Set) Created() Set {
	return set.Filter(func(change versionChange) bool {
		return change.Type == Create
	})
}

func (set Set) Len() int {
	return len(set)
}
