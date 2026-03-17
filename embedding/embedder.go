package embedding

// EmbeddingDim is the vector size produced by paraphrase-multilingual-MiniLM-L12-v2.
const EmbeddingDim = 384

// Embedder converts text into fixed-length floating-point vectors.
// Implementations can swap between local ONNX inference, Python subprocess,
// or remote API without changing callers.
type Embedder interface {
	Embed(text string) ([]float64, error)
	EmbedBatch(texts []string) ([][]float64, error)
}
