import { useEffect, useState } from 'react';

export interface MetricData {
  timestamp: number;
  cpu_usage: number;
  cpu_load: number;
  cpu_temp: number;
  memory_usage: number;
  memory_total: number;
  memory_used: number;
  disk_read: number;
  disk_write: number;
  network_connections: number;
  network_up: number;
  network_down: number;
  created_at: string;
}

export interface HistoryQueryParams {
  start: number;
  end: number;
}

const API_BASE = '/api';

export const fetchCurrentMetrics = async (): Promise<MetricData> => {
  const response = await fetch(`${API_BASE}/metrics/current`);
  if (!response.ok) {
    throw new Error('Failed to fetch current metrics');
  }
  return response.json();
};

export const fetchHistoryMetrics = async (params: HistoryQueryParams): Promise<MetricData[]> => {
  const query = new URLSearchParams({
    start: params.start.toString(),
    end: params.end.toString(),
  });
  const response = await fetch(`${API_BASE}/metrics/range?${query.toString()}`);
  if (!response.ok) {
    throw new Error('Failed to fetch history metrics');
  }
  return response.json();
};

// 实时数据 Hook (SSE)
export const useRealtimeMetrics = (enabled: boolean) => {
  const [data, setData] = useState<MetricData | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    if (!enabled) {
      setConnected(false);
      return;
    }

    const eventSource = new EventSource('/sse/realtime');

    eventSource.onopen = () => {
      setConnected(true);
      setError(null);
    };

    eventSource.onmessage = (event) => {
      try {
        const parsed = JSON.parse(event.data);
        setData(parsed);
      } catch (e) {
        console.error('Failed to parse SSE data', e);
      }
    };

    eventSource.onerror = () => {
      setConnected(false);
      setError(new Error('SSE Connection Error'));
      eventSource.close();
    };

    return () => {
      eventSource.close();
      setConnected(false);
    };
  }, [enabled]);

  return { data, connected, error };
};
