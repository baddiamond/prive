package core

import "errors"

// CodedError is an error the user may see. Code is a translation key in
// frontend/src/locales; Params are interpolation values. The wrapped Err is for logs only.
type CodedError struct {
	Code   string
	Params map[string]string
	Err    error
}

// NewError returns a CodedError with the given translation key.
func NewError(code string) *CodedError {
	return &CodedError{Code: code}
}

// Error returns the code, followed by the cause when there is one (for logs).
func (e *CodedError) Error() string {
	if e.Err != nil {
		return e.Code + ": " + e.Err.Error()
	}
	return e.Code
}

// Unwrap returns the underlying cause.
func (e *CodedError) Unwrap() error { return e.Err }

// Is makes errors.Is match any CodedError with the same code, so wrapped
// variants still match sentinels such as ErrNotLoggedIn.
func (e *CodedError) Is(target error) bool {
	t, ok := target.(*CodedError)
	return ok && t.Code == e.Code
}

// Wrap returns a copy of e carrying cause, for logging.
func (e *CodedError) Wrap(cause error) *CodedError {
	return &CodedError{Code: e.Code, Params: e.Params, Err: cause}
}

// With returns a copy of e with an interpolation parameter added.
func (e *CodedError) With(key, value string) *CodedError {
	params := make(map[string]string, len(e.Params)+1)
	for k, v := range e.Params {
		params[k] = v
	}
	params[key] = value
	return &CodedError{Code: e.Code, Params: params, Err: e.Err}
}

// Translation keys of well-known errors. Every code must exist in frontend/src/locales/en.json.
const (
	CodeUnknown                = "errors.unknown"
	CodeAccountNotFound        = "errors.account.notFound"
	CodeNotLoggedIn            = "errors.account.notLoggedIn"
	CodeConnectFailed          = "errors.account.connectFailed"
	CodeEngineNotFound         = "errors.engine.notFound"
	CodeLoginMethodUnsupported = "errors.engine.loginMethodUnsupported"
	CodeCapabilityUnsupported  = "errors.engine.capabilityUnsupported"
	CodeNotImplemented         = "errors.engine.notImplemented"
	CodeSendFailed             = "errors.message.sendFailed"
	CodeShuttingDown           = "errors.app.shuttingDown"
	// CodeHistoryUnavailable means older messages cannot be asked for this chat, e.g. it
	// has no stored message to start from.
	CodeHistoryUnavailable = "errors.history.unavailable"
	// CodeHistoryNoAnswer means the network did not answer a history request in time,
	// e.g. the phone is offline.
	CodeHistoryNoAnswer = "errors.history.noAnswer"
	// CodeMessageEmpty means the user tried to send a message without text.
	CodeMessageEmpty = "errors.message.empty"
	// CodeMessageTooLong means the text is longer than MaxMessageLength.
	CodeMessageTooLong = "errors.message.tooLong"
)

// Sentinel errors for use with errors.Is.
var (
	ErrUnknown                = NewError(CodeUnknown)
	ErrAccountNotFound        = NewError(CodeAccountNotFound)
	ErrNotLoggedIn            = NewError(CodeNotLoggedIn)
	ErrConnectFailed          = NewError(CodeConnectFailed)
	ErrEngineNotFound         = NewError(CodeEngineNotFound)
	ErrLoginMethodUnsupported = NewError(CodeLoginMethodUnsupported)
	ErrCapabilityUnsupported  = NewError(CodeCapabilityUnsupported)
	ErrNotImplemented         = NewError(CodeNotImplemented)
	ErrSendFailed             = NewError(CodeSendFailed)
	ErrShuttingDown           = NewError(CodeShuttingDown)
	ErrHistoryUnavailable     = NewError(CodeHistoryUnavailable)
	ErrMessageEmpty           = NewError(CodeMessageEmpty)
	ErrMessageTooLong         = NewError(CodeMessageTooLong)
)

// KnownCodes lists every translation key the core may send to the UI,
// including the account state keys. Tests check them against the locale files.
func KnownCodes() []string {
	codes := []string{
		CodeUnknown, CodeAccountNotFound, CodeNotLoggedIn, CodeConnectFailed,
		CodeEngineNotFound, CodeLoginMethodUnsupported, CodeCapabilityUnsupported,
		CodeNotImplemented, CodeSendFailed, CodeShuttingDown, CodeHistoryUnavailable, CodeHistoryNoAnswer,
		CodeMessageEmpty, CodeMessageTooLong,
	}
	for _, s := range accountStates {
		codes = append(codes, s.Key())
	}
	return codes
}

// AsCoded returns err as a CodedError, or ErrUnknown wrapping err if it is not one.
func AsCoded(err error) *CodedError {
	if err == nil {
		return nil
	}
	var ce *CodedError
	if errors.As(err, &ce) {
		return ce
	}
	return ErrUnknown.Wrap(err)
}
