package content

// CompletionState mirrors the game's ChallengesService::EChallengeCompletionState
// (Inactive = 0, Active = 1, Completed = 2); NotCompleted is the game's Active.
// https://github.com/OrfeasZ/ZHMTools/blob/099c155a898c8754e1f14bd48745fcfaa4d232ae/Libraries/ResourceLib/Src/Generated/KNT/ZHMGen.h#L362
type CompletionState int

//goland:noinspection ALL
const (
	CompletionState_Inactive     CompletionState = 0
	CompletionState_NotCompleted CompletionState = 1
	CompletionState_Completed    CompletionState = 2
)

type UnlockedState int

//goland:noinspection ALL
const (
	UnlockState_Locked   UnlockedState = 0
	UnlockState_Unlocked UnlockedState = 1
)
