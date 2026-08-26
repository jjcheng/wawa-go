package types

type StorageClass string

const (
	StorageClassStandard StorageClass = "Standard"
	StorageClassCool     StorageClass = "Cool"
	StorageClassArchive  StorageClass = "Archive"
	StorageClassCold     StorageClass = "Cold"
)
