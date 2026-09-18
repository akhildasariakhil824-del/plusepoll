import { useState, useEffect, useRef } from 'react';

export const usePollWebSocket = (pollId, onUpdate) => {
  const [connectionStatus, setConnectionStatus] = useState('connecting');
  const [liveData, setLiveData] = useState({
    totalVotes: 0,
    optionVotes: {},
    percentages: {},
    viewerCount: 1,
    isActive: true,
  });
  
  const wsRef = useRef(null);
  const reconnectTimeoutRef = useRef(null);
  const attemptsRef = useRef(0);

  useEffect(() => {
    if (!pollId) return;

    let isMounted = true;

    const connectWebSocket = () => {
      // Determine protocol and host
      let wsUrl = import.meta.env.VITE_WS_BASE;
      if (!wsUrl) {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        wsUrl = `${protocol}//${window.location.host}/api/ws/polls/${pollId}`;
      } else {
        wsUrl = `${wsUrl}/api/ws/polls/${pollId}`;
      }

      setConnectionStatus(attemptsRef.current === 0 ? 'connecting' : 'reconnecting');

      try {
        const ws = new WebSocket(wsUrl);
        wsRef.current = ws;

        ws.onopen = () => {
          if (!isMounted) return;
          setConnectionStatus('connected');
          attemptsRef.current = 0;
        };

        ws.onmessage = (event) => {
          if (!isMounted) return;
          try {
            const data = JSON.parse(event.data);
            
            setLiveData((prev) => {
              const updated = {
                ...prev,
                totalVotes: data.total_votes !== undefined ? data.total_votes : prev.totalVotes,
                optionVotes: data.option_votes || prev.optionVotes,
                percentages: data.percentages || prev.percentages,
                viewerCount: data.viewer_count !== undefined ? data.viewer_count : prev.viewerCount,
                isActive: data.is_active !== undefined ? data.is_active : prev.isActive,
              };

              if (onUpdate) {
                onUpdate(updated, data.type);
              }
              return updated;
            });
          } catch (err) {
            console.error('Error parsing WS message:', err);
          }
        };

        ws.onclose = () => {
          if (!isMounted) return;
          setConnectionStatus('disconnected');
          // Reconnect with exponential backoff (max 5s)
          attemptsRef.current += 1;
          const delay = Math.min(1000 * Math.pow(1.5, attemptsRef.current), 5000);
          reconnectTimeoutRef.current = setTimeout(() => {
            if (isMounted) connectWebSocket();
          }, delay);
        };

        ws.onerror = (err) => {
          console.warn('WebSocket encountered error, closing to trigger reconnect:', err);
          ws.close();
        };
      } catch (err) {
        console.error('Failed to instantiate WebSocket:', err);
        setConnectionStatus('disconnected');
      }
    };

    connectWebSocket();

    return () => {
      isMounted = false;
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [pollId]);

  return { connectionStatus, liveData };
};
