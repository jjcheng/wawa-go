package types

type Environment string

const (
	EnvironmentDevelop    Environment = "DEVELOP"
	EnvironmentStaging    Environment = "STAGING"
	EnvironmentProduction Environment = "PRODUCTION"
)
