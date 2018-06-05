package diff

import (
	"strconv"

	"github.com/blang/semver"
)

type TriVersion [3]uint64

func (v TriVersion) String() string {
	return "v" +
		strconv.FormatUint(v[0], 10) + "." +
		strconv.FormatUint(v[1], 10) + "." +
		strconv.FormatUint(v[2], 10)
}

func FromVersion(v semver.Version) TriVersion {
	return TriVersion{v.Major, v.Minor, v.Patch}
}

func (v TriVersion) Major() uint64 {
	return v[0]
}

func (v TriVersion) Minor() uint64 {
	return v[1]
}

func (v TriVersion) Patch() uint64 {
	return v[2]
}

func (v TriVersion) Less(x TriVersion) bool {
	for i, r := range v {
		if r < x[i] {
			return true
		} else if r > x[i] {
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
