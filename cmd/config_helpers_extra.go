package cmd

func derefInt(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}
