package main

func getCost(level int) (cost float64) {
	return 0
	switch level {
	case 100:
		return 5
	case 1000:
		return 50
	case 10000:
		return 500
	case 50000:
		return 2500
	default:
		return 0
	}
}
