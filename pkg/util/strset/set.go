package strset

import "encoding/json"

type Set map[string]struct{}

func FromSlice(slice []string) Set {
	var set = make(map[string]struct{}, len(slice))
	for _, item := range slice {
		set[item] = struct{}{}
	}
	return set
}

func (set Set) Len() int {
	return len(set)
}

func (set Set) Put(item string) Set {
	set = set.Copy()
	set[item] = struct{}{}
	return set
}

func (set Set) Delete(item string) Set {
	set = set.Copy()
	delete(set, item)
	return set
}

func (set Set) Copy() Set {
	var cp = make(map[string]struct{}, len(set))
	for item, _ := range set {
		cp[item] = struct{}{}
	}
	return set
}

func (set Set) Sub(x Set) Set {
	set = set.Copy()
	for item := range x {
		delete(set, item)
	}
	return set
}

func (set Set) SubSlice(x []string) Set {
	return set.Sub(FromSlice(x))
}

func (set Set) Add(x Set) Set {
	set = set.Copy()
	for item := range x {
		set.Put(item)
	}
	return set
}

func (set Set) AddSlice(x []string) Set {
	return set.Add(FromSlice(x))
}

func (set Set) Items() []string {
	var items = make([]string, 0, len(set))
	for item := range set {
		items = append(items, item)
	}
	return items
}

func (set Set) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(set.Items(), "", "  ")
}

func (set *Set) UnmarshalJSON(p []byte) error {
	var slSet = make([]string, 0, 16)
	var err = json.Unmarshal(p, &slSet)
	if err != nil {
		return err
	}
	*set = FromSlice(slSet)
	return nil
}

func (set Set) In(item string) bool {
	_, ok := set[item]
	return ok
}
