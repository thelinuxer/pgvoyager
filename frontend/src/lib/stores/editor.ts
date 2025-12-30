import { writable, get } from 'svelte/store';
import { tabs, activeTab } from './tabs';

export interface EditorState {
	content: string;
	lastUpdatedBy: 'user' | 'claude' | null;
}

export interface EditorAction {
	action: 'insert' | 'replace';
	text: string;
	position?: { line: number; column: number };
}

function createEditorStore() {
	const { subscribe, set, update } = writable<EditorState>({
		content: '',
		lastUpdatedBy: null
	});

	// Pending action from Claude that the editor should apply
	const pendingAction = writable<EditorAction | null>(null);

	// Track if we have an active query editor
	let hasActiveQueryTab = false;
	activeTab.subscribe((tab) => {
		hasActiveQueryTab = tab?.type === 'query';
	});

	return {
		subscribe,

		// Called by QueryEditor when user types
		setContent(content: string) {
			update((state) => ({
				...state,
				content,
				lastUpdatedBy: 'user'
			}));
		},

		// Called when Claude sends an action via WebSocket
		applyAction(action: EditorAction) {
			// If no query tab is open, create one with the content
			if (!hasActiveQueryTab) {
				const content = action.action === 'replace' ? action.text : action.text;
				tabs.openQuery({ title: 'Claude Query', initialSql: content });

				// Update our state
				update((state) => ({
					...state,
					content,
					lastUpdatedBy: 'claude'
				}));
				return;
			}

			if (action.action === 'replace') {
				update((state) => ({
					...state,
					content: action.text,
					lastUpdatedBy: 'claude'
				}));
			} else if (action.action === 'insert') {
				const currentState = get({ subscribe });
				const content = currentState.content;

				// If position is specified, insert at that position
				// Otherwise, append to end
				let newContent: string;
				if (action.position) {
					// Convert line/column to character offset
					const lines = content.split('\n');
					let offset = 0;
					for (let i = 0; i < action.position.line && i < lines.length; i++) {
						offset += lines[i].length + 1; // +1 for newline
					}
					offset += Math.min(action.position.column, lines[action.position.line]?.length || 0);

					newContent = content.slice(0, offset) + action.text + content.slice(offset);
				} else {
					// Append to end with newline if content exists
					newContent = content ? content + '\n' + action.text : action.text;
				}

				update((state) => ({
					...state,
					content: newContent,
					lastUpdatedBy: 'claude'
				}));
			}

			// Also set pending action for CodeMirror to pick up
			pendingAction.set(action);
		},

		// Get pending action and clear it
		getPendingAction(): EditorAction | null {
			const action = get(pendingAction);
			pendingAction.set(null);
			return action;
		},

		// Subscribe to pending actions
		subscribeToPendingActions: pendingAction.subscribe,

		getContent(): string {
			return get({ subscribe }).content;
		}
	};
}

export const editorStore = createEditorStore();
