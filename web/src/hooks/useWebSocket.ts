import { useEffect, useRef, useState } from 'react';

export interface WebSocketEvent {
  type: string;
  payload: any;
  time: string;
}

export interface UseWebSocketOptions {
  onMessage?: (event: WebSocketEvent) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  reconnectInterval?: number;
}

export function useWebSocket(url: string, options: UseWebSocketOptions = {}) {
  const [isConnected, setIsConnected] = useState(false);
  const [lastMessage, setLastMessage] = useState<WebSocketEvent | null>(null);
  const ws = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<NodeJS.Timeout>();

  const connect = () => {
    try {
      // Convert http:// to ws://
      const wsUrl = url.replace(/^http/, 'ws');
      ws.current = new WebSocket(wsUrl);

      ws.current.onopen = () => {
        console.log(`WebSocket connected: ${url}`);
        setIsConnected(true);
        if (options.onConnect) {
          options.onConnect();
        }
      };

      ws.current.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data) as WebSocketEvent;
          setLastMessage(data);
          if (options.onMessage) {
            options.onMessage(data);
          }
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err);
        }
      };

      ws.current.onerror = (error) => {
        console.error('WebSocket error:', error);
      };

      ws.current.onclose = () => {
        console.log(`WebSocket disconnected: ${url}`);
        setIsConnected(false);
        if (options.onDisconnect) {
          options.onDisconnect();
        }

        // Attempt to reconnect
        const interval = options.reconnectInterval || 5000;
        reconnectTimer.current = setTimeout(() => {
          console.log(`Reconnecting to ${url}...`);
          connect();
        }, interval);
      };
    } catch (err) {
      console.error('Failed to create WebSocket:', err);
    }
  };

  useEffect(() => {
    connect();

    return () => {
      if (reconnectTimer.current) {
        clearTimeout(reconnectTimer.current);
      }
      if (ws.current) {
        ws.current.close();
      }
    };
  }, [url]);

  return {
    isConnected,
    lastMessage,
  };
}
