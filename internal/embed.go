package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

var embeddingsURL = "http://localhost:8080/embed"

func init() {
	// Allow override via environment variable
	if url := os.Getenv("EMBEDDINGS_URL"); url != "" {
		embeddingsURL = url
	}
}

type teiRequest struct {
	Inputs string `json:"inputs"`
}

// GetEmbedding returns a 768-dim embedding for a query (search)
func GetEmbedding(text string) ([]float64, error) {
	return getEmbeddingWithPrefix("search_query: " + text)
}

// GetDocumentEmbedding returns a 768-dim embedding for storing a document
func GetDocumentEmbedding(text string) ([]float64, error) {
	return getEmbeddingWithPrefix("search_document: " + text)
}

func getEmbeddingWithPrefix(text string) ([]float64, error) {
	reqBody := teiRequest{
		Inputs: text,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", embeddingsURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embeddings service unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embeddings service error: %d", resp.StatusCode)
	}

	// TEI returns [[float, float, ...]] for single input
	var result [][]float64
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result) == 0 || len(result[0]) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return result[0], nil
}
