package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"lumina/internal/db"
	"lumina/internal/sys"
)

// Default consumer client ID (supports PKCE with loopback redirect URI http://127.0.0.1:8989/oauth/callback)
const (
	DefaultConsumerClientID = "718149887827-0g70r11e5h23jov2246g1a8k3715c0e1.apps.googleusercontent.com"
	LoopbackPort            = "8989"
	LoopbackRedirectURI     = "http://127.0.0.1:8989/oauth/callback"
	GooglePhotosScope       = "https://www.googleapis.com/auth/photoslibrary https://www.googleapis.com/auth/photoslibrary.sharing"
)

// GooglePhotosSyncManager orchestrates authentication, batch uploads, album routing, and deduplication.
type GooglePhotosSyncManager struct {
	mu            sync.RWMutex
	database      *db.Database
	clientID      string
	clientSecret  string
	accessToken   string
	refreshToken  string
	tokenExpiry   time.Time
	httpClient    *http.Client
	albumCache    map[string]string // AlbumName -> AlbumID
	serverRunning bool
	cancelServer  context.CancelFunc
}

// NewGooglePhotosSyncManager creates a sync manager.
func NewGooglePhotosSyncManager(database *db.Database) *GooglePhotosSyncManager {
	ctx := context.Background()
	savedRefreshToken := database.GetSetting(ctx, "gphotos_refresh_token", "")
	customClientID := database.GetSetting(ctx, "gphotos_client_id", DefaultConsumerClientID)
	customSecret := database.GetSetting(ctx, "gphotos_client_secret", "")

	return &GooglePhotosSyncManager{
		database:     database,
		clientID:     customClientID,
		clientSecret: customSecret,
		refreshToken: savedRefreshToken,
		httpClient:   &http.Client{Timeout: 60 * time.Second},
		albumCache:   make(map[string]string),
	}
}

// IsConnected returns whether Google Photos is authenticated.
func (g *GooglePhotosSyncManager) IsConnected() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.refreshToken != ""
}

// Start1ClickSignIn launches PKCE OAuth flow, opens system browser, and listens on 127.0.0.1:8989.
func (g *GooglePhotosSyncManager) Start1ClickSignIn(ctx context.Context, onAuthSuccess func()) (string, error) {
	verifier, challenge, err := GeneratePKCE()
	if err != nil {
		return "", err
	}

	stateBytes := make([]byte, 16)
	_, _ = randRead(stateBytes)
	state := base64URLEncode(stateBytes)

	authURL := fmt.Sprintf("https://accounts.google.com/o/oauth2/v2/auth?"+
		"client_id=%s&"+
		"redirect_uri=%s&"+
		"response_type=code&"+
		"scope=%s&"+
		"code_challenge=%s&"+
		"code_challenge_method=S256&"+
		"state=%s&"+
		"access_type=offline&"+
		"prompt=consent",
		url.QueryEscape(g.clientID),
		url.QueryEscape(LoopbackRedirectURI),
		url.QueryEscape(GooglePhotosScope),
		url.QueryEscape(challenge),
		url.QueryEscape(state),
	)

	// Start local HTTP listener for the OAuth callback
	listener, err := net.Listen("tcp", "127.0.0.1:"+LoopbackPort)
	if err != nil {
		return "", fmt.Errorf("failed to bind loopback port %s: %w", LoopbackPort, err)
	}

	server := &http.Server{}
	serverCtx, cancel := context.WithCancel(ctx)
	g.cancelServer = cancel
	server.BaseContext = func(net.Listener) context.Context { return serverCtx }

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Authentication failed: missing authorization code")
			return
		}

		// Exchange code for tokens
		err := g.exchangeCodeForTokens(code, verifier)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Token exchange failed: %v", err)
			return
		}

		// Show beautiful translucent success page to user
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
	<title>Lumina Photos — Connected</title>
	<style>
		body { background: #0f172a; color: #f8fafc; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0; }
		.card { background: rgba(30, 41, 59, 0.7); border: 1px solid rgba(255, 255, 255, 0.15); border-radius: 1.25rem; padding: 2.5rem; text-align: center; max-width: 420px; box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5); backdrop-filter: blur(16px); }
		h1 { color: #38bdf8; margin-bottom: 0.5rem; font-size: 1.5rem; }
		p { color: #94a3b8; font-size: 0.95rem; line-height: 1.5; }
		.pill { display: inline-block; margin-top: 1.25rem; padding: 0.5rem 1.25rem; background: #0284c7; color: white; border-radius: 9999px; font-weight: 500; font-size: 0.875rem; }
	</style>
</head>
<body>
	<div class="card">
		<h1>✨ Google Photos Connected!</h1>
		<p>Lumina Photos has been authorized successfully. You can close this browser tab and return to the application.</p>
		<div class="pill">Ready to Sync</div>
	</div>
</body>
</html>`)

		if onAuthSuccess != nil {
			go onAuthSuccess()
		}

		// Gracefully shut down loopback server
		go func() {
			time.Sleep(1 * time.Second)
			_ = server.Shutdown(context.Background())
		}()
	})

	server.Handler = mux

	go func() {
		_ = server.Serve(listener)
	}()

	// Open browser using OS protocol handler
	go OpenExternalURL(authURL)

	return authURL, nil
}

// OpenExternalURL opens a URL in the user's default browser on Windows.
// Guaranteed no visible console window (Rule 5).
func OpenExternalURL(urlStr string) {
	if runtime.GOOS == "windows" {
		_ = sys.SilentCommand("rundll32", "url.dll,FileProtocolHandler", urlStr).Start()
	} else {
		_ = sys.SilentCommand("xdg-open", urlStr).Start()
	}
}

func (g *GooglePhotosSyncManager) exchangeCodeForTokens(code string, verifier string) error {
	data := url.Values{
		"client_id":     {g.clientID},
		"client_secret": {g.clientSecret},
		"code":          {code},
		"code_verifier": {verifier},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {LoopbackRedirectURI},
	}

	resp, err := g.httpClient.PostForm("https://oauth2.googleapis.com/token", data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token exchange error (%d): %s", resp.StatusCode, string(b))
	}

	var tokenRes struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenRes); err != nil {
		return err
	}

	g.mu.Lock()
	g.accessToken = tokenRes.AccessToken
	if tokenRes.RefreshToken != "" {
		g.refreshToken = tokenRes.RefreshToken
		_ = g.database.SetSetting(context.Background(), "gphotos_refresh_token", tokenRes.RefreshToken)
	}
	g.tokenExpiry = time.Now().Add(time.Duration(tokenRes.ExpiresIn-60) * time.Second)
	g.mu.Unlock()

	return nil
}

// EnsureValidAccessToken refreshes access_token if expired.
func (g *GooglePhotosSyncManager) EnsureValidAccessToken(ctx context.Context) (string, error) {
	g.mu.RLock()
	tok := g.accessToken
	exp := g.tokenExpiry
	ref := g.refreshToken
	g.mu.RUnlock()

	if tok != "" && time.Now().Before(exp) {
		return tok, nil
	}

	if ref == "" {
		return "", fmt.Errorf("not authenticated to Google Photos")
	}

	data := url.Values{
		"client_id":     {g.clientID},
		"client_secret": {g.clientSecret},
		"refresh_token": {ref},
		"grant_type":    {"refresh_token"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("refresh token failed (%d): %s", resp.StatusCode, string(b))
	}

	var tokenRes struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenRes); err != nil {
		return "", err
	}

	g.mu.Lock()
	g.accessToken = tokenRes.AccessToken
	g.tokenExpiry = time.Now().Add(time.Duration(tokenRes.ExpiresIn-60) * time.Second)
	g.mu.Unlock()

	return tokenRes.AccessToken, nil
}

// UploadBatch uploads up to 50 photos in a single batch, preserving tags and descriptions.
func (g *GooglePhotosSyncManager) UploadBatch(ctx context.Context, photos []*db.PhotoRecord, onProgress func(syncedCount int, totalCount int)) (int, error) {
	token, err := g.EnsureValidAccessToken(ctx)
	if err != nil {
		return 0, err
	}

	synced := 0

	// Loop in chunks of 50 photos (Google Photos batchCreate limit is 50 items)
	const chunkSize = 50
	for i := 0; i < len(photos); i += chunkSize {
		end := i + chunkSize
		if end > len(photos) {
			end = len(photos)
		}
		chunk := photos[i:end]

		for _, p := range chunk {
			select {
			case <-ctx.Done():
				return synced, ctx.Err()
			default:
			}

			// Deduplicate: check if already in registry
			if g.database.IsSyncedToGPhotos(ctx, p.Sha256) {
				synced++
				if onProgress != nil {
					onProgress(synced, len(photos))
				}
				continue
			}

			// Determine source file path
			filePath := p.OrganizedPath
			if filePath == "" || !fileExists(filePath) {
				filePath = p.OriginalPath
			}
			if !fileExists(filePath) {
				continue
			}

			// 1. Upload byte stream to obtain uploadToken
			uploadToken, err := g.uploadPhotoBytes(ctx, token, filePath)
			if err != nil {
				continue
			}

			// 2. Prepare rich description embedding tags + OCR
			description := strings.TrimSpace(fmt.Sprintf("%s\nTags: %s\nText: %s", p.AiDescription, p.Tags, p.OcrText))

			// 3. Resolve or create album for canonical device
			albumID, _ := g.getOrCreateAlbum(ctx, token, p.CanonicalDevice)

			// 4. Batch create media item
			remoteID, err := g.createMediaItem(ctx, token, albumID, uploadToken, p.FileName, description)
			if err == nil {
				_ = g.database.MarkSyncedToGPhotos(ctx, p.Sha256, remoteID, p.CanonicalDevice)
				synced++
				if onProgress != nil {
					onProgress(synced, len(photos))
				}
			}
		}
	}

	return synced, nil
}

func (g *GooglePhotosSyncManager) uploadPhotoBytes(ctx context.Context, token string, filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(filePath))
	mime := "image/jpeg"
	if ext == ".png" {
		mime = "image/png"
	} else if ext == ".mp4" || ext == ".mov" {
		mime = "video/mp4"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://photoslibrary.googleapis.com/v1/uploads", f)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Goog-Upload-Content-Type", mime)
	req.Header.Set("X-Goog-Upload-Protocol", "raw")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload bytes failed (%d): %s", resp.StatusCode, string(b))
	}

	uploadToken, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(uploadToken), nil
}

func (g *GooglePhotosSyncManager) getOrCreateAlbum(ctx context.Context, token string, albumTitle string) (string, error) {
	if albumTitle == "" {
		albumTitle = "Lumina Photos"
	}

	g.mu.RLock()
	if id, exists := g.albumCache[albumTitle]; exists {
		g.mu.RUnlock()
		return id, nil
	}
	g.mu.RUnlock()

	reqBody := map[string]any{
		"album": map[string]string{
			"title": albumTitle,
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://photoslibrary.googleapis.com/v1/albums", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var albumRes struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&albumRes); err == nil && albumRes.ID != "" {
		g.mu.Lock()
		g.albumCache[albumTitle] = albumRes.ID
		g.mu.Unlock()
		return albumRes.ID, nil
	}

	return "", nil
}

func (g *GooglePhotosSyncManager) createMediaItem(ctx context.Context, token string, albumID string, uploadToken string, fileName string, description string) (string, error) {
	item := map[string]any{
		"description": description,
		"simpleMediaItem": map[string]string{
			"uploadToken": uploadToken,
			"fileName":    fileName,
		},
	}

	payload := map[string]any{
		"newMediaItems": []map[string]any{item},
	}
	if albumID != "" {
		payload["albumId"] = albumID
	}

	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://photoslibrary.googleapis.com/v1/mediaItems:batchCreate", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var batchRes struct {
		NewMediaItemResults []struct {
			MediaItem struct {
				ID string `json:"id"`
			} `json:"mediaItem"`
			Status struct {
				Message string `json:"message"`
			} `json:"status"`
		} `json:"newMediaItemResults"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&batchRes); err == nil {
		if len(batchRes.NewMediaItemResults) > 0 {
			return batchRes.NewMediaItemResults[0].MediaItem.ID, nil
		}
	}

	return "", fmt.Errorf("create media item failed")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func randRead(b []byte) (int, error) {
	return io.ReadFull(strings.NewReader(fmt.Sprintf("%d", time.Now().UnixNano())), b)
}

func base64URLEncode(b []byte) string {
	return strings.TrimRight(strings.ReplaceAll(strings.ReplaceAll(fmt.Sprintf("%x", b), "+", "-"), "/", "_"), "=")
}
