// cmd/ingester/main.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"bimbi-backend/internal/config"
)

const (
	sourceDir      = "./source_documents"
	collectionName = "bimbi_knowledge_base"
	chunkSize      = 1000
	chunkOverlap   = 200
	chromaTenant   = "default_tenant"
	chromaDatabase = "default_database"

	geminiEmbedModel      = "gemini-embedding-001"
	geminiEmbedAPIVersion = "v1"
)

type chromaCreateCollectionReq struct {
	Name     string            `json:"name"`
	Metadata map[string]string `json:"metadata"`
}

type chromaAddDocumentsReq struct {
	IDs        []string            `json:"ids"`
	Documents  []string            `json:"documents"`
	Embeddings [][]float32         `json:"embeddings"`
	Metadatas  []map[string]string `json:"metadatas"`
}

type chromaCollection struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var (
	httpClient   = &http.Client{Timeout: 30 * time.Second}
	geminiAPIKey string
)

func main() {
	log.Println("🚀 Bimbi AI — Document Ingestion Pipeline Starting...")
	cfg := config.LoadConfig()
	geminiAPIKey = cfg.GeminiKey
	chromaBaseURL := cfg.ChromaURL

	log.Printf("🗑️  Dropping existing collection '%s'...", collectionName)
	_ = dropCollection(chromaBaseURL, collectionName)

	collectionID, err := ensureCollection(chromaBaseURL, collectionName)
	if err != nil {
		log.Fatalf("FATAL: ChromaDB collection setup failed: %v", err)
	}
	log.Printf("✅ ChromaDB collection ready (ID: %s)", collectionID)

	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		log.Fatalf("FATAL: Cannot read '%s': %v", sourceDir, err)
	}

	totalChunks := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(sourceDir, entry.Name())
		ext := strings.ToLower(filepath.Ext(entry.Name()))

		if ext != ".pdf" && ext != ".txt" {
			continue
		}

		log.Printf("📄 Processing: %s", entry.Name())

		var text string
		if ext == ".pdf" {
			text, err = extractTextFromPDF(filePath)
		} else {
			raw, _ := os.ReadFile(filePath)
			text = string(raw)
		}

		if err != nil || strings.TrimSpace(text) == "" {
			continue
		}

		chunks := splitTextIntoChunks(text, chunkSize, chunkOverlap)
		log.Printf("  ✂️  Split into %d chunks", len(chunks))

		batchSize := 10
		for i := 0; i < len(chunks); i += batchSize {
			end := i + batchSize
			if end > len(chunks) {
				end = len(chunks)
			}
			batch := chunks[i:end]

			embeds32, err := embedTexts(batch)
			if err != nil {
				log.Printf("  ❌ Embedding batch failed: %v", err)
				continue
			}

			ids := make([]string, len(batch))
			metas := make([]map[string]string, len(batch))
			for j := range batch {
				ids[j] = fmt.Sprintf("%s_chunk_%d", sanitizeID(entry.Name()), i+j)
				metas[j] = map[string]string{
					"source": entry.Name(),
					"chunk":  fmt.Sprintf("%d", i+j),
				}
			}

			if err := addToChroma(chromaBaseURL, collectionID, ids, batch, embeds32, metas); err != nil {
				log.Printf("  ❌ Add to Chroma failed: %v", err)
				continue
			}

			totalChunks += len(batch)
			time.Sleep(500 * time.Millisecond)
		}
	}

	log.Printf("\n🎉 Ingestion complete! Total chunks: %d", totalChunks)
}

func embedTexts(texts []string) ([][]float32, error) {
	results := make([][]float32, 0, len(texts))
	for _, text := range texts {
		payload := map[string]interface{}{
			"content":  map[string]interface{}{"parts": []map[string]string{{"text": text}}},
			"taskType": "RETRIEVAL_DOCUMENT",
		}
		body, _ := json.Marshal(payload)
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/%s/models/%s:embedContent?key=%s", geminiEmbedAPIVersion, geminiEmbedModel, geminiAPIKey)
		resp, err := httpClient.Post(url, "application/json", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
			return nil, fmt.Errorf("bad status: %d", resp.StatusCode)
		}
		var res struct {
			Embedding struct {
				Values []float32 `json:"values"`
			} `json:"embedding"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&res)
		resp.Body.Close()
		results = append(results, res.Embedding.Values)
		time.Sleep(100 * time.Millisecond)
	}
	return results, nil
}

func chromaV2Base(baseURL string) string {
	return fmt.Sprintf("%s/api/v2/tenants/%s/databases/%s", baseURL, chromaTenant, chromaDatabase)
}

func dropCollection(baseURL, name string) error {
	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/collections/%s", chromaV2Base(baseURL), name), nil)
	resp, err := httpClient.Do(req)
	if err == nil {
		resp.Body.Close()
	}
	return err
}

func ensureCollection(baseURL, name string) (string, error) {
	v2base := chromaV2Base(baseURL)
	resp, err := httpClient.Get(fmt.Sprintf("%s/collections/%s", v2base, name))
	if err == nil && resp.StatusCode == 200 {
		defer resp.Body.Close()
		var col chromaCollection
		_ = json.NewDecoder(resp.Body).Decode(&col)
		return col.ID, nil
	}
	if resp != nil {
		resp.Body.Close()
	}

	body, _ := json.Marshal(chromaCreateCollectionReq{Name: name, Metadata: map[string]string{"description": "KB"}})
	createResp, err := httpClient.Post(fmt.Sprintf("%s/collections", v2base), "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer createResp.Body.Close()
	var col chromaCollection
	_ = json.NewDecoder(createResp.Body).Decode(&col)
	return col.ID, nil
}

func addToChroma(baseURL, collectionID string, ids, docs []string, embeds [][]float32, metas []map[string]string) error {
	body, _ := json.Marshal(chromaAddDocumentsReq{IDs: ids, Documents: docs, Embeddings: embeds, Metadatas: metas})
	resp, err := httpClient.Post(fmt.Sprintf("%s/collections/%s/add", chromaV2Base(baseURL), collectionID), "application/json", bytes.NewReader(body))
	if err == nil {
		resp.Body.Close()
	}
	return err
}

func splitTextIntoChunks(text string, size, overlap int) []string {
	var chunks []string
	runes := []rune(text)
	for i := 0; i < len(runes); i += size - overlap {
		end := i + size
		if end > len(runes) {
			end = len(runes)
		}
		chunk := strings.TrimSpace(string(runes[i:end]))
		if utf8.RuneCountInString(chunk) > 50 {
			chunks = append(chunks, chunk)
		}
		if end == len(runes) {
			break
		}
	}
	return chunks
}

func extractTextFromPDF(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	content := string(data)
	var sb strings.Builder
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Tj") || strings.Contains(line, "TJ") {
			inString := false
			for i := 0; i < len(line); i++ {
				if line[i] == '(' {
					inString = true
				} else if line[i] == ')' {
					inString = false
				} else if inString && line[i] >= 32 && line[i] < 127 {
					sb.WriteByte(line[i])
				}
			}
			sb.WriteString(" ")
		}
	}
	if len(strings.TrimSpace(sb.String())) < 100 {
		var fallback strings.Builder
		for _, b := range data {
			if b >= 32 && b < 127 || b == '\n' {
				fallback.WriteByte(b)
			}
		}
		return fallback.String(), nil
	}
	return sb.String(), nil
}

func sanitizeID(name string) string {
	name = strings.TrimSuffix(name, filepath.Ext(name))
	var sb strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('_')
		}
	}
	return fmt.Sprintf("%s_%d", sb.String(), rand.Intn(100000))
}
