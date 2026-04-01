package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// registerUsageRoutes binds usage statistics endpoints to the ServeMux.
func (h *Handler) registerUsageRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/usage", h.handleGetUsage)
}

// UsageStats represents aggregated usage statistics for a model.
type UsageStats struct {
	ModelName     string  `json:"model_name"`
	Model         string  `json:"model"`
	MessageCount  int     `json:"message_count"`
	InputTokens   int     `json:"input_tokens"`
	OutputTokens  int     `json:"output_tokens"`
	TotalTokens   int     `json:"total_tokens"`
	EstimatedCost float64 `json:"estimated_cost"`
	Currency      string  `json:"currency"`
	SessionCount  int     `json:"session_count"`
}

// UsageResponse is the response for GET /api/usage.
type UsageResponse struct {
	Stats              []UsageStats `json:"stats"`
	TotalInputTokens   int          `json:"total_input_tokens"`
	TotalOutputTokens  int          `json:"total_output_tokens"`
	TotalTokens        int          `json:"total_tokens"`
	TotalMessageCount  int          `json:"total_message_count"`
	TotalEstimatedCost float64      `json:"total_estimated_cost"`
	Currency           string       `json:"currency"`
	DateRange          string       `json:"date_range"`
}

// modelPricing holds pricing information for a model.
type modelPricing struct {
	InputPricePerMTok  float64 // USD per 1M input tokens
	OutputPricePerMTok float64 // USD per 1M output tokens
}

// modelPricingDB contains known model pricing information.
// Prices are approximate and may vary. Update as needed.
var modelPricingDB = map[string]modelPricing{
	// OpenAI models
	"gpt-4o":        {InputPricePerMTok: 2.50, OutputPricePerMTok: 10.00},
	"gpt-4o-mini":   {InputPricePerMTok: 0.15, OutputPricePerMTok: 0.60},
	"gpt-4":         {InputPricePerMTok: 30.00, OutputPricePerMTok: 60.00},
	"gpt-4-turbo":   {InputPricePerMTok: 10.00, OutputPricePerMTok: 30.00},
	"gpt-3.5-turbo": {InputPricePerMTok: 0.50, OutputPricePerMTok: 1.50},
	"o1":            {InputPricePerMTok: 15.00, OutputPricePerMTok: 60.00},
	"o1-mini":       {InputPricePerMTok: 1.10, OutputPricePerMTok: 4.40},
	"o3-mini":       {InputPricePerMTok: 1.10, OutputPricePerMTok: 4.40},

	// Anthropic models
	"claude-sonnet-4-20250514":          {InputPricePerMTok: 3.00, OutputPricePerMTok: 15.00},
	"claude-sonnet-4-20250514-thinking": {InputPricePerMTok: 3.00, OutputPricePerMTok: 15.00},
	"claude-3-5-sonnet-20241022":        {InputPricePerMTok: 3.00, OutputPricePerMTok: 15.00},
	"claude-3-5-haiku-20241022":         {InputPricePerMTok: 0.80, OutputPricePerMTok: 4.00},
	"claude-3-opus-20240229":            {InputPricePerMTok: 15.00, OutputPricePerMTok: 75.00},
	"claude-sonnet-4-5-20250929":        {InputPricePerMTok: 3.00, OutputPricePerMTok: 15.00},

	// Google Gemini models
	"gemini-2.0-flash":      {InputPricePerMTok: 0.10, OutputPricePerMTok: 0.40},
	"gemini-2.0-flash-lite": {InputPricePerMTok: 0.075, OutputPricePerMTok: 0.30},
	"gemini-1.5-pro":        {InputPricePerMTok: 1.25, OutputPricePerMTok: 5.00},
	"gemini-1.5-flash":      {InputPricePerMTok: 0.075, OutputPricePerMTok: 0.30},
	"gemini-2.5-pro":        {InputPricePerMTok: 1.25, OutputPricePerMTok: 10.00},

	// Groq models
	"llama-3.3-70b-versatile": {InputPricePerMTok: 0.59, OutputPricePerMTok: 0.79},
	"llama-3.1-8b-instant":    {InputPricePerMTok: 0.05, OutputPricePerMTok: 0.08},
	"mixtral-8x7b-32768":      {InputPricePerMTok: 0.24, OutputPricePerMTok: 0.24},
	"gemma2-9b-it":            {InputPricePerMTok: 0.20, OutputPricePerMTok: 0.20},

	// DeepSeek models
	"deepseek-chat":     {InputPricePerMTok: 0.14, OutputPricePerMTok: 0.28},
	"deepseek-reasoner": {InputPricePerMTok: 0.14, OutputPricePerMTok: 1.10},

	// Qwen models (common on OpenRouter)
	"qwen/qwen-plus":            {InputPricePerMTok: 0.40, OutputPricePerMTok: 1.20},
	"qwen/qwen-turbo":           {InputPricePerMTok: 0.05, OutputPricePerMTok: 0.20},
	"qwen/qwen-max":             {InputPricePerMTok: 1.60, OutputPricePerMTok: 6.40},
	"qwen/qwen2.5-72b-instruct": {InputPricePerMTok: 0.40, OutputPricePerMTok: 1.20},
	"qwen/qwen3-235b-a22b":      {InputPricePerMTok: 0.40, OutputPricePerMTok: 1.20},
	"qwen/qwen3-30b-a3b":        {InputPricePerMTok: 0.10, OutputPricePerMTok: 0.30},
	"qwen/qwen3-32b":            {InputPricePerMTok: 0.10, OutputPricePerMTok: 0.30},
	"qwen/qwen3-8b":             {InputPricePerMTok: 0.05, OutputPricePerMTok: 0.15},
	"qwen/qwen3-14b":            {InputPricePerMTok: 0.08, OutputPricePerMTok: 0.24},
	"qwen/qwen3-coder":          {InputPricePerMTok: 0.10, OutputPricePerMTok: 0.30},

	// Meta Llama models (common on OpenRouter)
	"meta-llama/llama-3.3-70b-instruct":  {InputPricePerMTok: 0.12, OutputPricePerMTok: 0.30},
	"meta-llama/llama-3.1-405b-instruct": {InputPricePerMTok: 0.80, OutputPricePerMTok: 0.80},
	"meta-llama/llama-3.1-70b-instruct":  {InputPricePerMTok: 0.12, OutputPricePerMTok: 0.30},
	"meta-llama/llama-3.1-8b-instruct":   {InputPricePerMTok: 0.02, OutputPricePerMTok: 0.05},
	"meta-llama/llama-3-70b-instruct":    {InputPricePerMTok: 0.12, OutputPricePerMTok: 0.30},
	"meta-llama/llama-3-8b-instruct":     {InputPricePerMTok: 0.02, OutputPricePerMTok: 0.05},
	"meta-llama/llama-4-maverick":        {InputPricePerMTok: 0.15, OutputPricePerMTok: 0.60},
	"meta-llama/llama-4-scout":           {InputPricePerMTok: 0.05, OutputPricePerMTok: 0.15},

	// Mistral models (common on OpenRouter)
	"mistralai/mistral-large":          {InputPricePerMTok: 1.00, OutputPricePerMTok: 3.00},
	"mistralai/mistral-medium":         {InputPricePerMTok: 0.40, OutputPricePerMTok: 2.00},
	"mistralai/mistral-small":          {InputPricePerMTok: 0.10, OutputPricePerMTok: 0.30},
	"mistralai/mistral-7b-instruct":    {InputPricePerMTok: 0.02, OutputPricePerMTok: 0.05},
	"mistralai/mixtral-8x7b-instruct":  {InputPricePerMTok: 0.05, OutputPricePerMTok: 0.05},
	"mistralai/mixtral-8x22b-instruct": {InputPricePerMTok: 0.30, OutputPricePerMTok: 0.30},
	"mistralai/mistral-nemo":           {InputPricePerMTok: 0.03, OutputPricePerMTok: 0.09},
	"mistralai/codestral-2501":         {InputPricePerMTok: 0.15, OutputPricePerMTok: 0.45},

	// Free models (OpenRouter free tier, etc.)
	"qwen/qwen3.6-plus-preview:free": {InputPricePerMTok: 0, OutputPricePerMTok: 0},
}

// getModelPricing returns pricing for a model, trying exact match first,
// then prefix match for model families.
func getModelPricing(modelName string) modelPricing {
	// Try exact match first
	if pricing, ok := modelPricingDB[modelName]; ok {
		return pricing
	}

	// Try prefix match for model families
	lowerName := strings.ToLower(modelName)
	for pattern, pricing := range modelPricingDB {
		if strings.HasPrefix(lowerName, strings.ToLower(pattern)) {
			return pricing
		}
	}

	// Check for common model patterns
	switch {
	case strings.Contains(lowerName, "gpt-4o-mini"):
		return modelPricingDB["gpt-4o-mini"]
	case strings.Contains(lowerName, "gpt-4o"):
		return modelPricingDB["gpt-4o"]
	case strings.Contains(lowerName, "gpt-4-turbo"):
		return modelPricingDB["gpt-4-turbo"]
	case strings.Contains(lowerName, "gpt-4"):
		return modelPricingDB["gpt-4"]
	case strings.Contains(lowerName, "gpt-3.5"):
		return modelPricingDB["gpt-3.5-turbo"]
	case strings.Contains(lowerName, "claude-sonnet-4-5"):
		return modelPricingDB["claude-sonnet-4-5-20250929"]
	case strings.Contains(lowerName, "claude-sonnet-4"):
		return modelPricingDB["claude-sonnet-4-20250514"]
	case strings.Contains(lowerName, "claude-3-5-sonnet"):
		return modelPricingDB["claude-3-5-sonnet-20241022"]
	case strings.Contains(lowerName, "claude-3-5-haiku"):
		return modelPricingDB["claude-3-5-haiku-20241022"]
	case strings.Contains(lowerName, "claude-3-opus"):
		return modelPricingDB["claude-3-opus-20240229"]
	case strings.Contains(lowerName, "gemini-2.5"):
		return modelPricingDB["gemini-2.5-pro"]
	case strings.Contains(lowerName, "gemini-2.0-flash-lite"):
		return modelPricingDB["gemini-2.0-flash-lite"]
	case strings.Contains(lowerName, "gemini-2.0-flash"):
		return modelPricingDB["gemini-2.0-flash"]
	case strings.Contains(lowerName, "gemini-1.5-pro"):
		return modelPricingDB["gemini-1.5-pro"]
	case strings.Contains(lowerName, "gemini-1.5-flash"):
		return modelPricingDB["gemini-1.5-flash"]
	case strings.Contains(lowerName, "deepseek-reasoner"):
		return modelPricingDB["deepseek-reasoner"]
	case strings.Contains(lowerName, "deepseek-chat"):
		return modelPricingDB["deepseek-chat"]
	case strings.Contains(lowerName, "llama-3.3-70b"):
		return modelPricingDB["llama-3.3-70b-versatile"]
	case strings.Contains(lowerName, "llama-3.1-8b"):
		return modelPricingDB["llama-3.1-8b-instant"]
	case strings.Contains(lowerName, "free"):
		return modelPricing{InputPricePerMTok: 0, OutputPricePerMTok: 0}
	}

	// Default: unknown model, no pricing
	return modelPricing{}
}

// calculateCost computes the estimated cost based on token usage and model pricing.
func calculateCost(inputTokens, outputTokens int, pricing modelPricing) float64 {
	inputCost := float64(inputTokens) / 1_000_000 * pricing.InputPricePerMTok
	outputCost := float64(outputTokens) / 1_000_000 * pricing.OutputPricePerMTok
	return inputCost + outputCost
}

// usageDateFilter holds the date range for filtering usage data.
type usageDateFilter struct {
	StartDate time.Time
	EndDate   time.Time
}

// parseUsageDateFilter parses date filter parameters from the request.
// Defaults to today's date if no parameters provided.
func parseUsageDateFilter(r *http.Request) usageDateFilter {
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endDate := startDate.Add(24 * time.Hour)

	if startStr := r.URL.Query().Get("start_date"); startStr != "" {
		if t, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = t
			if endStr := r.URL.Query().Get("end_date"); endStr != "" {
				if endT, err := time.Parse("2006-01-02", endStr); err == nil {
					endDate = endT.Add(24 * time.Hour)
				} else {
					endDate = startDate.Add(24 * time.Hour)
				}
			} else {
				endDate = startDate.Add(24 * time.Hour)
			}
		}
	}

	return usageDateFilter{
		StartDate: startDate,
		EndDate:   endDate,
	}
}

// messageWithUsage is a custom struct to parse usage info from message content.
// The agent loop stores usage info in the message content as JSON fields.
type messageWithUsage struct {
	Role             string               `json:"role"`
	Content          string               `json:"content"`
	Media            []string             `json:"media,omitempty"`
	ReasoningContent string               `json:"reasoning_content,omitempty"`
	ToolCalls        []providers.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string               `json:"tool_call_id,omitempty"`
	// Usage fields stored by agent loop
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
	// Model info
	Model string `json:"model,omitempty"`
}

// handleGetUsage returns usage statistics aggregated by model.
//
//	GET /api/usage?start_date=2024-01-01&end_date=2024-01-31
func (h *Handler) handleGetUsage(w http.ResponseWriter, r *http.Request) {
	dateFilter := parseUsageDateFilter(r)

	dir, err := h.sessionsDir()
	if err != nil {
		http.Error(w, "failed to resolve sessions directory", http.StatusInternalServerError)
		return
	}

	// Load config to map model names to model identifiers
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	// Build model name to model identifier mapping
	modelNameToModel := make(map[string]string)
	for _, m := range cfg.ModelList {
		modelNameToModel[m.ModelName] = m.Model
	}

	// Read all session files and aggregate usage data
	entries, err := os.ReadDir(dir)
	if err != nil {
		// Directory doesn't exist yet = no usage data
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(UsageResponse{
			Stats:     []UsageStats{},
			Currency:  "USD",
			DateRange: fmt.Sprintf("%s to %s", dateFilter.StartDate.Format("2006-01-02"), dateFilter.EndDate.Add(-24*time.Hour).Format("2006-01-02")),
		})
		return
	}

	// Track per-model stats and session counts
	type modelStats struct {
		MessageCount int
		InputTokens  int
		OutputTokens int
		TotalTokens  int
		SessionKeys  map[string]struct{}
	}
	statsByModel := make(map[string]*modelStats)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".jsonl") {
			continue
		}

		// Skip meta files
		if strings.HasSuffix(name, ".meta.json") {
			continue
		}

		baseName := strings.TrimSuffix(name, ".jsonl")
		sess, err := h.readSessionByBaseName(dir, baseName)
		if err != nil {
			continue
		}

		// Check if session falls within the date filter
		// Use Created or Updated, whichever is available
		sessionTime := sess.Updated
		if sessionTime.IsZero() {
			sessionTime = sess.Created
		}
		if !sessionTime.IsZero() {
			if sessionTime.Before(dateFilter.StartDate) || sessionTime.After(dateFilter.EndDate) {
				continue
			}
		}

		// Get default model name from config
		defaultModelName := cfg.Agents.Defaults.GetModelName()
		if defaultModelName == "" {
			// Try to get from first model in list
			if len(cfg.ModelList) > 0 {
				defaultModelName = cfg.ModelList[0].ModelName
			}
		}

		// Process messages to extract usage info
		for _, msg := range sess.Messages {
			if msg.Role != "assistant" {
				continue
			}

			// Try to parse message as messageWithUsage to extract token info
			var msgWithUsage messageWithUsage
			msgData, _ := json.Marshal(msg)
			if err := json.Unmarshal(msgData, &msgWithUsage); err != nil {
				continue
			}

			// Determine model name
			modelName := msgWithUsage.Model
			if modelName == "" {
				modelName = defaultModelName
			}
			if modelName == "" {
				// Use "unknown" as fallback model name
				modelName = "unknown"
			}

			if _, exists := statsByModel[modelName]; !exists {
				statsByModel[modelName] = &modelStats{
					SessionKeys: make(map[string]struct{}),
				}
			}

			ms := statsByModel[modelName]
			ms.MessageCount++
			ms.InputTokens += msgWithUsage.PromptTokens
			ms.OutputTokens += msgWithUsage.CompletionTokens
			ms.TotalTokens += msgWithUsage.TotalTokens
			ms.SessionKeys[sess.Key] = struct{}{}
		}
	}

	// Build response
	stats := make([]UsageStats, 0, len(statsByModel))
	var totalInputTokens, totalOutputTokens, totalTokens, totalMessageCount int
	var totalEstimatedCost float64

	for modelName, ms := range statsByModel {
		modelIdentifier := modelNameToModel[modelName]
		if modelIdentifier == "" {
			modelIdentifier = modelName
		}

		pricing := getModelPricing(modelIdentifier)
		if pricing.InputPricePerMTok == 0 && pricing.OutputPricePerMTok == 0 {
			// Try with model name as well
			pricing = getModelPricing(modelName)
		}

		estimatedCost := calculateCost(ms.InputTokens, ms.OutputTokens, pricing)

		stat := UsageStats{
			ModelName:     modelName,
			Model:         modelIdentifier,
			MessageCount:  ms.MessageCount,
			InputTokens:   ms.InputTokens,
			OutputTokens:  ms.OutputTokens,
			TotalTokens:   ms.TotalTokens,
			EstimatedCost: estimatedCost,
			Currency:      "USD",
			SessionCount:  len(ms.SessionKeys),
		}
		stats = append(stats, stat)

		totalInputTokens += ms.InputTokens
		totalOutputTokens += ms.OutputTokens
		totalTokens += ms.TotalTokens
		totalMessageCount += ms.MessageCount
		totalEstimatedCost += estimatedCost
	}

	// Sort by total tokens descending
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].TotalTokens > stats[j].TotalTokens
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(UsageResponse{
		Stats:              stats,
		TotalInputTokens:   totalInputTokens,
		TotalOutputTokens:  totalOutputTokens,
		TotalTokens:        totalTokens,
		TotalMessageCount:  totalMessageCount,
		TotalEstimatedCost: totalEstimatedCost,
		Currency:           "USD",
		DateRange:          fmt.Sprintf("%s to %s", dateFilter.StartDate.Format("2006-01-02"), dateFilter.EndDate.Add(-24*time.Hour).Format("2006-01-02")),
	})
}

// readSessionByBaseName reads a session file by its base name (sanitized key without extension).
func (h *Handler) readSessionByBaseName(dir, baseName string) (sessionFile, error) {
	jsonlPath := filepath.Join(dir, baseName+".jsonl")
	metaPath := filepath.Join(dir, baseName+".meta.json")

	// Reconstruct session key from base name
	sessionKey := strings.ReplaceAll(baseName, "_", ":")

	meta, err := h.readSessionMeta(metaPath, sessionKey)
	if err != nil {
		return sessionFile{}, err
	}

	messages, err := h.readSessionMessages(jsonlPath, meta.Skip)
	if err != nil {
		return sessionFile{}, err
	}

	updated := meta.UpdatedAt
	created := meta.CreatedAt
	if created.IsZero() || updated.IsZero() {
		if info, statErr := os.Stat(jsonlPath); statErr == nil {
			if created.IsZero() {
				created = info.ModTime()
			}
			if updated.IsZero() {
				updated = info.ModTime()
			}
		}
	}

	return sessionFile{
		Key:      meta.Key,
		Messages: messages,
		Summary:  meta.Summary,
		Created:  created,
		Updated:  updated,
	}, nil
}

// readSessionMessagesForUsage reads session messages and extracts usage information.
func (h *Handler) readSessionMessagesForUsage(path string, skip int) ([]providers.Message, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	msgs := make([]providers.Message, 0)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), maxSessionJSONLLineSize)

	seen := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		seen++
		if seen <= skip {
			continue
		}

		var msg providers.Message
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		msgs = append(msgs, msg)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return msgs, nil
}
