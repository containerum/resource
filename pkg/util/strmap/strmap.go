package strmap

type StringMap map[string]interface{}

func (strmap StringMap) Len() int {
	return len(strmap)
}

func (strmap StringMap) Copy() StringMap {
	var cp = make(StringMap, strmap.Len())
	for k, v := range strmap {
		cp[k] = v
	}
	return cp
}

func (strmap StringMap) New() StringMap {
	return make(StringMap, strmap.Len())
}

func (strmap StringMap) Filter(pred func(string, interface{}) bool) StringMap {
	var filtered = strmap.New()
	for k, v := range strmap {
		if pred(k, v) {
			filtered[k] = v
		}
	}
	return filtered
}

func (strmap StringMap) Keys() []string {
	var keys = make([]string, strmap.Len())
	for key := range strmap {
		keys = append(keys, key)
	}
	return keys
}

func (strmap StringMap) Set(key string, value interface{}) StringMap {
	strmap = strmap.Copy()
	strmap[key] = value
	return strmap
}
