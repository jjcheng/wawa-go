package types

type BuyerStage string

const (
	BuyerStageExplore  BuyerStage = "Explore"
	BuyerStageEvaluate BuyerStage = "Evaluate"
	BuyerStageValidate BuyerStage = "Validate"
	BuyerStageDecide   BuyerStage = "Decide"
)
