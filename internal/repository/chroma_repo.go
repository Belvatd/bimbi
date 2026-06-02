package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"bimbi-backend/internal/domain"
)

const (
	chromaCollectionName = "bimbi_knowledge_base"
	chromaTenant         = "default_tenant"
	chromaDatabase       = "default_database"

	queryEmbedModel      = "gemini-embedding-001"
	queryEmbedAPIVersion = "v1"
)

type chromaQueryByEmbeddingRequest struct {
	QueryEmbeddings [][]float32 `json:"query_embeddings"`
	NResults        int         `json:"n_results"`
	Include         []string    `json:"include"`
}

type chromaQueryResponse struct {
	Documents [][]string            `json:"documents"`
	Distances [][]float64           `json:"distances"`
	IDs       [][]string            `json:"ids"`
	Metadatas [][]map[string]string `json:"metadatas"`
}

type chromaCollection struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type chromaRepo struct {
	chromaURL  string
	geminiKey  string
	httpClient *http.Client
}

func NewChromaRepo(chromaURL, geminiKey string) domain.VectorRepo {
	return &chromaRepo{
		chromaURL:  chromaURL,
		geminiKey:  geminiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (r *chromaRepo) chromaV2Base() string {
	return fmt.Sprintf("%s/api/v2/tenants/%s/databases/%s", r.chromaURL, chromaTenant, chromaDatabase)
}

func (r *chromaRepo) Query(ctx context.Context, queryText string, topK int) (string, []string, error) {
	queryEmbedding, err := r.embedQuery(queryText)
	if err != nil {
		return "", nil, fmt.Errorf("embed query: %w", err)
	}

	collectionID, err := r.getCollectionID()
	if err != nil {
		return "", nil, fmt.Errorf("getCollectionID: %w", err)
	}

	reqBody := chromaQueryByEmbeddingRequest{
		QueryEmbeddings: [][]float32{queryEmbedding},
		NResults:        topK,
		Include:         []string{"documents", "distances", "metadatas"},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, fmt.Errorf("marshal query request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/query", r.chromaV2Base(), collectionID)
	resp, err := r.httpClient.Post(url, "application/json", bytes.NewReader(bodyBytes))
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

	var sb strings.Builder
	for i, doc := range qResp.Documents[0] {
		sourceName := ""
		if i < len(qResp.Metadatas[0]) {
			if src, ok := qResp.Metadatas[0][i]["source"]; ok && src != "" {
				sourceName = src
			}
		}
		if sourceName != "" {
			sb.WriteString(fmt.Sprintf("[Konteks %d — Sumber: %s]\n%s\n\n", i+1, sourceName, doc))
		} else {
			sb.WriteString(fmt.Sprintf("[Konteks %d]\n%s\n\n", i+1, doc))
		}
	}

	return strings.TrimSpace(sb.String()), sources, nil
}

func (r *chromaRepo) embedQuery(text string) ([]float32, error) {
	payload := map[string]interface{}{
		"content": map[string]interface{}{
			"parts": []map[string]string{
				{"text": text},
			},
		},
		"taskType": "RETRIEVAL_QUERY",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal embed payload: %w", err)
	}

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/%s/models/%s:embedContent?key=%s",
		queryEmbedAPIVersion, queryEmbedModel, r.geminiKey,
	)

	resp, err := r.httpClient.Post(url, "application/json", bytes.NewReader(body))
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

func (r *chromaRepo) getCollectionID() (string, error) {
	url := fmt.Sprintf("%s/collections/%s", r.chromaV2Base(), chromaCollectionName)
	resp, err := r.httpClient.Get(url)
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
