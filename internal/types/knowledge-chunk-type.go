package types

type KnowledgeChunkType string

const (
	KnowledgeChunkTypeText     KnowledgeChunkType = "TEXT"
	KnowledgeChunkTypeImage    KnowledgeChunkType = "IMAGE"
	KnowledgeChunkTypeDocument KnowledgeChunkType = "DOCUMENT"
	KnowledgeChunkTypeVideo    KnowledgeChunkType = "VIDEO"
)

var KnowledgeChunkTypes = []KnowledgeChunkType{
	KnowledgeChunkTypeImage,
	KnowledgeChunkTypeDocument,
	KnowledgeChunkTypeVideo,
	KnowledgeChunkTypeText,
}
