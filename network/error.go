package network

import (
	"github.com/icon-project/goloop/common/errors"
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
	ErrIllegalArgument           = errors.ErrIllegalArgument
)
