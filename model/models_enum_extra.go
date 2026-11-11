package model

// StrategyType is the type of strategy used for provisioning.
type StrategyType = string

const (
	SingleSourceStrategyType StrategyType = "SingleSourceProvisionStrategy"
	SplitStrategyType        StrategyType = "SplitProvisionStrategy"
	WaterfallStrategyType    StrategyType = "WaterfallProvisionStrategy"
)
