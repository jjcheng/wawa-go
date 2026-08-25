package types

type PNHookCategory string

const (
	PNHookCategoryProjectTransactions       PNHookCategory = "project_transactions"        // 6 units sold last 7 days
	PHHookCategoryProjectStats              PNHookCategory = "project_stats"               // avg price is
	PNHookCategoryProjectInfo               PNHookCategory = "project_info"                // largest land size
	PNHookCategoryProjectLocation           PNHookCategory = "project_location"            // lorong chuan mrt is
	PNHookCategoryProjectUnits              PNHookCategory = "project_units"               // there are 30 3 bedders available
	PNHookCategoryProjectStacks             PNHookCategory = "project_stacks"              // quiet stacks are
	PHHookCategoryNearbyProjectTransactions PNHookCategory = "nearby_project_transactions" // nearby projects are

)
