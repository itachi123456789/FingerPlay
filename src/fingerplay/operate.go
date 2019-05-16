package main

func getWonOperate(op int) int {
	if op == Scissors {
		return Stone
	} else if op == Stone {
		return Paper
	} else {
		return Scissors
	}
}
