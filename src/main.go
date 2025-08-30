package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	// 실제 존재하는 공식 SDK 경로
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- 데이터 구조 정의 (Structs) ---

type SearchSchema struct {
	Query string `json:"query" jsonschema:"Korean keywords for searching places in South Korea."`
}

type MapDocument struct {
	PlaceName    string `json:"place_name"`
	AddressName  string `json:"address_name"`
	CategoryName string `json:"category_name"`
	PlaceURL     string `json:"place_url"`
	Phone        string `json:"phone"`
}

type KakaoLocalSearchResponse struct {
	Documents []MapDocument `json:"documents"`
}

type WebDocument struct {
	Title    string `json:"title"`
	Contents string `json:"contents"`
}

type DaumWebSearchResponse struct {
	Documents []WebDocument `json:"documents"`
}

type ImageDocument struct {
	ImageURL string `json:"image_url"`
}

type DaumImageSearchResponse struct {
	Documents []ImageDocument `json:"documents"`
}

type KakaoLocalSearchResult struct {
	PlaceName    string        `json:"place_name"`
	AddressName  string        `json:"address_name"`
	CategoryName string        `json:"category_name"`
	PlaceURL     string        `json:"place_url"`
	Phone        string        `json:"phone"`
	ImageURL     string        `json:"image_url"`
	Comments     []WebDocument `json:"comments"`
}

// --- 전역 변수 및 상수 ---

var (
	kakaoAPIKey string
	httpClient  = &http.Client{Timeout: 10 * time.Second}
	logger      *slog.Logger
)

const (
	kakaoMapURL   = "https://dapi.kakao.com/v2/local/search/keyword.json"
	daumSearchURL = "https://dapi.kakao.com/v2/search"
	systemPrompt  = `
Using the provided JSON results, compile a detailed and visually appealing Markdown summary for the user.

Each place **MUST** include:

## [{place_name}]({place_url})

![Image]({image_url})

- **Address**: {address_name}
- **Category**: {category_name}
- **Contact**: {phone}
- **Summary**: Briefly summarize the overall sentiment or notable points based on provided comments. Consider aspects such as positive features, negative issues, and unique highlights.

Note:
- The summary should be directly derived by analyzing and condensing the provided comments.
- Ensure all listed elements (title with link, image, address, category, contact, and summary) are always included for every place.`
)

// --- Kakao API 호출 헬퍼 함수 ---

func makeKakaoRequest(ctx context.Context, url string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "KakaoAK "+kakaoAPIKey)

	logger.Debug("Making Kakao API request", "url", url)
	resp, err := httpClient.Do(req)
	if err != nil {
		logger.Error("Kakao API request failed", "url", url, "error", err)
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Warn("Kakao API request returned non-200 status", "url", url, "status", resp.Status)
		return fmt.Errorf("API request failed with status: %s", resp.Status)
	}

	logger.Debug("Kakao API request successful", "url", url, "status", resp.Status)
	return json.NewDecoder(resp.Body).Decode(target)
}

func fetchMapDocuments(ctx context.Context, query string) ([]MapDocument, error) {
	var response KakaoLocalSearchResponse
	url := fmt.Sprintf("%s?query=%s", kakaoMapURL, query)
	err := makeKakaoRequest(ctx, url, &response)
	if err != nil {
		return nil, err
	}
	logger.Info("Fetched map documents", "query", query, "count", len(response.Documents))
	return response.Documents, nil
}

func fetchWebDocuments(ctx context.Context, query string) ([]WebDocument, error) {
	var response DaumWebSearchResponse
	url := fmt.Sprintf("%s/web?query=%s&page=1&size=5", daumSearchURL, query)
	err := makeKakaoRequest(ctx, url, &response)
	if err != nil {
		return nil, err
	}
	logger.Debug("Fetched web documents", "query", query, "count", len(response.Documents))
	return response.Documents, nil
}

func fetchImageDocument(ctx context.Context, query string) (ImageDocument, error) {
	var response DaumImageSearchResponse
	url := fmt.Sprintf("%s/image?query=%s&page=1&size=1", daumSearchURL, query)
	err := makeKakaoRequest(ctx, url, &response)
	if err != nil {
		return ImageDocument{}, err
	}
	if len(response.Documents) > 0 {
		logger.Debug("Fetched image document", "query", query, "image_url", response.Documents[0].ImageURL)
		return response.Documents[0], nil
	}
	logger.Debug("No image document found", "query", query)
	return ImageDocument{}, nil
}

// --- MCP 도구 콜백 함수 ---

func searchTool(ctx context.Context, req *mcp.CallToolRequest, options SearchSchema) (*mcp.CallToolResult, any, error) {
	logger.Info("searchTool called", "query", options.Query)
	// 1. GetSession()이 반환하는 mcp.Session 인터페이스를 *mcp.ServerSession 타입으로 단언합니다.
	serverSession, ok := req.GetSession().(*mcp.ServerSession)
	if !ok {
		return nil, nil, fmt.Errorf("internal error: could not cast session to *mcp.ServerSession")
	}

	if kakaoAPIKey == "" {
		logger.Error("Tool Execution Failed: KAKAO_API_KEY is not configured.")
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: "Tool Execution Failed: KAKAO_API_KEY is not configured."}},
		}, nil, nil
	}
	if options.Query == "" {
		logger.Warn("Query is empty")
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: "Query is empty"}},
		}, nil, nil
	}

	// 2. 타입 단언으로 얻은 serverSession 변수를 사용하여 NotifyProgress를 호출합니다.
	logger.Info("Notifying progress with system prompt")
	_ = serverSession.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
		ProgressToken: req.Params.GetProgressToken(),
		Message:       systemPrompt,
	})

	mapDocuments, err := fetchMapDocuments(ctx, options.Query)
	if err != nil {
		logger.Error("Failed to fetch map documents", "query", options.Query, "error", err)
		return nil, nil, fmt.Errorf("failed to fetch map documents: %w", err)
	}

	for _, doc := range mapDocuments {
		logger.Debug("Processing map document", "place_name", doc.PlaceName)
		var wg sync.WaitGroup
		var webDocs []WebDocument
		var imgDoc ImageDocument
		var webErr, imgErr error

		wg.Add(2)
		go func(d MapDocument) {
			defer wg.Done()
			webDocs, webErr = fetchWebDocuments(ctx, d.PlaceName)
		}(doc)
		go func(d MapDocument) {
			defer wg.Done()
			imgDoc, imgErr = fetchImageDocument(ctx, d.PlaceName)
		}(doc)
		wg.Wait()

		if webErr != nil || imgErr != nil {
			logger.Error("Error fetching details for place", "place_name", doc.PlaceName, "web_error", webErr, "image_error", imgErr)
			continue
		}

		result := KakaoLocalSearchResult{
			PlaceName:    doc.PlaceName,
			AddressName:  doc.AddressName,
			CategoryName: doc.CategoryName,
			PlaceURL:     doc.PlaceURL,
			Phone:        doc.Phone,
			Comments:     webDocs,
			ImageURL:     imgDoc.ImageURL,
		}

		resultBytes, err := json.Marshal(result)
		if err != nil {
			logger.Error("Failed to marshal result", "place_name", doc.PlaceName, "error", err)
			continue
		}

		// 3. 여기서도 serverSession 변수를 사용합니다.
		logger.Debug("Notifying progress with search result", "place_name", doc.PlaceName)
		_ = serverSession.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
			ProgressToken: req.Params.GetProgressToken(),
			// Value 필드는 MCP 프로토콜의 ProgressNotificationParams에 없으므로 Message 필드를 사용합니다.
			Message: string(resultBytes),
		})
	}

	logger.Info("Search complete. All results have been streamed.")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "Search complete. All results have been streamed."}},
	}, nil, nil
}

// --- HTTP 로깅 미들웨어 ---
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start),
			"remote_addr", r.RemoteAddr,
		)
	})
}

// --- 서버 실행 ---

func main() {
	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	kakaoAPIKey = os.Getenv("KAKAO_API_KEY")
	if kakaoAPIKey == "" {
		logger.Error("FATAL: KAKAO_API_KEY environment variable is not set.")
		os.Exit(1)
	}

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-server-kakao-map-go",
		Version: "0.0.1",
	}, nil)

	mcp.AddTool(mcpServer, &mcp.Tool{
		Name:        "kakao_map_place_recommender",
		Description: "Recommends relevant places in South Korea based on user queries.",
	}, searchTool)

	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return mcpServer
	}, nil)

	mux := http.NewServeMux()
	mux.Handle("/", loggingMiddleware(handler))

	port := "8080"
	logger.Info("MCP server with SSE is running", "url", "http://localhost:"+port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		logger.Error("Failed to start HTTP server", "error", err)
		os.Exit(1)
	}
}
