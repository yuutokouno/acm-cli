package embedding

import (
	"fmt"
	"math"
	"os"

	ort "github.com/yalue/onnxruntime_go"
)

// ONNXEmbedder generates sentence embeddings using a local ONNX model.
// It expects the paraphrase-multilingual-MiniLM-L12-v2 model (384-dim output).
//
// Prerequisites:
//   - ONNX Runtime shared library: https://github.com/microsoft/onnxruntime/releases
//   - model.onnx: models/model.onnx
//   - tokenizer.json: models/tokenizer.json
//
// Set libPath to the path of the libonnxruntime shared library (.dylib / .so).
type ONNXEmbedder struct {
	session   *ort.DynamicAdvancedSession
	tokenizer *wordPieceTokenizer
}

var _ Embedder = (*ONNXEmbedder)(nil)

// NewONNXEmbedder initialises the ONNX Runtime and loads the model.
// Returns an error with clear instructions if any prerequisite is missing.
func NewONNXEmbedder(modelPath, tokenizerPath, libPath string) (*ONNXEmbedder, error) {
	for _, p := range []string{modelPath, tokenizerPath, libPath} {
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf(
				"prerequisite not found: %s\n"+
					"Run 'make download-model' to download model files.\n"+
					"See README.md for ONNX Runtime setup instructions.",
				p,
			)
		}
	}

	ort.SetSharedLibraryPath(libPath)
	if err := ort.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("init ONNX Runtime: %w", err)
	}

	tok, err := loadWordPieceTokenizer(tokenizerPath)
	if err != nil {
		return nil, fmt.Errorf("load tokenizer: %w", err)
	}

	// paraphrase-multilingual-MiniLM-L12-v2 input/output names
	inputNames := []string{"input_ids", "attention_mask", "token_type_ids"}
	outputNames := []string{"last_hidden_state"}

	session, err := ort.NewDynamicAdvancedSession(modelPath, inputNames, outputNames, nil)
	if err != nil {
		return nil, fmt.Errorf("load ONNX model: %w", err)
	}

	return &ONNXEmbedder{session: session, tokenizer: tok}, nil
}

// Close releases ONNX Runtime resources.
func (e *ONNXEmbedder) Close() error {
	if err := e.session.Destroy(); err != nil {
		return err
	}
	return ort.DestroyEnvironment()
}

func (e *ONNXEmbedder) Embed(text string) ([]float64, error) {
	inputIDs, attnMask, tokenTypeIDs := e.tokenizer.encode(text)
	seqLen := int64(len(inputIDs))
	shape := ort.NewShape(1, seqLen)

	idsTensor, err := ort.NewTensor(shape, inputIDs)
	if err != nil {
		return nil, fmt.Errorf("create input_ids tensor: %w", err)
	}
	defer idsTensor.Destroy()

	maskTensor, err := ort.NewTensor(shape, attnMask)
	if err != nil {
		return nil, fmt.Errorf("create attention_mask tensor: %w", err)
	}
	defer maskTensor.Destroy()

	typeTensor, err := ort.NewTensor(shape, tokenTypeIDs)
	if err != nil {
		return nil, fmt.Errorf("create token_type_ids tensor: %w", err)
	}
	defer typeTensor.Destroy()

	outShape := ort.NewShape(1, seqLen, EmbeddingDim)
	outTensor, err := ort.NewEmptyTensor[float32](outShape)
	if err != nil {
		return nil, fmt.Errorf("create output tensor: %w", err)
	}
	defer outTensor.Destroy()

	err = e.session.Run(
		[]ort.ArbitraryTensor{idsTensor, maskTensor, typeTensor},
		[]ort.ArbitraryTensor{outTensor},
	)
	if err != nil {
		return nil, fmt.Errorf("ONNX inference: %w", err)
	}

	hidden := outTensor.GetData() // [1 * seqLen * EmbeddingDim]
	return meanPool(hidden, attnMask, int(seqLen)), nil
}

func (e *ONNXEmbedder) EmbedBatch(texts []string) ([][]float64, error) {
	vecs := make([][]float64, len(texts))
	for i, t := range texts {
		v, err := e.Embed(t)
		if err != nil {
			return nil, fmt.Errorf("embed[%d]: %w", i, err)
		}
		vecs[i] = v
	}
	return vecs, nil
}

// meanPool computes the attention-masked mean of hidden states and L2-normalizes.
// hidden layout: [seqLen * EmbeddingDim] (batch size 1 already squeezed).
func meanPool(hidden []float32, attnMask []int64, seqLen int) []float64 {
	sum := make([]float64, EmbeddingDim)
	var maskSum float64

	for i := 0; i < seqLen; i++ {
		if attnMask[i] == 0 {
			continue
		}
		maskSum++
		base := i * EmbeddingDim
		for j := 0; j < EmbeddingDim; j++ {
			sum[j] += float64(hidden[base+j])
		}
	}
	if maskSum == 0 {
		return sum
	}
	for i := range sum {
		sum[i] /= maskSum
	}

	var norm float64
	for _, v := range sum {
		norm += v * v
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return sum
	}
	for i := range sum {
		sum[i] /= norm
	}
	return sum
}
