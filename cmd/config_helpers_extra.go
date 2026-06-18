package cmd

func derefInt(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}

func derefInt64(p *int64, def int64) int64 {
	if p == nil {
		return def
	}
	return *p
}
