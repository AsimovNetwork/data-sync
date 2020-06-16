package response

import "fmt"

type DataNotExistError struct {
	Code uint16
	Msg  string
}

type CallBlockChainError struct {
	Code uint16
	Msg  string
}

func (err *DataNotExistError) Error() string {
	return fmt.Sprintf(" DataNoExistError code = %d, msg = %s", err.Code, err.Msg)
}

func NewDataNoExistError() *DataNotExistError {
	return &DataNotExistError{
		Code: DataNotExist,
		Msg:  errorInfo[DataNotExist],
	}
}

func IsDataNotExistError(err interface{}) bool {
	switch err.(type) {
	case *DataNotExistError:
		return true
	default:
		return false
	}
}

func (err *CallBlockChainError) Error() string {
	return fmt.Sprintf(" CallBlockChainError code = %d, msg = %s", err.Code, err.Msg)
}

func NewCallBlockChainError() *CallBlockChainError {
	return &CallBlockChainError{
		Code: CallBlockChainFailed,
		Msg:  errorInfo[CallBlockChainFailed],
	}
}

func IsCallBlockChainError(err interface{}) bool {
	switch err.(type) {
	case *CallBlockChainError:
		return true
	default:
		return false
	}
}
