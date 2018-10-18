package mpt

func compareHex(a, b []byte) int {
	min := len(a)
	if min > len(b) {
		min = len(b)
	}

	match := 0
	for ; match < min; match++ {
		if a[match] != b[match] {
			break
		}
	}
	return match
}
