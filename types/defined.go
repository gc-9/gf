package types

func PairToMap[L int | string, R string](pairs []PairValue[L, R]) map[L]R {
	m := make(map[L]R)
	for _, p := range pairs {
		m[p.Id] = p.Name
	}
	return m
}

var StatusOptions = []PairValue[int, string]{
	{1, "启用"},
	{-1, "关闭"},
}
var StatusMap = PairToMap(StatusOptions)
