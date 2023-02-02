package network

import (
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

const (
	AlreadyListenedError = errors.CodeNetwork + iota
	AlreadyClosedError
	AlreadyDialingError
	AlreadyRegisteredReactorError
	AlreadyRegisteredProtocolError
	NotRegisteredReactorError
	NotRegisteredProtocolError
	NotRegisteredRoleError
	NotAuthorizedError
	NotAvailableError
	NotStartedError
	QueueOverflowError
	DuplicatedPacketError
	DuplicatedPeerError
	InvalidMessageSequenceError
	InvalidSignatureError
)

var (
	ErrAlreadyListened           = errors.NewBase(AlreadyListenedError, "AlreadyListened")
	ErrAlreadyClosed             = errors.NewBase(AlreadyClosedError, "AlreadyClosed")
	ErrAlreadyDialing            = errors.NewBase(AlreadyDialingError, "AlreadyDialing")
	ErrAlreadyRegisteredReactor  = errors.NewBase(AlreadyRegisteredReactorError, "AlreadyRegisteredReactor")
	ErrAlreadyRegisteredProtocol = errors.NewBase(AlreadyRegisteredProtocolError, "AlreadyRegisteredProtocol")
	ErrNotRegisteredReactor      = errors.NewBase(NotRegisteredReactorError, "NotRegisteredReactor")
	ErrNotRegisteredProtocol     = errors.NewBase(NotRegisteredProtocolError, "NotRegisteredProtocol")
	ErrNotRegisteredRole         = errors.NewBase(NotRegisteredRoleError, "NotRegisteredRole")
	ErrNotAuthorized             = errors.NewBase(NotAuthorizedError, "NotAuthorized")
	ErrNotAvailable              = errors.NewBase(NotAvailableError, "NotAvailable")
	ErrNotStarted                = errors.NewBase(NotStartedError, "NotStarted")
	ErrQueueOverflow             = errors.NewBase(QueueOverflowError, "QueueOverflow")
	ErrDuplicatedPacket          = errors.NewBase(DuplicatedPacketError, "DuplicatedPacket")
	ErrDuplicatedPeer            = errors.NewBase(DuplicatedPeerError, "DuplicatedPeer")
	ErrInvalidMessageSequence    = errors.NewBase(InvalidMessageSequenceError, "InvalidMessageSequence")
	ErrInvalidSignature          = errors.NewBase(InvalidSignatureError, "InvalidSignatureError")
	ErrIllegalArgument           = errors.ErrIllegalArgument
)

type Error struct {
	error
	IsTemporary       bool
	Operation         string
	OperationArgument interface{}
}

func (e *Error) Temporary() bool { return e.IsTemporary }

func (e *Error) Unwrap() error { return e.error }

func newNetworkError(err error, op string, opArg interface{}) module.NetworkError {
	if err != nil {
		isTemporary := false
		if QueueOverflowError.Equals(err) {
			isTemporary = true
		}
		return &Error{err, isTemporary, op, opArg}
	}
	return nil
}
