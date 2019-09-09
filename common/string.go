package common

func StrLeft(n int, s string) string {
	if len(s) > n {
		return string([]byte(s)[0:n])
	}
	return s
}
