package simplykb

func reciprocalRank(rank int, constant int) float64 {
	if rank <= 0 {
		return 0
	}
	if constant <= 0 {
		constant = defaultRRFConstant
	}
	return 1.0 / float64(constant+rank)
}
