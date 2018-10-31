package mpt

// first return : length of same bytes
// second return : true if same
func compareHex(a, b []byte) (int, bool) {
	min := len(a)
	same := true
	if min > len(b) {
		min = len(b)
	}

	match := 0
	for ; match < min; match++ {
		if a[match] != b[match] {
			same = false
			break
		}
	}

	if same == true && len(a) != len(b) {
		same = false
	}

	return match, same
}
