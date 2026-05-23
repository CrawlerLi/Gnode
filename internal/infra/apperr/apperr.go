package apperr

type ErrorCode int

const (
	//业务错误码
	ErrInvaidTX = iota
	ErrInsufficientFunds
	ErrNotFundTX
)

var ErrorCodeStrings = map[ErrorCode]string{
	ErrInvaidTX:          "ErrInvaildTX",
	ErrInsufficientFunds: "ErrInsufficientFunds",
	ErrNotFundTX:         "ErrNotFundTX",
}

type Error struct {
	ErrorCode  ErrorCode
	Descripton string
	Err        error
}

func (e Error) Error() string {
	if e.Err != nil {
		return e.Descripton + ": " + e.Err.Error()
	}
	return e.Descripton
}

func makeError(c ErrorCode, desc string, err error) Error {
	return Error{ErrorCode: c, Descripton: desc, Err: err}
}
