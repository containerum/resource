package diff

import (
	"strconv"

	"sort"
	"strings"

	"github.com/blang/semver"
)

type TriVersion struct {
	Index     [3]uint64
	TagString string
}

func (v TriVersion) Tags() []string {
	return strings.Split(v.TagString, ":")
}

func (v TriVersion) String() string {
	var vstr string
	if v.Index != (TriVersion{}).Index {
		vstr = "v" +
			strconv.FormatUint(v.Index[0], 10) + "." +
			strconv.FormatUint(v.Index[1], 10) + "." +
			strconv.FormatUint(v.Index[2], 10)
	}
	if v.TagString != "" {
		vstr += v.TagString
	}
	return vstr
}

func FromVersion(v semver.Version) TriVersion {
	var tags = make([]string, 0, len(v.Build))
	for _, tag := range v.Build {
		if tag != "latest" {
			tags = append(tags, tag)
		}
	}
	sort.Strings(tags)
	return TriVersion{
		Index:     [...]uint64{v.Major, v.Minor, v.Patch},
		TagString: strings.Join(tags, ":"),
	}
}

func (v TriVersion) Major() uint64 {
	return v.Index[0]
}

func (v TriVersion) Minor() uint64 {
	return v.Index[1]
}

func (v TriVersion) Patch() uint64 {
	return v.Index[2]
}

func (v TriVersion) Less(x TriVersion) bool {
	if v == x {
		return false
	}
	if v.Index == x.Index {
		return v.TagString < x.TagString
	}
	for i, r := range v.Index {
		if r < x.Index[i] {
			return true
		} else if r > x.Index[i] {
			return false
		}
	}
	return false
}

func (v TriVersion) LessEqual(x TriVersion) bool {
	if v == x {
		return true
	}
	return v.Less(x)
}

func (v TriVersion) CompatibleWith(x TriVersion) bool {
	return v.Major() == x.Major() &&
		v.Minor() <= x.Minor()
}
