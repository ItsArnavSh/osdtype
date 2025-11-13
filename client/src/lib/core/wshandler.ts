import { BASE_URL, WS_BASE_URL } from '$lib/config';

type MessageHandler = (data: any) => void;

export class GameComm {
	private ws: WebSocket | null = null;
	private messageHandlers: MessageHandler[] = [];
	private isConnecting: boolean = false;

	// Connect to WebSocket
	async connect(useWs = true): Promise<void> {
		if (useWs) {
			if (this.isConnecting) {
				return;
			}

			// Reuse existing connection if open
			if (this.ws && this.ws.readyState === WebSocket.OPEN) {
				console.log('Already connected');
				return;
			}

			this.isConnecting = true;

			return new Promise((resolve, reject) => {
				this.ws = new WebSocket(`${WS_BASE_URL}/user/imonline`);

				this.ws.onopen = () => {
					console.log('Online via WebSocket');
					this.isConnecting = false;
					resolve();
				};

				this.ws.onmessage = (event) => {
					try {
						const data = JSON.parse(event.data);
						console.log('Received:', data);
						// Call all registered message handlers
						this.messageHandlers.forEach((handler) => handler(data));
					} catch (err) {
						console.error('Failed to parse message:', err);
					}
				};

				this.ws.onclose = () => {
					console.log('Disconnected from WebSocket');
					this.ws = null;
					this.isConnecting = false;
				};

				this.ws.onerror = (err) => {
					console.error('WS Error:', err);
					this.isConnecting = false;
					reject(err);
				};
			});
		} else {
			// HTTP fallback for local/dev
			const res = await fetch(`${BASE_URL}/user/imonline`, {
				method: 'GET',
				credentials: 'include'
			});
			if (!res.ok) throw new Error('Session Not Marked Online');
			return await res.json();
		}
	}

	// Send data through WebSocket
	send(data: any): void {
		if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
			console.error('WebSocket not connected');
			return;
		}
		this.ws.send(JSON.stringify(data));
		console.log('Sent:', data);
	}

	// Register a message handler
	onMessage(handler: MessageHandler): void {
		this.messageHandlers.push(handler);
	}

	// Remove a message handler
	offMessage(handler: MessageHandler): void {
		this.messageHandlers = this.messageHandlers.filter((h) => h !== handler);
	}

	// Check if connected
	isConnected(): boolean {
		return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
	}

	// Disconnect
	disconnect(): void {
		if (this.ws) {
			this.ws.close();
			this.ws = null;
		}
	}
}
