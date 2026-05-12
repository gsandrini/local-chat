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
    },
};

/*  APP  */
function LocalChat() {
    return {
        models: [],
        selectedModel: '',
        prompt: '',
        output: '',
        loading: false,
        error: null,
        ollamaOnline: false,
        ollamaIsSystemd: false,
        ollamaLoading: false,
        ollamaModelLoading: false,
        history: [],

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

                window.runtime.EventsOn('chat:done', () => {
                    this.loading = false;
                });

                window.runtime.EventsOn('chat:history', (history) => {
                    this.history = history;
                });
            }
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
            } finally {
                setTimeout(() => {
                    this.ollamaModelLoading = false;
                }, 600); // the duration must be equal to the animation                                        
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
            await window.go.main.App.ClearHistory();
            this.loading = false;
        },

        handleKeydown(event) {
            if ((event.metaKey || event.ctrlKey) && event.key === 'Enter') {
                event.preventDefault();
                this.send();
            }
        },

        closeError(event) {
            this.error = null;
        }
    };
}
