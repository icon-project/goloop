package icutils

func MergeMaps(maps ...map[string]interface{}) map[string]interface{} {
	size := len(maps)
	if size == 0 {
		return nil
	}

	if size == 1 {
		return maps[0]
	}

	ret := maps[0]
	for i := 1; i <= size; i++ {
		for k, v := range maps[i] {
			ret[k] = v
		}
	}

	return ret
}
