package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"runtime"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx              context.Context
	db               *sql.DB
	ollamaCmd        *exec.Cmd
	cancelChat       context.CancelFunc
	history          []Message
	currentSessionID int64
	lang             string
	systemPrompt     string
	systemFileName   string
}

// translations holds all UI strings per language
var translations = map[string]map[string]string{
	"it": {
		"ollamaAlreadyRunning": "Ollama è già in esecuzione.",
		"ollamaNotFound":       "ollama non trovato nel PATH.",
		"ollamaStartFailed":    "Impossibile avviare Ollama: ",
		"ollamaNotResponding":  "Ollama avviato ma non risponde ancora — riprova tra poco.",
		"ollamaStarted":        "Ollama avviato.",
		"ollamaNotRunning":     "Ollama non era in esecuzione.",
		"ollamaStillActive":    "Ollama risulta ancora attivo.",
		"ollamaStopped":        "Ollama fermato.",
		"ollamaUnreachable":    "Ollama non raggiungibile su localhost:11434",
		"ollamaParseError":     "Errore nel parsing dei modelli",
		"selectModel":          "Seleziona un modello prima di inviare.",
		"promptEmpty":          "Il prompt non può essere vuoto.",
		"jsonError":            "Errore JSON: ",
		"requestError":         "Errore creazione richiesta: ",
		"connectionError":      "Errore connessione Ollama: ",
		"responseError":        "Errore lettura risposta: ",
	},
	"en": {
		"ollamaAlreadyRunning": "Ollama is already running.",
		"ollamaNotFound":       "ollama not found in PATH.",
		"ollamaStartFailed":    "Unable to start Ollama: ",
		"ollamaNotResponding":  "Ollama started but not responding yet — try again in a moment.",
		"ollamaStarted":        "Ollama started.",
		"ollamaNotRunning":     "Ollama was not running.",
		"ollamaStillActive":    "Ollama appears to still be active.",
		"ollamaStopped":        "Ollama stopped.",
		"ollamaUnreachable":    "Ollama unreachable on localhost:11434",
		"ollamaParseError":     "Error parsing models",
		"selectModel":          "Select a model before sending.",
		"promptEmpty":          "Prompt cannot be empty.",
		"jsonError":            "JSON error: ",
		"requestError":         "Request error: ",
		"connectionError":      "Ollama connection error: ",
		"responseError":        "Response read error: ",
	},
}

// SetLanguage sets the active language for backend messages
func (a *App) SetLanguage(lang string) {
	if _, ok := translations[lang]; ok {
		a.lang = lang
	}
}

// t is a helper that returns the translated string for a key
func (a *App) t(key string) string {
	if msgs, ok := translations[a.lang]; ok {
		if val, ok := msgs[key]; ok {
			return val
		}
	}
	// fallback to italian
	if msgs, ok := translations["it"]; ok {
		if val, ok := msgs[key]; ok {
			return val
		}
	}
	return key
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{lang: "it"}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	db, err := initDB()
	if err != nil {
		fmt.Println("Warning: could not initialise history DB:", err)
		return
	}
	a.db = db
}

// OllamaModel represents a model returned by the Ollama API
type OllamaModel struct {
	Name string `json:"name"`
}

// OllamaListResponse is the response from /api/tags
type OllamaListResponse struct {
	Models []OllamaModel `json:"models"`
}

// ChatResult is the result returned to the frontend
type ChatResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// FileContextResult is returned after loading a file as system prompt
type FileContextResult struct {
	Success  bool   `json:"success"`
	FileName string `json:"file_name"`
	Message  string `json:"message"`
}

// Messages
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OllamaStatus checks if Ollama is reachable
func (a *App) OllamaStatus() bool {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:11434/api/tags")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return true
}

// Check if Ollama is started by Systemd
func (a *App) OllamaIsSystemd() bool {
    if runtime.GOOS == "windows" {
        return false
    }
    out, err := exec.Command("systemctl", "is-active", "ollama").Output()
    if err != nil {
        return false
    }
    return strings.TrimSpace(string(out)) == "active"
}

// OllamaStart starts `ollama serve` as a background process
func (a *App) OllamaStart() ChatResult {
	if a.OllamaStatus() {
		return ChatResult{Success: true, Message: a.t("ollamaAlreadyRunning")}
	}

	path, err := exec.LookPath("ollama")
	if err != nil {
		return ChatResult{Success: false, Message: a.t("ollamaNotFound")}
	}

	cmd := exec.Command(path, "serve")

    // Hide the Terminal window on Windows
    setSysProcAttr(cmd)

	if err := cmd.Start(); err != nil {
		return ChatResult{Success: false, Message: a.t("ollamaStartFailed") + err.Error()}
	}
	a.ollamaCmd = cmd

	// Wait up to 10s for Ollama HTTP server to become reachable
	serverReady := false
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		if a.OllamaStatus() {
			serverReady = true
			break
		}
	}

	if !serverReady {
		return ChatResult{Success: false, Message: a.t("ollamaNotResponding")}
	}

	// Extra wait for the model store to finish loading after the HTTP server is up
	for i := 0; i < 10; i++ {
		names, err := a.GetModels()
		if err == nil && len(names) > 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}

	return ChatResult{Success: true, Message: a.t("ollamaStarted")}
}

// OllamaStop stops Ollama using pkill
func (a *App) OllamaStop() ChatResult {
    if !a.OllamaStatus() {
        return ChatResult{Success: true, Message: a.t("ollamaNotRunning")}
    }

    if a.ollamaCmd != nil && a.ollamaCmd.Process != nil {
        _ = a.ollamaCmd.Process.Kill()
        a.ollamaCmd = nil
    }

    // Fallback: termina il processo per nome
    if runtime.GOOS == "windows" {
        _ = exec.Command("taskkill", "/F", "/IM", "ollama.exe").Run()
    } else {
        _ = exec.Command("pkill", "-f", "ollama serve").Run()
    }

    time.Sleep(600 * time.Millisecond)
    if a.OllamaStatus() {
        return ChatResult{Success: false, Message: a.t("ollamaStillActive")}
    }
    return ChatResult{Success: true, Message: a.t("ollamaStopped")}
}

// GetModels returns the list of locally available Ollama models
func (a *App) GetModels() ([]string, error) {
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://localhost:11434/api/tags")
	if err != nil {
		return nil, fmt.Errorf("%s", a.t("ollamaUnreachable"))
	}
	defer resp.Body.Close()

	var result OllamaListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("%s", a.t("ollamaParseError"))
	}

	names := make([]string, 0, len(result.Models))
	for _, m := range result.Models {
		names = append(names, m.Name)
	}
	return names, nil
}

// OpenFileContext opens a native file picker and sets the chosen .txt/.md as system prompt
func (a *App) OpenFileContext() FileContextResult {
	path, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Apri file di contesto",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Testo (*.txt, *.md)", Pattern: "*.txt;*.md"},
		},
	})

	if err != nil {
		return FileContextResult{Success: false, Message: err.Error()}
	}
	if path == "" {
		return FileContextResult{Success: false, Message: ""}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return FileContextResult{Success: false, Message: err.Error()}
	}

	a.systemPrompt = string(data)
	a.systemFileName = filepath.Base(path)

	if a.db != nil && a.currentSessionID != 0 {
		_, _ = a.db.Exec(
			`UPDATE sessions SET system_prompt = ?, system_file_name = ? WHERE id = ?`,
			a.systemPrompt, a.systemFileName, a.currentSessionID,
		)
	}

	return FileContextResult{
		Success:  true,
		FileName: a.systemFileName,
	}
}

// ClearFileContext removes the current system prompt / file context
func (a *App) ClearFileContext() {
	a.systemPrompt = ""
	a.systemFileName = ""

	if a.db != nil && a.currentSessionID != 0 {
		_, _ = a.db.Exec(
			`UPDATE sessions SET system_prompt = '', system_file_name = '' WHERE id = ?`,
			a.currentSessionID,
		)
	}
}

// GetFileContext returns the current file name (empty string if none)
func (a *App) GetFileContext() string {
	return a.systemFileName
}

type ollamaChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type ollamaChatChunk struct {
	Message Message `json:"message"`
	Done    bool    `json:"done"`
}

// StopChat cancels the current streaming generation
func (a *App) StopChat() {
	if a.cancelChat != nil {
		a.cancelChat()
		a.cancelChat = nil
	}
}

// Chat sends a prompt to Ollama and streams the response token by token via events
func (a *App) Chat(model, prompt string) ChatResult {
	if strings.TrimSpace(model) == "" {
		return ChatResult{
			Success: false,
			Message: a.t("selectModel"),
		}
	}

	if strings.TrimSpace(prompt) == "" {
		return ChatResult{
			Success: false,
			Message: a.t("promptEmpty"),
		}
	}

	a.history = append(a.history, Message{
		Role:    "user",
		Content: prompt,
	})
	wailsRuntime.EventsEmit(a.ctx, "chat:history", a.history)

	ctx, cancel := context.WithCancel(context.Background())
	a.cancelChat = cancel

	defer func() {
		cancel()
		a.cancelChat = nil
	}()


	// Build messages: inject file context as explicit user+assistant exchange
	// before the conversation history, so every model understands the document.
	messages := make([]Message, 0, len(a.history)+3)
	if strings.TrimSpace(a.systemPrompt) != "" {
		// 1. System instruction
        messages = append(messages, Message{
            Role:    "system",
            Content: "You are a helpful assistant. The user has provided the content of a text file as reference. Use this information to answer their questions.",
        })
        // 2. Fake user message that presents the file explicitly
        messages = append(messages, Message{
            Role:    "user",
            Content: fmt.Sprintf("Here is the content of the file \"%s\":\n\n%s", a.systemFileName, a.systemPrompt),
        })
        // 3. Fake assistant acknowledgement
        messages = append(messages, Message{
            Role:    "assistant",
            Content: fmt.Sprintf("I have received the content of the file \"%s\". I am ready to answer your questions about it.", a.systemFileName),
        })
	}
	messages = append(messages, a.history...)

	reqData := ollamaChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}

	reqBody, err := json.Marshal(reqData)
	if err != nil {
		return ChatResult{
			Success: false,
			Message: a.t("jsonError") + err.Error(),
		}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"http://localhost:11434/api/chat",
		strings.NewReader(string(reqBody)),
	)
	if err != nil {
		return ChatResult{
			Success: false,
			Message: a.t("requestError") + err.Error(),
		}
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			wailsRuntime.EventsEmit(a.ctx, "chat:done", "")
			return ChatResult{Success: true, Message: ""}
		}
		return ChatResult{
			Success: false,
			Message: a.t("connectionError") + err.Error(),
		}
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var assistantResponse strings.Builder

	for {
		var chunk ollamaChatChunk
		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF || ctx.Err() != nil {
				break
			}
			return ChatResult{
				Success: false,
				Message: a.t("responseError") + err.Error(),
			}
		}

		token := chunk.Message.Content
		assistantResponse.WriteString(token)
		wailsRuntime.EventsEmit(a.ctx, "chat:token", token)

		if chunk.Done {
			break
		}
	}

	a.history = append(a.history, Message{
		Role:    "assistant",
		Content: assistantResponse.String(),
	})

	wailsRuntime.EventsEmit(a.ctx, "chat:history", a.history)
	wailsRuntime.EventsEmit(a.ctx, "chat:done", "")
	if a.db != nil {
		if err := a.saveSession(model); err != nil {
			fmt.Println("Warning: could not save session:", err)
		}
	}

	return ChatResult{
		Success: true,
		Message: "",
	}
}

func (a *App) ClearHistory() {
	a.history = []Message{}
}

func (a *App) GetHistory() []Message {
	return a.history
}
