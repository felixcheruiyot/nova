package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const claudeModel = "claude-haiku-4-5-20251001"
const claudeAPI = "https://api.anthropic.com/v1/messages"

var aiClient = &http.Client{Timeout: 60 * time.Second}

// moduleAI returns the built-in `ai` module.
func moduleAI() *Value {
	return &Value{Type: TypeModule, ModuleVal: map[string]*Value{
		"prompt":    {Type: TypeBuiltin, BuiltinVal: aiPrompt},
		"summarize": {Type: TypeBuiltin, BuiltinVal: aiSummarize},
		"translate": {Type: TypeBuiltin, BuiltinVal: aiTranslate},
		"classify":  {Type: TypeBuiltin, BuiltinVal: aiClassify},
		"extract":   {Type: TypeBuiltin, BuiltinVal: aiExtract},
	}}
}

// aiCall sends a single-turn message to the Claude API and returns the text.
func aiCall(system, user string) (string, error) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return "", fmt.Errorf("ai: ANTHROPIC_API_KEY environment variable is not set")
	}

	body, _ := json.Marshal(map[string]any{
		"model":      claudeModel,
		"max_tokens": 1024,
		"system":     system,
		"messages":   []map[string]string{{"role": "user", "content": user}},
	})

	req, err := http.NewRequest("POST", claudeAPI, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("x-api-key", key)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := aiClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ai: request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("ai: failed to decode response: %w", err)
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("ai: API error %d: %s", resp.StatusCode, result.Error.Message)
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("ai: empty response from API")
	}
	return result.Content[0].Text, nil
}

// ai.prompt(text) — general-purpose completion.
func aiPrompt(args []*Value) (*Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("ai.prompt() expects 1 argument: text")
	}
	out, err := aiCall("You are a helpful assistant.", args[0].String())
	if err != nil {
		return nil, err
	}
	return StringVal(out), nil
}

// ai.summarize(text) — return a concise summary.
func aiSummarize(args []*Value) (*Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("ai.summarize() expects 1 argument: text")
	}
	out, err := aiCall(
		"Summarize the following text concisely. Return only the summary, no preamble.",
		args[0].String(),
	)
	if err != nil {
		return nil, err
	}
	return StringVal(out), nil
}

// ai.translate(text, language) — translate to the target language.
func aiTranslate(args []*Value) (*Value, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("ai.translate() expects 2 arguments: text, language")
	}
	lang := args[1].String()
	out, err := aiCall(
		fmt.Sprintf("Translate the following text to %s. Return only the translation, no preamble.", lang),
		args[0].String(),
	)
	if err != nil {
		return nil, err
	}
	return StringVal(out), nil
}

// ai.classify(text, labels) — classify text into one of the provided labels.
// labels may be a string "a,b,c" or a list value.
func aiClassify(args []*Value) (*Value, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("ai.classify() expects 2 arguments: text, labels")
	}
	var labelStr string
	if args[1].Type == TypeList {
		parts := make([]string, len(args[1].ListVal))
		for i, v := range args[1].ListVal {
			parts[i] = v.String()
		}
		labelStr = strings.Join(parts, ", ")
	} else {
		labelStr = args[1].String()
	}
	out, err := aiCall(
		fmt.Sprintf("Classify the following text into exactly one of these categories: %s.\nReturn only the category name, nothing else.", labelStr),
		args[0].String(),
	)
	if err != nil {
		return nil, err
	}
	return StringVal(strings.TrimSpace(out)), nil
}

// ai.extract(text, fields) — extract structured fields from text; returns a map.
func aiExtract(args []*Value) (*Value, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("ai.extract() expects 2 arguments: text, fields")
	}
	fields := args[1].String()
	prompt := fmt.Sprintf(
		"Extract the following fields from the text: %s.\nReturn a JSON object with those fields. Return only valid JSON, no explanation.",
		fields,
	)
	out, err := aiCall(prompt, args[0].String())
	if err != nil {
		return nil, err
	}
	// Parse the JSON response back to a Nova map
	var raw map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &raw); err != nil {
		// If it's not valid JSON, return as string
		return StringVal(out), nil
	}
	m := &Value{Type: TypeMap, MapVal: make(map[string]*Value)}
	for k, v := range raw {
		m.MapVal[k] = anyToValue(v)
		m.MapOrder = append(m.MapOrder, k)
	}
	return m, nil
}

func anyToValue(v any) *Value {
	switch t := v.(type) {
	case nil:
		return Null
	case bool:
		return BoolVal(t)
	case float64:
		return NumberVal(t)
	case string:
		return StringVal(t)
	case []any:
		list := make([]*Value, len(t))
		for i, el := range t {
			list[i] = anyToValue(el)
		}
		return &Value{Type: TypeList, ListVal: list}
	case map[string]any:
		m := &Value{Type: TypeMap, MapVal: make(map[string]*Value)}
		for k, val := range t {
			m.MapVal[k] = anyToValue(val)
			m.MapOrder = append(m.MapOrder, k)
		}
		return m
	}
	return StringVal(fmt.Sprintf("%v", v))
}
