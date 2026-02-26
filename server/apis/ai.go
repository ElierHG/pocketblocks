package apis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pedrozadotdev/pocketblocks/server/daos"
	"github.com/pocketbase/pocketbase"
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
}

// --- Config: store/retrieve OpenAI API key in PBL settings ---

func (api *aiApi) getConfig(c echo.Context) error {
	if !api.ob.isAdmin(c) {
		return errResp(c, 401, "Unauthorized")
	}

	settings := api.dao.GetPblSettings()
	s, _ := settings.Clone()

	hasKey := false
	if key := api.getStoredAPIKey(); key != "" {
		hasKey = true
	}

	return okResp(c, map[string]interface{}{
		"hasApiKey":  hasKey,
		"model":     getAIModel(s),
	})
}

func (api *aiApi) setConfig(c echo.Context) error {
	if !api.ob.isAdmin(c) {
		return errResp(c, 401, "Unauthorized")
	}

	var body struct {
		ApiKey string `json:"apiKey"`
		Model  string `json:"model"`
	}
	if err := c.Bind(&body); err != nil {
		return errResp(c, 400, "Invalid request")
	}

	if body.ApiKey != "" {
		if err := api.storeAPIKey(body.ApiKey); err != nil {
			return errResp(c, 500, "Failed to store API key")
		}
	}

	return okResp(c, map[string]interface{}{"success": true})
}

func (api *aiApi) getStoredAPIKey() string {
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

func (api *aiApi) storeAPIKey(key string) error {
	return api.dao.SaveParam("pbl_openai_key", key)
}

func getAIModel(s interface{}) string {
	return "gpt-4o"
}

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

	apiKey := api.getStoredAPIKey()
	if apiKey == "" {
		return errResp(c, 400, "OpenAI API key not configured. Please set it in Settings > AI.")
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

	openaiReq := map[string]interface{}{
		"model": "gpt-4o",
		"messages": []map[string]interface{}{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userMessage},
		},
		"temperature":  0.7,
		"max_tokens":   16000,
		"response_format": map[string]interface{}{"type": "json_object"},
	}

	reqBody, err := json.Marshal(openaiReq)
	if err != nil {
		return errResp(c, 500, "Failed to build AI request")
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return errResp(c, 500, "Failed to create AI request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errResp(c, 500, "Failed to call AI service: "+err.Error())
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errResp(c, 500, "Failed to read AI response")
	}

	if resp.StatusCode != 200 {
		return errResp(c, resp.StatusCode, "AI service error: "+string(respBody))
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

	content := openaiResp.Choices[0].Message.Content

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
