package fastsync

type heightSet struct {
	begin      int64
	end        int64
	additional []int64
}

func newHeightSet(begin int64, end int64) *heightSet {
	return &heightSet{
		begin: begin,
		end:   end,
	}
}

func (hs *heightSet) popLowest() (int64, bool) {
	if len(hs.additional) > 0 {
		var mi int
		var m int64
		for i, v := range hs.additional {
			if i == 0 || v < m {
				m = v
				mi = i
			}
		}
		hs.additional = append(hs.additional[:mi], hs.additional[mi+1:]...)
		return m, true
	}
	if hs.begin <= hs.end {
		res := hs.begin
		hs.begin++
		return res, true
	}
	return -1, false
}

func (hs *heightSet) getLowest() (int64, bool) {
	if len(hs.additional) > 0 {
		var m int64
		for i, v := range hs.additional {
			if i == 0 || v < m {
				m = v
			}
		}
		return m, true
	}
	if hs.begin <= hs.end {
		return hs.begin, true
	}
	return -1, false
}

func (hs *heightSet) add(h int64) {
	if h >= hs.begin {
		panic("bad arg")
	}
	hs.additional = append(hs.additional, h)
}
