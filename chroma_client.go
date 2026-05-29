package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ─── ChromaDB HTTP Client (API v2) ───────────────────────────────────────────
// ChromaDB 1.0+ uses the v2 API with tenant/database scoping.
// Default tenant: "default_tenant", Default database: "default_database"
// Since our collection uses manually provided embeddings (no built-in embedding fn),
// we must embed the query ourselves and pass query_embeddings to ChromaDB.

const (
	chromaCollectionName = "bimbi_knowledge_base"
	chromaTenant         = "default_tenant"
	chromaDatabase       = "default_database"

	// Gemini embedding model (must match what was used during ingestion)
	queryEmbedModel      = "gemini-embedding-001"
	queryEmbedAPIVersion = "v1"
)

// chromaQueryByEmbeddingRequest is the payload for ChromaDB /query when
// using pre-computed embeddings instead of text.
type chromaQueryByEmbeddingRequest struct {
	QueryEmbeddings [][]float32 `json:"query_embeddings"`
	NResults        int         `json:"n_results"`
	Include         []string    `json:"include"`
}

// chromaQueryResponse holds the response from ChromaDB query.
type chromaQueryResponse struct {
	Documents [][]string            `json:"documents"`
	Distances [][]float64           `json:"distances"`
	IDs       [][]string            `json:"ids"`
	Metadatas [][]map[string]string `json:"metadatas"`
}

// chromaCollection holds collection metadata.
type chromaCollection struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// httpClient is a shared HTTP client with timeout.
var httpClient = &http.Client{Timeout: 30 * time.Second}

// chromaV2Base returns the base URL for the v2 tenant+database scoped API.
func chromaV2Base() string {
	return fmt.Sprintf("%s/api/v2/tenants/%s/databases/%s", chromaURL, chromaTenant, chromaDatabase)
}

// queryChromaDB retrieves the top-k most relevant document chunks from ChromaDB.
// It first embeds the query text using Gemini, then sends the vector to ChromaDB.
// Returns: (ragContext string, sources []string, error)
func queryChromaDB(_ context.Context, queryText string, topK int) (string, []string, error) {
	// Step 1: Embed the query text using Gemini
	queryEmbedding, err := embedQuery(queryText)
	if err != nil {
		return "", nil, fmt.Errorf("embed query: %w", err)
	}

	// Step 2: Get collection ID
	collectionID, err := getCollectionID()
	if err != nil {
		return "", nil, fmt.Errorf("getCollectionID: %w", err)
	}

	// Step 3: Query ChromaDB with the embedding vector
	reqBody := chromaQueryByEmbeddingRequest{
		QueryEmbeddings: [][]float32{queryEmbedding},
		NResults:        topK,
		Include:         []string{"documents", "distances", "metadatas"},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, fmt.Errorf("marshal query request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/query", chromaV2Base(), collectionID)
	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", nil, fmt.Errorf("http POST to ChromaDB: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", nil, fmt.Errorf("chromaDB query returned %d: %s", resp.StatusCode, string(body))
	}

	var qResp chromaQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&qResp); err != nil {
		return "", nil, fmt.Errorf("decode ChromaDB response: %w", err)
	}

	if len(qResp.Documents) == 0 || len(qResp.Documents[0]) == 0 {
		return "Tidak ada dokumen relevan ditemukan di knowledge base.", []string{}, nil
	}

	// Kumpulkan nama file sumber (unik) dari metadata
	sourceSet := make(map[string]struct{})
	var sources []string
	for _, meta := range qResp.Metadatas[0] {
		if src, ok := meta["source"]; ok && src != "" {
			if _, seen := sourceSet[src]; !seen {
				sourceSet[src] = struct{}{}
				sources = append(sources, src)
			}
		}
	}

	// Gabungkan chunks menjadi satu string konteks
	var sb strings.Builder
	for i, doc := range qResp.Documents[0] {
		sb.WriteString(fmt.Sprintf("[Konteks %d]\n%s\n\n", i+1, doc))
	}

	return strings.TrimSpace(sb.String()), sources, nil
}

// embedQuery calls the Gemini embedContent API to embed a single query string.
func embedQuery(text string) ([]float32, error) {
	payload := map[string]interface{}{
		"content": map[string]interface{}{
			"parts": []map[string]string{
				{"text": text},
			},
		},
		"taskType": "RETRIEVAL_QUERY", // Use RETRIEVAL_QUERY for search queries
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal embed payload: %w", err)
	}

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/%s/models/%s:embedContent?key=%s",
		queryEmbedAPIVersion, queryEmbedModel, geminiKey,
	)

	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gemini embedContent POST: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini embedContent returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Embedding struct {
			Values []float32 `json:"values"`
		} `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode embedContent response: %w", err)
	}

	return result.Embedding.Values, nil
}

// getCollectionID fetches the ChromaDB collection ID by name using v2 API.
func getCollectionID() (string, error) {
	url := fmt.Sprintf("%s/collections/%s", chromaV2Base(), chromaCollectionName)
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("http GET collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ChromaDB GET collection returned %d: %s", resp.StatusCode, string(body))
	}

	var col chromaCollection
	if err := json.NewDecoder(resp.Body).Decode(&col); err != nil {
		return "", fmt.Errorf("decode collection response: %w", err)
	}

	return col.ID, nil
}
