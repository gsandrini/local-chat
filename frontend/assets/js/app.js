'use strict';

/*  TRANSLATIONS  */
const TRANSLATIONS = {
    it: {
        appTitle: 'LocalChat',
        statusWaiting: 'attendi...',
        statusOnline: 'online',
        statusOffline: 'offline',
        btnStartOllama: 'Avvia Ollama',
        btnStopOllama: 'Ferma Ollama',
        labelModel: 'Modello',
        modelStartOllama: '— avvia Ollama per vedere i modelli —',
        modelNoneFound: '— nessun modello trovato —',
        labelPrompt: 'Domanda',
        promptHint: '(Ctrl+Enter per inviare)',
        promptPlaceholder: 'Scrivi qui la tua domanda...',
        btnSend: 'Invia',
        btnSending: 'Generazione in corso...',
        btnStopChat: 'Stop',
        footerMadeWith: 'Sviluppata con il supporto di',
        footerBy: '(Anthropic)',
        ollamaIsSystemd: 'Ollama è gestito da systemd — usa il terminale per avviarlo/fermarlo',
        newChat: 'Nuova chat',
        noSessions: 'Nessuna chat salvata',
        sidebarTitle: 'Chats',
        sidebarHide: 'Nascondi',
        sidebarShow: 'Mostra',
        deleteSession: 'Cancella',
        renameSession: 'Rinomina',
        renameModalTitle: 'Rinomina chat',
        btnCancel: 'Annulla',
        btnSave: 'Salva',
        attachFile: 'Allega file di contesto (.txt, .md)',
        removeFile: 'Rimuovi file di contesto',
        fileContext: 'Contesto attivo',
    },
    en: {
        appTitle: 'LocalChat',
        statusWaiting: 'please wait...',
        statusOnline: 'online',
        statusOffline: 'offline',
        btnStartOllama: 'Start Ollama',
        btnStopOllama: 'Stop Ollama',
        labelModel: 'Model',
        modelStartOllama: '— start Ollama to see models —',
        modelNoneFound: '— no models found —',
        labelPrompt: 'Question',
        promptHint: '(Ctrl+Enter to send)',
        promptPlaceholder: 'Type your question here...',
        btnSend: 'Send',
        btnSending: 'Generating...',
        btnStopChat: 'Stop',
        footerMadeWith: 'Built with the support of',
        footerBy: '(Anthropic)',
        ollamaIsSystemd: 'Ollama is managed by systemd — stop/start from terminal',
        newChat: 'New chat',
        noSessions: 'No saved chats',
        sidebarTitle: 'Chats',
        sidebarHide: 'Hide',
        sidebarShow: 'Show',
        deleteSession: 'Delete',
        renameSession: 'Rename',
        renameModalTitle: 'Rename chat',
        btnCancel: 'Cancel',
        btnSave: 'Save',
        attachFile: 'Attach context file (.txt, .md)',
        removeFile: 'Remove context file',
        fileContext: 'Active context',
    },
};

/*  APP  */
function LocalChat() {
    return {
        models: [],
        selectedModel: '',
        prompt: '',
        loading: false,
        error: null,
        ollamaOnline: false,
        ollamaIsSystemd: false,
        ollamaLoading: false,
        ollamaModelLoading: false,
        history: [],

        // file context
        fileContextName: '',

        // history sidebar
        sessions: [],
        activeSessionId: null,
        sidebarOpen: true,
        renameModal: { open: false, id: null, title: '' },

        // i18n
        lang: navigator.language.startsWith('it') ? 'it' : 'en',
        get t() { return TRANSLATIONS[this.lang]; },

        async init() {
            await this.checkStatus();

            if (window.runtime) {
                window.runtime.EventsOff('chat:token');
                window.runtime.EventsOff('chat:done');
                window.runtime.EventsOff('chat:history');

                window.runtime.EventsOn('chat:token', (token) => {
                    const last = this.history[this.history.length - 1];
                    if (last && last.role === 'assistant') {
                        last.content += token;
                    } else {
                        this.history.push({ role: 'assistant', content: token });
                    }

                    this.$nextTick(() => {
                        const el = this.$refs.history;
                        if (el) el.scrollTop = el.scrollHeight;
                    });
                });

                window.runtime.EventsOn('chat:done', async () => {
                    this.loading = false;
                    this.sessions = await window.go.main.App.GetSessions();
                    if (this.sessions.length > 0 && this.activeSessionId === null) {
                        this.activeSessionId = this.sessions[0].id;
                    }
                });

                window.runtime.EventsOn('chat:history', (history) => {
                    this.history = history;
                });
            }

            // Load session list and current file context on startup
            this.sessions = await window.go.main.App.GetSessions();
            this.fileContextName = await window.go.main.App.GetFileContext();
        },

        async checkStatus() {
            if (!window.go?.main?.App) {
                return;
            }
            this.ollamaIsSystemd = await window.go.main.App.OllamaIsSystemd();
            this.ollamaOnline = await window.go.main.App.OllamaStatus();
            if (this.ollamaOnline) await this.loadModels();
        },

        async loadModels() {
            if (!window.go?.main?.App) {
                return;
            }
            try {
                this.ollamaModelLoading = true;
                const result = await window.go.main.App.GetModels() || [];
                this.models = result;
                if (this.models.length > 0 && !this.selectedModel) {
                    this.selectedModel = this.models[0];
                }
            } catch (e) {
                this.models = [];
                this.error = String(e);
            } finally {
                setTimeout(() => {
                    this.ollamaModelLoading = false;
                }, 600);
            }
        },

        async startOllama() {
            if (!window.go?.main?.App) {
                return;
            }
            this.ollamaLoading = true;
            this.error = null;
            try {
                const res = await window.go.main.App.OllamaStart();
                if (res.success) {
                    this.ollamaOnline = true;
                    await this.loadModels();
                } else {
                    this.error = res.message;
                    this.ollamaOnline = false;
                }
            } catch (e) {
                this.error = String(e);
            } finally {
                this.ollamaLoading = false;
            }
        },

        async stopOllama() {
            if (!window.go?.main?.App) {
                return;
            }
            this.ollamaLoading = true;
            this.error = null;
            try {
                const res = await window.go.main.App.OllamaStop();
                if (res.success) {
                    this.ollamaOnline = false;
                    this.models = [];
                    this.selectedModel = '';
                } else {
                    this.error = res.message;
                }
            } catch (e) {
                this.error = String(e);
            } finally {
                this.ollamaLoading = false;
            }
        },

        async send() {
            if (this.loading || !this.prompt.trim()) return;
            this.error = null;
            this.loading = true;
            const text = this.prompt;
            this.history.push({ role: 'user', content: text });
            this.prompt = '';

            try {
                if (window.go?.main?.App) {
                    const res = await window.go.main.App.Chat(this.selectedModel, text);
                    if (!res.success && res.message) {
                        this.error = res.message;
                        this.loading = false;
                    }
                }
            } catch (e) {
                this.error = String(e);
                this.loading = false;
            }
        },

        async stopChat() {
            if (!window.go?.main?.App) {
                return;
            }
            await window.go.main.App.StopChat();
            this.loading = false;
        },

        async newChat() {
            if (!window.go?.main?.App) {
                return;
            }
            this.history = [];
            this.activeSessionId = null;
            this.fileContextName = '';
            await window.go.main.App.NewSession();
            this.loading = false;
        },

        // File context
        async attachFile() {
            if (!window.go?.main?.App) return;
            try {
                const res = await window.go.main.App.OpenFileContext();
                if (res.success) {
                    this.fileContextName = res.file_name;
                } else if (res.message) {
                    this.error = res.message;
                }
            } catch (e) {
                this.error = String(e);
            }
        },

        async removeFileContext() {
            if (!window.go?.main?.App) return;
            try {
                await window.go.main.App.ClearFileContext();
                this.fileContextName = '';
            } catch (e) {
                this.error = String(e);
            }
        },

        // Session management
        async loadSession(id) {
            if (!window.go?.main?.App) {
                return;
            }
            try {
                const messages = await window.go.main.App.LoadSession(id);
                this.history = messages || [];
                this.activeSessionId = id;
                // Restore file context name for this session
                this.fileContextName = await window.go.main.App.GetFileContext();
                this.$nextTick(() => {
                    const el = this.$refs.history;
                    if (el) el.scrollTop = el.scrollHeight;
                });
            } catch (e) {
                this.error = String(e);
            }
        },

        async deleteSession(id) {
            if (!window.go?.main?.App) return;
            try {
                await window.go.main.App.DeleteSession(id);
                this.sessions = this.sessions.filter(s => s.id !== id);
                if (this.activeSessionId === id) {
                    this.history = [];
                    this.activeSessionId = null;
                    this.fileContextName = '';
                }
            } catch (e) {
                this.error = String(e);
            }
        },

        openRenameModal(session) {
            this.renameModal = { open: true, id: session.id, title: session.title };
            this.$nextTick(() => {
                const el = this.$refs.renameInput;
                if (el) { el.focus(); el.select(); }
            });
        },

        closeRenameModal() {
            this.renameModal = { open: false, id: null, title: '' };
        },

        async saveRename() {
            const { id, title } = this.renameModal;
            if (!title.trim() || !window.go?.main?.App) return;
            try {
                await window.go.main.App.RenameSession(id, title.trim());
                const session = this.sessions.find(s => s.id === id);
                if (session) session.title = title.trim();
            } catch (e) {
                this.error = String(e);
            } finally {
                this.closeRenameModal();
            }
        },

        handleKeydown(event) {
            if ((event.metaKey || event.ctrlKey) && event.key === 'Enter') {
                event.preventDefault();
                this.send();
            }
        },

        closeError() {
            this.error = null;
        },
    };
}
