import { writable, get } from 'svelte/store';
import { editorStore, type EditorAction } from './editor';

// Get API base URL dynamically based on environment
function getApiBase(): string {
	if (typeof window === 'undefined') return '';

	// Development mode (Vite dev server on port 5173)
	if (window.location.port === '5173') {
		return 'http://localhost:5137';
	}

	// Production mode - use same origin
	return '';
}

// Get WebSocket base URL dynamically
function getWsBase(): string {
	if (typeof window === 'undefined') return '';

	// Development mode (Vite dev server on port 5173)
	if (window.location.port === '5173') {
		return 'ws://localhost:5137';
	}

	// Production mode - derive from current location
	const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
	return `${protocol}//${window.location.host}`;
}

// Terminal type - using interface to avoid importing CommonJS module
interface ITerminal {
	write(data: string): void;
	onData(callback: (data: string) => void): void;
	clear(): void;
}

export interface EditorState {
	content: string;
	selection?: { start: number; end: number };
	cursor?: { line: number; column: number };
}

interface ClaudeTerminalState {
	sessionId: string | null;
	connectionId: string | null; // Track which connection this session is for
	isConnected: boolean;
	isConnecting: boolean;
	error: string | null;
}

function createClaudeTerminalStore() {
	const { subscribe, set, update } = writable<ClaudeTerminalState>({
		sessionId: null,
		connectionId: null,
		isConnected: false,
		isConnecting: false,
		error: null
	});

	let ws: WebSocket | null = null;
	let terminal: ITerminal | null = null;
	let destroyingPromise: Promise<void> | null = null;

	async function createSession(connectionId: string): Promise<string | null> {
		// Wait for any ongoing destroy to complete first
		if (destroyingPromise) {
			await destroyingPromise;
		}

		// Check if we already have a session
		const currentState = get({ subscribe });
		if (currentState.sessionId) {
			// If same connection, reuse existing session
			if (currentState.connectionId === connectionId) {
				return currentState.sessionId;
			}
			// If different connection, update the session's connection (no restart needed)
			const updated = await updateSessionConnection(currentState.sessionId, connectionId);
			if (updated) {
				update((state) => ({ ...state, connectionId }));
				if (terminal) {
					terminal.write('\r\n\x1b[33mSwitched to a different database connection.\x1b[0m\r\n');
				}
				return currentState.sessionId;
			}
			// If update failed, fall through to create new session
		}

		update((state) => ({ ...state, isConnecting: true, error: null }));

		try {
			const response = await fetch(`${getApiBase()}/api/claude/sessions`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ connectionId })
			});

			if (!response.ok) {
				const error = await response.json();
				throw new Error(error.error || 'Failed to create session');
			}

			const data = await response.json();
			update((state) => ({
				...state,
				sessionId: data.sessionId,
				connectionId: connectionId
			}));
			return data.sessionId;
		} catch (e) {
			const errorMessage = e instanceof Error ? e.message : 'Failed to create session';
			update((state) => ({ ...state, error: errorMessage, isConnecting: false }));
			return null;
		}
	}

	async function destroySession(): Promise<void> {
		const state = get({ subscribe });
		if (!state.sessionId) return;

		// Track the destroy operation so createSession can wait for it
		const doDestroy = async () => {
			disconnect();

			try {
				await fetch(`${getApiBase()}/api/claude/sessions/${state.sessionId}`, {
					method: 'DELETE'
				});
			} catch (e) {
				console.error('Failed to destroy session:', e);
			}

			set({
				sessionId: null,
				connectionId: null,
				isConnected: false,
				isConnecting: false,
				error: null
			});
		};

		destroyingPromise = doDestroy();
		await destroyingPromise;
		destroyingPromise = null;
	}

	// Update the session's database connection without restarting Claude
	async function updateSessionConnection(sessionId: string, connectionId: string): Promise<boolean> {
		try {
			const response = await fetch(`${getApiBase()}/api/claude/sessions/${sessionId}/connection`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ connectionId })
			});

			if (!response.ok) {
				console.error('Failed to update session connection');
				return false;
			}

			return true;
		} catch (e) {
			console.error('Failed to update session connection:', e);
			return false;
		}
	}

	// Check if connection changed and update session if needed
	async function ensureSessionForConnection(connectionId: string): Promise<boolean> {
		const state = get({ subscribe });

		if (state.connectionId !== connectionId) {
			if (state.sessionId) {
				// Update existing session's connection
				const updated = await updateSessionConnection(state.sessionId, connectionId);
				if (updated) {
					update((s) => ({ ...s, connectionId }));
					return true;
				}
			}
			// No session or update failed, create new session
			const sessionId = await createSession(connectionId);
			return sessionId !== null;
		}

		return state.sessionId !== null;
	}

	function getConnectionId(): string | null {
		return get({ subscribe }).connectionId;
	}

	// Store pending resize to send when WebSocket opens
	let pendingResize: { cols: number; rows: number } | null = null;

	function connect(term: ITerminal): void {
		const state = get({ subscribe });
		if (!state.sessionId) {
			update((s) => ({ ...s, error: 'No session ID' }));
			return;
		}

		terminal = term;

		// Create WebSocket connection
		const wsUrl = `${getWsBase()}/api/claude/terminal/${state.sessionId}`;
		ws = new WebSocket(wsUrl);

		ws.onopen = () => {
			update((s) => ({ ...s, isConnected: true, isConnecting: false }));
			// Send pending resize immediately
			if (pendingResize && ws && ws.readyState === WebSocket.OPEN) {
				ws.send(JSON.stringify({ type: 'resize', data: pendingResize }));
				pendingResize = null;
			}
		};

		ws.onmessage = (event) => {
			try {
				const msg = JSON.parse(event.data);
				if (msg.type === 'output' && terminal) {
					terminal.write(msg.data);
				} else if (msg.type === 'editor_action') {
					// Handle editor actions from Claude
					handleEditorAction(msg.data);
				}
			} catch (e) {
				console.error('Failed to parse WebSocket message:', e);
			}
		};

		ws.onerror = (error) => {
			console.error('WebSocket error:', error);
			update((s) => ({ ...s, error: 'WebSocket connection error' }));
		};

		ws.onclose = () => {
			update((s) => ({ ...s, isConnected: false }));
			ws = null;
		};

		// Forward terminal input to WebSocket
		terminal.onData((data) => {
			sendInput(data);
		});
	}

	function disconnect(): void {
		if (ws) {
			ws.close();
			ws = null;
		}
		terminal = null;
		update((s) => ({ ...s, isConnected: false }));
	}

	function sendInput(data: string): void {
		if (ws && ws.readyState === WebSocket.OPEN) {
			ws.send(JSON.stringify({ type: 'input', data: { data } }));
		}
	}

	function resize(cols: number, rows: number): void {
		if (ws && ws.readyState === WebSocket.OPEN) {
			ws.send(JSON.stringify({ type: 'resize', data: { cols, rows } }));
		} else {
			// Store for sending when WebSocket opens
			pendingResize = { cols, rows };
		}
	}

	function updateEditorState(state: EditorState): void {
		if (ws && ws.readyState === WebSocket.OPEN) {
			ws.send(JSON.stringify({ type: 'editor_update', data: state }));
		}
	}

	interface EditorActionData {
		action: string;
		text: string;
		position?: { line: number; column: number };
	}

	function handleEditorAction(data: EditorActionData): void {
		// Apply the action to the editor store
		if (data.action === 'insert' || data.action === 'replace') {
			editorStore.applyAction({
				action: data.action as 'insert' | 'replace',
				text: data.text,
				position: data.position
			});
		}
	}

	return {
		subscribe,
		createSession,
		destroySession,
		ensureSessionForConnection,
		getConnectionId,
		connect,
		disconnect,
		sendInput,
		resize,
		updateEditorState
	};
}

export const claudeTerminal = createClaudeTerminalStore();
