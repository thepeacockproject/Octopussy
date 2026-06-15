package content

type CompletionState int

const (
	CompletionState_NotCompleted CompletionState = 1
	CompletionState_Completed    CompletionState = 2
)

type UnlockedState int

const (
	UnlockState_Locked   UnlockedState = 0
	UnlockState_Unlocked UnlockedState = 1
)
