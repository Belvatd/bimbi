//go:build ignore
// +build ignore

// ingestion/ingest.go — Standalone script to process PDF documents and store
// their embeddings into ChromaDB. Run with: go run ingestion/ingest.go
// Supports ChromaDB 1.0+ (v2 API with tenant/database scoping)
// Uses Gemini text-embedding-004 via direct REST API.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/joho/godotenv"
)

// ─── Configuration ────────────────────────────────────────────────────────────

const (
	sourceDir      = "./source_documents"
	collectionName = "bimbi_knowledge_base"
	chunkSize      = 1000 // characters per chunk
	chunkOverlap   = 200  // overlap between chunks
	// ChromaDB v2 API tenant/database (defaults for self-hosted)
	chromaTenant   = "default_tenant"
	chromaDatabase = "default_database"
)

// ─── Structs ─────────────────────────────────────────────────────────────────

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

// geminiEmbedRequest is the payload for Gemini Embedding API (batch mode).
type geminiEmbedRequest struct {
	Requests []geminiEmbedContentReq `json:"requests"`
}

type geminiEmbedContentReq struct {
	Model   string         `json:"model"`
	Content geminiContent  `json:"content"`
	TaskType string        `json:"taskType"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiEmbedBatchResponse struct {
	Embeddings []struct {
		Values []float32 `json:"values"`
	} `json:"embeddings"`
}

// geminiAPIKey holds the Google API key (set in main).
var geminiAPIKey string

// ─── Main ─────────────────────────────────────────────────────────────────────

func main() {
	log.Println("🚀 Bimbi AI — Document Ingestion Pipeline Starting...")

	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env not found, using system env")
	}

	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		log.Fatal("FATAL: GOOGLE_API_KEY is not set")
	}
	geminiAPIKey = apiKey

	chromaBaseURL := os.Getenv("CHROMADB_URL")
	if chromaBaseURL == "" {
		chromaBaseURL = "http://localhost:8000"
	}

	_ = context.Background() // kept for consistency

	// Drop existing collection to prevent duplicates
	log.Printf("🗑️  Dropping existing collection '%s' to prevent duplicates...", collectionName)
	if err := dropCollection(chromaBaseURL, collectionName); err != nil {
		log.Printf("Warning: could not drop collection: %v (may not exist yet)", err)
	}

	// Ensure ChromaDB collection exists
	collectionID, err := ensureCollection(chromaBaseURL, collectionName)
	if err != nil {
		log.Fatalf("FATAL: ChromaDB collection setup failed: %v", err)
	}
	log.Printf("✅ ChromaDB collection '%s' ready (ID: %s)", collectionName, collectionID)

	// Read source documents
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
			log.Printf("  ⏭ Skipping unsupported file: %s", entry.Name())
			continue
		}

		log.Printf("📄 Processing: %s", entry.Name())

		var text string
		if ext == ".pdf" {
			text, err = extractTextFromPDF(filePath)
		} else {
			var raw []byte
			raw, err = os.ReadFile(filePath)
			text = string(raw)
		}

		if err != nil {
			log.Printf("  ❌ Error reading file: %v", err)
			continue
		}

		if strings.TrimSpace(text) == "" {
			log.Printf("  ⚠️  No text extracted from: %s", entry.Name())
			continue
		}

		chunks := splitTextIntoChunks(text, chunkSize, chunkOverlap)
		log.Printf("  ✂️  Split into %d chunks", len(chunks))

		// Generate embeddings in batches of 10
		batchSize := 10
		for i := 0; i < len(chunks); i += batchSize {
			end := i + batchSize
			if end > len(chunks) {
				end = len(chunks)
			}
			batch := chunks[i:end]

			embeds32, err := embedTexts(batch)
			if err != nil {
				log.Printf("  ❌ Embedding batch %d-%d failed: %v", i, end, err)
				continue
			}

			// Create IDs and metadatas
			ids := make([]string, len(batch))
			metadatas := make([]map[string]string, len(batch))
			for j := range batch {
				ids[j] = fmt.Sprintf("%s_chunk_%d", sanitizeID(entry.Name()), i+j)
				metadatas[j] = map[string]string{
					"source": entry.Name(),
					"chunk":  fmt.Sprintf("%d", i+j),
				}
			}

			if err := addToChroma(chromaBaseURL, collectionID, ids, batch, embeds32, metadatas); err != nil {
				log.Printf("  ❌ Failed to add batch to ChromaDB: %v", err)
				continue
			}

			totalChunks += len(batch)
			log.Printf("  ✅ Stored chunks %d–%d", i, end-1)

			// Rate limiting: small delay between batches
			time.Sleep(500 * time.Millisecond)
		}
	}

	log.Printf("\n🎉 Ingestion complete! Total chunks stored: %d", totalChunks)
}

// ─── Gemini Embedding API ─────────────────────────────────────────────────────

// gemini-embedding-001 supports embedContent via v1 API.
const geminiEmbedModel = "gemini-embedding-001"
const geminiEmbedAPIVersion = "v1"

// embedTexts generates embeddings for a batch of texts using Gemini embedContent API.
// Uses individual calls per text since batchEmbedContents is not available on all keys.
func embedTexts(texts []string) ([][]float32, error) {
	results := make([][]float32, 0, len(texts))

	for _, text := range texts {
		payload := map[string]interface{}{
			"content": map[string]interface{}{
				"parts": []map[string]string{
					{"text": text},
				},
			},
			"taskType": "RETRIEVAL_DOCUMENT",
		}

		body, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal embed request: %w", err)
		}

		url := fmt.Sprintf(
			"https://generativelanguage.googleapis.com/%s/models/%s:embedContent?key=%s",
			geminiEmbedAPIVersion, geminiEmbedModel, geminiAPIKey,
		)

		resp, err := httpClient.Post(url, "application/json", bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("gemini embedContent POST: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("gemini embedContent returned %d: %s", resp.StatusCode, string(respBody))
		}

		var result struct {
			Embedding struct {
				Values []float32 `json:"values"`
			} `json:"embedding"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode embedContent response: %w", err)
		}
		resp.Body.Close()

		results = append(results, result.Embedding.Values)

		// Small delay to avoid rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	return results, nil
}

// ─── ChromaDB Helpers (v2 API) ────────────────────────────────────────────────

var httpClient = &http.Client{Timeout: 30 * time.Second}

// chromaV2Base returns the scoped base URL for ChromaDB v2.
func chromaV2Base(baseURL string) string {
	return fmt.Sprintf("%s/api/v2/tenants/%s/databases/%s", baseURL, chromaTenant, chromaDatabase)
}

// dropCollection deletes the ChromaDB collection if it exists.
func dropCollection(baseURL, name string) error {
	url := fmt.Sprintf("%s/collections/%s", chromaV2Base(baseURL), name)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("create DELETE request: %w", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE collection: %w", err)
	}
	defer resp.Body.Close()
	// 200 OK or 404 (doesn't exist) are both acceptable
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("DELETE collection returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ensureCollection creates the ChromaDB collection if it doesn't exist, and returns its ID.
func ensureCollection(baseURL, name string) (string, error) {
	v2base := chromaV2Base(baseURL)

	// Check if collection already exists
	getURL := fmt.Sprintf("%s/collections/%s", v2base, name)
	resp, err := httpClient.Get(getURL)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		var col chromaCollection
		if err := json.NewDecoder(resp.Body).Decode(&col); err == nil && col.ID != "" {
			log.Printf("  ℹ️  Collection '%s' already exists (ID: %s)", name, col.ID)
			return col.ID, nil
		}
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Create the collection
	createURL := fmt.Sprintf("%s/collections", v2base)
	body, _ := json.Marshal(chromaCreateCollectionReq{
		Name:     name,
		Metadata: map[string]string{"description": "Bimbi AI Educational Psychology Knowledge Base"},
	})

	createResp, err := httpClient.Post(createURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create collection POST: %w", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusOK && createResp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(createResp.Body)
		return "", fmt.Errorf("ChromaDB create collection returned %d: %s", createResp.StatusCode, string(bodyBytes))
	}

	var col chromaCollection
	if err := json.NewDecoder(createResp.Body).Decode(&col); err != nil {
		return "", fmt.Errorf("decode create collection response: %w", err)
	}

	return col.ID, nil
}

// addToChroma sends documents and their embeddings to ChromaDB.
func addToChroma(baseURL, collectionID string, ids, docs []string, embeds [][]float32, metas []map[string]string) error {
	url := fmt.Sprintf("%s/collections/%s/add", chromaV2Base(baseURL), collectionID)
	reqBody := chromaAddDocumentsReq{
		IDs:        ids,
		Documents:  docs,
		Embeddings: embeds,
		Metadatas:  metas,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal add request: %w", err)
	}

	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("http POST add: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ChromaDB add returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ─── Text Processing ─────────────────────────────────────────────────────────

// splitTextIntoChunks splits text into overlapping chunks of a given character size.
func splitTextIntoChunks(text string, size, overlap int) []string {
	var chunks []string
	runes := []rune(text)
	total := len(runes)

	for start := 0; start < total; start += size - overlap {
		end := start + size
		if end > total {
			end = total
		}
		chunk := strings.TrimSpace(string(runes[start:end]))
		if utf8.RuneCountInString(chunk) > 50 { // skip tiny chunks
			chunks = append(chunks, chunk)
		}
		if end == total {
			break
		}
	}

	return chunks
}

// extractTextFromPDF extracts plain text from a PDF file.
// Since pure Go PDF libraries have limitations, we use pdftotext if available,
// or fall back to reading raw bytes and extracting visible ASCII text.
func extractTextFromPDF(filePath string) (string, error) {
	// Attempt 1: use pdftotext (part of poppler-utils, commonly available)
	// We read the file and do basic text extraction as fallback
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read PDF: %w", err)
	}

	// Extract readable ASCII/UTF-8 text from raw PDF bytes
	// This is a best-effort extraction without a full PDF parser
	var sb strings.Builder
	content := string(data)

	// Extract text between BT (Begin Text) and ET (End Text) markers in PDF
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Extract string literals from PDF stream (Tj, TJ operators)
		if strings.Contains(line, "Tj") || strings.Contains(line, "TJ") {
			extracted := extractPDFString(line)
			if extracted != "" {
				sb.WriteString(extracted)
				sb.WriteString(" ")
			}
		}
	}

	result := sb.String()

	// If extraction yielded very little, return raw readable content
	if len(strings.TrimSpace(result)) < 100 {
		var fallback strings.Builder
		for _, b := range data {
			if b >= 32 && b < 127 {
				fallback.WriteByte(b)
			} else if b == '\n' || b == '\r' {
				fallback.WriteByte('\n')
			}
		}
		return fallback.String(), nil
	}

	return result, nil
}

// extractPDFString extracts text content from a PDF text operation line.
func extractPDFString(line string) string {
	var result strings.Builder
	inString := false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if ch == '(' {
			inString = true
			continue
		}
		if ch == ')' {
			inString = false
			continue
		}
		if inString && ch >= 32 && ch < 127 {
			result.WriteByte(ch)
		}
	}
	return result.String()
}

// sanitizeID creates a safe document ID from a filename.
func sanitizeID(name string) string {
	// Remove extension, replace non-alphanumeric with underscore
	name = strings.TrimSuffix(name, filepath.Ext(name))
	var sb strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('_')
		}
	}
	// Append random suffix to avoid collisions
	sb.WriteString(fmt.Sprintf("_%d", rand.Intn(100000)))
	return sb.String()
}
