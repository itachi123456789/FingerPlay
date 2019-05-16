package main

/*
	ResponseCodeOK                  = 0
	ResponseCodeWaitReadyTimeout    = -1
	ResponseCodeWaitMatchTimeout    = -2
	ResponseCodeBadMatchStatus      = -3
	ResponseCodeBadAccessToken      = -4
	ResponseCodeBadReadyStatus      = -5
	ResponseCodeBadAccountStatus    = -6
	ResponseCodeBadRequestFormat    = -7
	ResponseCodeBadOperate          = -8
	ResponseCodeBadMatchId          = -9
	ResponseCodeBadRound            = -10
	ResponseCodeBadLevel            = -11
	ResponseCodeInternalError       = -12
	ResponseCodeInsufficientBalance = -13
	ResponseCodeKickOut             = -14
*/
func getCodeDescription(code int) string {
	switch code {
	case ResponseCodeOK:
		return "OK"
	case ResponseCodeWaitReadyTimeout:
		return "Wait ready timeout"
	case ResponseCodeWaitMatchTimeout:
		return "Wait match timeout"
	case ResponseCodeBadMatchStatus:
		return "Bad match status"
	case ResponseCodeBadAccessToken:
		return "Bad access token"
	case ResponseCodeBadReadyStatus:
		return "Bad ready status"
	case ResponseCodeBadAccountStatus:
		return "Bad account status"
	case ResponseCodeBadRequestFormat:
		return "Bad request format"
	case ResponseCodeBadOperate:
		return "Bad operate"
	case ResponseCodeBadMatchId:
		return "Bad match id"
	case ResponseCodeBadRound:
		return "Bad round"
	case ResponseCodeBadLevel:
		return "Bad level"
	case ResponseCodeInternalError:
		return "Internal error"
	case ResponseCodeInsufficientBalance:
		return "Insufficient balance"
	case ResponseCodeKickOut:
		return "Kick out"
	default:
		return "Undefined"
	}
}

func getOperateDescription(op int) string {
	switch op {
	case Stone:
		return "Stone"
	case Paper:
		return "Paper"
	case Scissors:
		return "Scissors"
	default:
		return "Undefined"
	}
}

func getResultDescription(result int) string {
	switch result {
	case Lost:
		return "Lost"
	case Won:
		return "Won"
	case Draw:
		return "Draw"
	default:
		return "Undefined"
	}
}

func getSign(n float64) string {
	if n >= 0 {
		return "+"
	} else {
		return ""
	}
}
