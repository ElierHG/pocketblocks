package apis

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/pedrozadotdev/pocketblocks/server/daos"
	"github.com/pocketbase/pocketbase"
)

const (
	codexClientID             = "app_EMoamEEZ73f0CkXaXp7hrann"
	codexDeviceCodeEndpoint   = "https://auth0.openai.com/oauth/device/code"
	codexTokenEndpoint        = "https://auth0.openai.com/oauth/token"
	codexDeviceVerificationUI = "https://auth.openai.com/codex/device"
	codexRefreshTokenEndpoint = "https://auth0.openai.com/oauth/token"
)

type aiApi struct {
	app *pocketbase.PocketBase
	dao *daos.Dao
	ob  *openblocksApi
}

func BindAiApi(app *pocketbase.PocketBase, dao *daos.Dao, ob *openblocksApi, e *echo.Echo) {
	api := &aiApi{app: app, dao: dao, ob: ob}

	e.GET("/api/ai/config", api.getConfig)
	e.PUT("/api/ai/config", api.setConfig)
	e.POST("/api/ai/chat", api.chat)
	e.POST("/api/ai/auth/save-tokens", api.saveTokens)
	e.POST("/api/ai/auth/codex-import", api.importCodexAuth)
}

// --- Codex auth.json structure ---

type codexAuthJSON struct {
	AuthMode    string     `json:"auth_mode"`
	OpenAIKey   string     `json:"openai_api_key"`
	Tokens      *tokenData `json:"tokens"`
	LastRefresh string     `json:"last_refresh"`
}

type tokenData struct {
	IDToken      interface{} `json:"id_token"`
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	AccountID    string      `json:"account_id"`
}

type storedAuth struct {
	AuthMethod   string `json:"auth_method"` // "api_key", "codex_chatgpt"
	APIKey       string `json:"api_key"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	AccountID    string `json:"account_id"`
}

// --- Config endpoint ---

func (api *aiApi) getConfig(c echo.Context) error {
	if !api.ob.isLoggedIn(c) {
		return errResp(c, 401, "Unauthorized")
	}

	auth := api.getStoredAuth()
	codexAvailable := api.codexAuthFileExists()
	isAdm := api.ob.isAdmin(c)

	model := "gpt-4o"
	if auth.AuthMethod == "codex_chatgpt" {
		model = "gpt-5-codex-mini"
	}
	return okResp(c, map[string]interface{}{
		"hasApiKey":      auth.APIKey != "",
		"hasCodexAuth":   auth.AccessToken != "",
		"authMethod":     auth.AuthMethod,
		"codexAvailable": codexAvailable,
		"isAdmin":        isAdm,
		"model":          model,
	})
}

func (api *aiApi) setConfig(c echo.Context) error {
	if !api.ob.isAdmin(c) {
		return errResp(c, 401, "Unauthorized")
	}

	var body struct {
		ApiKey string `json:"apiKey"`
		Clear  bool   `json:"clear"`
	}
	if err := c.Bind(&body); err != nil {
		return errResp(c, 400, "Invalid request")
	}

	if body.Clear {
		if err := api.saveAuth(storedAuth{}); err != nil {
			return errResp(c, 500, "Failed to clear auth")
		}
		return okResp(c, map[string]interface{}{"success": true})
	}

	if body.ApiKey != "" {
		auth := storedAuth{
			AuthMethod: "api_key",
			APIKey:     body.ApiKey,
		}
		if err := api.saveAuth(auth); err != nil {
			return errResp(c, 500, "Failed to store API key")
		}
	}

	return okResp(c, map[string]interface{}{"success": true})
}

// --- Save tokens from frontend device code flow ---

func (api *aiApi) saveTokens(c echo.Context) error {
	if !api.ob.isAdmin(c) {
		return errResp(c, 401, "Unauthorized")
	}

	var body struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	if err := c.Bind(&body); err != nil {
		return errResp(c, 400, "Invalid request")
	}

	if body.AccessToken == "" {
		return errResp(c, 400, "Access token is required")
	}

	auth := storedAuth{
		AuthMethod:   "codex_chatgpt",
		AccessToken:  body.AccessToken,
		RefreshToken: body.RefreshToken,
		AccountID:    extractAccountIDFromJWT(body.AccessToken),
	}
	if err := api.saveAuth(auth); err != nil {
		return errResp(c, 500, "Failed to store tokens")
	}

	return okResp(c, map[string]interface{}{"success": true})
}

// --- Import from Codex CLI ---

func (api *aiApi) importCodexAuth(c echo.Context) error {
	if !api.ob.isAdmin(c) {
		return errResp(c, 401, "Unauthorized")
	}

	codexAuth, err := api.readCodexAuthFile()
	if err != nil {
		return errResp(c, 400, "Could not read Codex CLI auth file: "+err.Error())
	}

	if codexAuth.OpenAIKey != "" {
		auth := storedAuth{
			AuthMethod: "api_key",
			APIKey:     codexAuth.OpenAIKey,
		}
		if err := api.saveAuth(auth); err != nil {
			return errResp(c, 500, "Failed to store credentials")
		}
		return okResp(c, map[string]interface{}{"method": "api_key"})
	}

	if codexAuth.Tokens != nil && codexAuth.Tokens.AccessToken != "" {
		accountID := codexAuth.Tokens.AccountID
		if accountID == "" {
			accountID = extractAccountIDFromJWT(codexAuth.Tokens.AccessToken)
		}
		auth := storedAuth{
			AuthMethod:   "codex_chatgpt",
			AccessToken:  codexAuth.Tokens.AccessToken,
			RefreshToken: codexAuth.Tokens.RefreshToken,
			AccountID:    accountID,
		}
		if err := api.saveAuth(auth); err != nil {
			return errResp(c, 500, "Failed to store credentials")
		}
		return okResp(c, map[string]interface{}{"method": "codex_chatgpt"})
	}

	return errResp(c, 400, "No valid credentials found in Codex CLI auth file")
}

// --- Storage helpers ---

func (api *aiApi) getStoredAuth() storedAuth {
	param, err := api.dao.FindParamByKey("pbl_ai_auth")
	if err != nil {
		old := api.getStoredAPIKeyLegacy()
		if old != "" {
			return storedAuth{AuthMethod: "api_key", APIKey: old}
		}
		return storedAuth{}
	}
	var auth storedAuth
	if err := json.Unmarshal(param.Value, &auth); err != nil {
		return storedAuth{}
	}
	return auth
}

func (api *aiApi) saveAuth(auth storedAuth) error {
	return api.dao.SaveParam("pbl_ai_auth", auth)
}

func (api *aiApi) getStoredAPIKeyLegacy() string {
	param, err := api.dao.FindParamByKey("pbl_openai_key")
	if err != nil {
		return ""
	}
	var key string
	if err := json.Unmarshal(param.Value, &key); err != nil {
		return ""
	}
	return key
}

// extractAccountIDFromJWT parses the chatgpt_account_id from a JWT access
// token's payload. The Codex CLI sends this as the ChatGPT-Account-ID header
// which is required for ChatGPT-authenticated API access.
func extractAccountIDFromJWT(token string) string {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) < 2 {
		return ""
	}
	payload := parts[1]
	if m := len(payload) % 4; m != 0 {
		payload += strings.Repeat("=", 4-m)
	}
	data, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return ""
	}
	var claims struct {
		Auth struct {
			AccountID string `json:"chatgpt_account_id"`
		} `json:"https://api.openai.com/auth"`
	}
	if err := json.Unmarshal(data, &claims); err != nil {
		return ""
	}
	return claims.Auth.AccountID
}

func (api *aiApi) codexAuthFileExists() bool {
	path := api.codexAuthPath()
	_, err := os.Stat(path)
	return err == nil
}

func (api *aiApi) codexAuthPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	codexHome := os.Getenv("CODEX_HOME")
	if codexHome == "" {
		codexHome = filepath.Join(home, ".codex")
	}
	return filepath.Join(codexHome, "auth.json")
}

func (api *aiApi) readCodexAuthFile() (*codexAuthJSON, error) {
	path := api.codexAuthPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	var auth codexAuthJSON
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", path, err)
	}
	return &auth, nil
}

// --- Token refresh ---

func (api *aiApi) refreshAccessToken(refreshToken string) (string, string, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", codexClientID)
	data.Set("refresh_token", refreshToken)
	data.Set("scope", "openid profile email")

	resp, err := http.PostForm(codexRefreshTokenEndpoint, data)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Error        string `json:"error"`
	}
	json.Unmarshal(body, &tokenResp)

	if tokenResp.Error != "" {
		return "", "", fmt.Errorf("refresh failed: %s", tokenResp.Error)
	}

	newRefresh := tokenResp.RefreshToken
	if newRefresh == "" {
		newRefresh = refreshToken
	}
	return tokenResp.AccessToken, newRefresh, nil
}

// --- Resolve the bearer token for OpenAI API calls ---

func (api *aiApi) resolveOpenAIAuth() (string, error) {
	auth := api.getStoredAuth()

	if auth.AuthMethod == "api_key" && auth.APIKey != "" {
		return auth.APIKey, nil
	}

	if auth.AuthMethod == "codex_chatgpt" && auth.AccessToken != "" {
		return auth.AccessToken, nil
	}

	return "", fmt.Errorf("no AI authentication configured")
}

func (api *aiApi) handleAuthFailureAndRetry(reqFn func(token string) (*http.Response, error)) (*http.Response, error) {
	auth := api.getStoredAuth()

	token := auth.APIKey
	if auth.AuthMethod == "codex_chatgpt" {
		token = auth.AccessToken
	}
	if token == "" {
		return nil, fmt.Errorf("no AI authentication configured")
	}

	resp, err := reqFn(token)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 401 && auth.AuthMethod == "codex_chatgpt" && auth.RefreshToken != "" {
		resp.Body.Close()
		newAccess, newRefresh, err := api.refreshAccessToken(auth.RefreshToken)
		if err != nil {
			return nil, fmt.Errorf("token refresh failed: %w", err)
		}
		auth.AccessToken = newAccess
		auth.RefreshToken = newRefresh
		api.saveAuth(auth)
		return reqFn(newAccess)
	}

	return resp, nil
}

// --- Chat endpoint ---

const systemPrompt = `You are an AI assistant integrated into PocketBlocks, a low-code app builder. You help users build pages and dashboards by generating and modifying page DSL (Domain Specific Language) JSON.

## DSL Structure
The page DSL is a JSON object with these top-level keys:
- "ui" - Contains the page layout and components
- "queries" - JavaScript data queries
- "tempStates" - Temporary state variables
- "transformers" - Data transformers
- "settings" - App settings (theme, title, etc.)

## UI Structure
The "ui" key contains:
- "compType": "page" (always for root)
- "comp": contains child components keyed by unique names (e.g., "button1", "table1")
- "layout": position data for each component

Each component in "comp" has:
- "compType": the component type (see available types below)
- "comp": component-specific properties
- "name": display name

## Layout System
Components use a grid layout (24 columns wide). Each component has layout info:
- "i": component key (matches the key in comp)
- "x": horizontal position (0-23, column based)
- "y": vertical position (row based, each row ~8px)
- "w": width in columns (1-24)
- "h": height in rows
- "pos": 0 (default)

## Available Component Types
- "input" - Text input field. Props: value, label, placeholder
- "textArea" - Multi-line text. Props: value, label, placeholder
- "password" - Password input. Props: value, label, placeholder
- "numberInput" - Number input. Props: value, label, min, max, step
- "slider" - Slider control. Props: value, min, max, step
- "rangeSlider" - Range slider. Props: start, end, min, max
- "rating" - Star rating. Props: value, max
- "switch" - Toggle switch. Props: value, label
- "select" - Dropdown select. Props: value, options, label
- "multiSelect" - Multi-select. Props: value, options, label
- "cascader" - Cascading select. Props: value, options
- "checkbox" - Checkbox. Props: value, label
- "radio" - Radio buttons. Props: value, options, label
- "segmentedControl" - Segmented control. Props: value, options
- "date" - Date picker. Props: value, label
- "dateRange" - Date range picker. Props: start, end
- "time" - Time picker. Props: value, label
- "timeRange" - Time range picker. Props: start, end
- "file" - File upload. Props: value, label, accept
- "button" - Button. Props: text, type (primary/default/link), onClick events
- "link" - Link/anchor. Props: text, href
- "dropdown" - Dropdown button. Props: label, options
- "text" - Display text/markdown. Props: value (supports {{expressions}})
- "table" - Data table. Props: data, columns, pagination
- "image" - Image display. Props: src, alt
- "progress" - Progress bar. Props: value (0-100)
- "progressCircle" - Circular progress. Props: value (0-100)
- "divider" - Horizontal divider
- "qrCode" - QR code. Props: value
- "form" - Form container with submit
- "container" - Generic container for nesting
- "tabbedContainer" - Tabbed container. Props: tabs
- "modal" - Modal dialog
- "listView" - List/repeater. Props: data
- "chart" - ECharts chart. Props: epiption (echarts option JSON)
- "navigation" - Navigation menu. Props: items
- "iframe" - Embedded iframe. Props: url
- "jsonExplorer" - JSON viewer. Props: value
- "jsonEditor" - JSON editor. Props: value
- "tree" - Tree view. Props: value
- "treeSelect" - Tree select. Props: value
- "audio" - Audio player. Props: src
- "video" - Video player. Props: src
- "drawer" - Side drawer
- "carousel" - Image carousel. Props: images
- "toggleButton" - Toggle button. Props: value
- "signature" - Signature pad
- "scanner" - QR/barcode scanner

## Component Properties
String properties can contain JavaScript expressions wrapped in {{ }}:
- Static: "Hello World"
- Dynamic: "{{query1.data.length}} items"
- Expression: "{{currentUser.name}}"

## Event Handlers
Components can have event handlers. Common events:
- onClick, onChange, onSubmit, onSelect
Event handler format in DSL:
"events": [{"name": "click", "handler": {"compType": "executeComp", "comp": {"methodName": "someMethod"}}}]

## Queries
JavaScript queries fetch/process data:
"queries": {"query1": {"compType": "js", "comp": {"script": "return fetch('/api/data').then(r => r.json())"}}}

## Rules
1. ALWAYS return valid JSON for the complete DSL
2. Use unique component names (e.g., "text1", "button1", "table1")
3. Position components using the 24-column grid
4. Keep the layout clean and well-organized
5. Use meaningful default values for components
6. When modifying existing DSL, preserve components that shouldn't change

## Response Format
You MUST respond with ONLY a JSON object with two keys:
- "explanation": Brief text explaining what you did
- "dsl": The complete page DSL JSON object

Do NOT include markdown code fences, explanatory text outside the JSON, or anything else. Return ONLY the JSON object.`

func (api *aiApi) chat(c echo.Context) error {
	if !api.ob.isLoggedIn(c) {
		return errResp(c, 401, "Unauthorized")
	}

	var body struct {
		Message    string      `json:"message"`
		CurrentDSL interface{} `json:"currentDSL"`
	}
	if err := c.Bind(&body); err != nil {
		return errResp(c, 400, "Invalid request")
	}

	if body.Message == "" {
		return errResp(c, 400, "Message is required")
	}

	currentDSLJSON := "{}"
	if body.CurrentDSL != nil {
		b, _ := json.Marshal(body.CurrentDSL)
		currentDSLJSON = string(b)
	}

	userMessage := body.Message
	if currentDSLJSON != "{}" {
		userMessage = fmt.Sprintf("Current page DSL:\n```json\n%s\n```\n\nUser request: %s", currentDSLJSON, body.Message)
	}

	auth := api.getStoredAuth()
	if auth.AuthMethod == "codex_chatgpt" {
		return api.chatViaResponses(c, userMessage, auth.AccountID)
	}
	return api.chatViaCompletions(c, userMessage)
}

func (api *aiApi) chatViaCompletions(c echo.Context, userMessage string) error {
	openaiReq := map[string]interface{}{
		"model": "gpt-4o",
		"messages": []map[string]interface{}{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userMessage},
		},
		"temperature":     0.7,
		"max_tokens":      16000,
		"response_format": map[string]interface{}{"type": "json_object"},
	}

	reqBody, err := json.Marshal(openaiReq)
	if err != nil {
		return errResp(c, 500, "Failed to build AI request")
	}

	resp, err := api.handleAuthFailureAndRetry(func(token string) (*http.Response, error) {
		req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(reqBody))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		return (&http.Client{}).Do(req)
	})
	if err != nil {
		return errResp(c, 500, "AI request failed: "+err.Error())
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errResp(c, 500, "Failed to read AI response")
	}

	if resp.StatusCode != 200 {
		if strings.Contains(string(respBody), "model.request") {
			return errResp(c, 502, "Your OpenAI API key lacks the 'model.request' scope. Create a new key with full permissions at https://platform.openai.com/api-keys, or use 'Sign in with ChatGPT' instead.")
		}
		return errResp(c, 502, "AI service error: "+string(respBody))
	}

	var openaiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &openaiResp); err != nil {
		return errResp(c, 500, "Failed to parse AI response")
	}

	if len(openaiResp.Choices) == 0 {
		return errResp(c, 500, "AI returned no response")
	}

	return api.parseAndReturnAIContent(c, openaiResp.Choices[0].Message.Content)
}

func (api *aiApi) chatViaResponses(c echo.Context, userMessage string, accountID string) error {
	openaiReq := map[string]interface{}{
		"model":        "gpt-5-codex-mini",
		"instructions": systemPrompt,
		"input": []map[string]interface{}{
			{"role": "user", "content": userMessage + "\n\nRespond in JSON format."},
		},
		"text": map[string]interface{}{
			"format": map[string]interface{}{
				"type": "json_object",
			},
		},
		"stream": true,
		"store":  false,
	}

	reqBody, err := json.Marshal(openaiReq)
	if err != nil {
		return errResp(c, 500, "Failed to build AI request")
	}

	resp, err := api.handleAuthFailureAndRetry(func(token string) (*http.Response, error) {
		req, err := http.NewRequest("POST", "https://chatgpt.com/backend-api/codex/responses", bytes.NewReader(reqBody))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Authorization", "Bearer "+token)
		if accountID != "" {
			req.Header.Set("ChatGPT-Account-ID", accountID)
		}
		return (&http.Client{}).Do(req)
	})
	if err != nil {
		return errResp(c, 500, "AI request failed: "+err.Error())
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errResp(c, 500, "Failed to read AI response")
	}

	if resp.StatusCode != 200 {
		respStr := string(respBody)
		if strings.Contains(respStr, "api.responses.write") || strings.Contains(respStr, "model.request") || strings.Contains(respStr, "insufficient permissions") {
			return errResp(c, 502, "Your ChatGPT sign-in token doesn't have API access. Please use an API key instead: go to Settings > Enter API Key, using a key from https://platform.openai.com/api-keys")
		}
		return errResp(c, 502, "AI service error: "+respStr)
	}

	// Parse SSE stream to extract the output text
	content := extractTextFromSSE(string(respBody))
	if content == "" {
		return errResp(c, 500, "AI returned no response")
	}

	return api.parseAndReturnAIContent(c, content)
}

// extractTextFromSSE parses a Responses API SSE stream and concatenates all
// output_text delta fragments into the complete text.
func extractTextFromSSE(sseData string) string {
	var result strings.Builder
	for _, line := range strings.Split(sseData, "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var event struct {
			Type string `json:"type"`
			// For response.output_text.delta
			Delta string `json:"delta"`
			// For response.output_item.done with full content
			Item struct {
				Type    string `json:"type"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"item"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		switch event.Type {
		case "response.output_text.delta":
			result.WriteString(event.Delta)
		case "response.output_item.done":
			if event.Item.Type == "message" {
				for _, c := range event.Item.Content {
					if c.Type == "output_text" && c.Text != "" {
						return c.Text
					}
				}
			}
		}
	}
	return result.String()
}

func (api *aiApi) parseAndReturnAIContent(c echo.Context, content string) error {
	var aiResult map[string]interface{}
	if err := json.Unmarshal([]byte(content), &aiResult); err != nil {
		return okResp(c, map[string]interface{}{
			"explanation": content,
			"dsl":         nil,
			"raw":         content,
		})
	}
	return okResp(c, aiResult)
}
