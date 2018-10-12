package module

type TransitionCallback interface {
	//	Called if error is ocurred.
	OnError(error)

	//	Called if validation is done.
	OnValidate(Transition)

	//	Called if execution is done.
	OnExecute(Transition)
}

type Transition interface {
	Parent() Transition
	//	NextValidators returns the addresses of next validators. 
	//	The function returns nil if the transition is not created by
	//	ServiceManager.ProposeTransition and is not validated yet.
	NextValidators() []Address
	Txs() [][]byte

	//	Execute executes this transition.
	//	The result is asynchronously notified by cb. canceler cancels the
	//	operation. canceler returns true and cb is not called if the
	//	cancellation was successful.
	Execute(cb TransitionCallback) (canceler func()bool, err error)

	GetState() State

	//	GetResult returns execution result.
	//	The function returns nil if the transition execution is not completed.
	GetResult() []byte

	//	LogBloom returns log bloom filter for this transition.
	//	The function returns nil if the transition execution is not completed.
	LogBloom() []byte

	//	ID returns ID of this transition.
	//	This function returns nil if ID is not set yet.
	ID() []byte
}

type State interface {
}

type ServiceManager interface {
	GetTransition(id []byte) Transition
	//	ProposeTransition proposes a Transition following the parent Transition.
	//	Returned Transition always passes validation.
	ProposeTransition(parent Transition) (Transition, error)
	//	CreateTransition creates a Transition following parent Transition.
	CreateTransition(parent Transition, txs [][]byte, patches [][]byte) (Transition, error)
	GetPatches(parent Transition) [][]byte
	//	PatchTransition creates a Transition by adding patch on a transition.
	PatchTransition(transtion Transition, patches [][]byte) Transition
	Commit(Transition, id []byte) error
	Finalize(Transition)
}
