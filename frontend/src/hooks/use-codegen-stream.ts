import { useState, useEffect, useCallback, useRef } from 'react';
import type { SSEStatusEvent, SSEOutputEvent, SSEProgressEvent, SSEDoneEvent, SSELogEvent, StreamEntry } from '@/types';

interface UseCodegenStreamOptions {
  taskId: number | null;
  enabled?: boolean;
}

interface CodegenStreamState {
  status: SSEStatusEvent | null;
  entries: StreamEntry[];
  progress: SSEProgressEvent | null;
  done: SSEDoneEvent | null;
  error: string | null;
  connected: boolean;
}

export function useCodegenStream({ taskId, enabled = true }: UseCodegenStreamOptions) {
  const [state, setState] = useState<CodegenStreamState>({
    status: null,
    entries: [],
    progress: null,
    done: null,
    error: null,
    connected: false,
  });

  const eventSourceRef = useRef<EventSource | null>(null);
  const lastEventIdRef = useRef<string>('');
  const doneRef = useRef(false);

  const connect = useCallback(() => {
    if (!taskId || !enabled || doneRef.current) return;

    const token = localStorage.getItem('token');
    const url = `/api/v1/codegen/${taskId}/stream?token=${encodeURIComponent(token || '')}${
      lastEventIdRef.current ? `&last_event_id=${lastEventIdRef.current}` : ''
    }`;

    const es = new EventSource(url);
    eventSourceRef.current = es;

    es.onopen = () => {
      setState((prev) => ({ ...prev, connected: true, error: null }));
    };

    es.addEventListener('status', (e: MessageEvent) => {
      lastEventIdRef.current = (e as MessageEvent & { lastEventId?: string }).lastEventId || '';
      try {
        const data: SSEStatusEvent = JSON.parse(e.data);
        setState((prev) => ({ ...prev, status: data }));
      } catch { /* ignore parse errors */ }
    });

    es.addEventListener('log', (e: MessageEvent) => {
      lastEventIdRef.current = (e as MessageEvent & { lastEventId?: string }).lastEventId || '';
      try {
        const data: SSELogEvent = JSON.parse(e.data);
        const entry: StreamEntry = { kind: 'log', data };
        setState((prev) => ({ ...prev, entries: [...prev.entries, entry] }));
      } catch { /* ignore parse errors */ }
    });

    es.addEventListener('output', (e: MessageEvent) => {
      lastEventIdRef.current = (e as MessageEvent & { lastEventId?: string }).lastEventId || '';
      try {
        const data: SSEOutputEvent = JSON.parse(e.data);
        const entry: StreamEntry = { kind: 'output', data };
        setState((prev) => ({ ...prev, entries: [...prev.entries, entry] }));
      } catch { /* ignore parse errors */ }
    });

    es.addEventListener('progress', (e: MessageEvent) => {
      lastEventIdRef.current = (e as MessageEvent & { lastEventId?: string }).lastEventId || '';
      try {
        const data: SSEProgressEvent = JSON.parse(e.data);
        setState((prev) => ({ ...prev, progress: data }));
      } catch { /* ignore parse errors */ }
    });

    es.addEventListener('task_error', (e: MessageEvent) => {
      lastEventIdRef.current = (e as MessageEvent & { lastEventId?: string }).lastEventId || '';
      try {
        const data = JSON.parse(e.data);
        setState((prev) => ({ ...prev, error: data.message || '发生错误' }));
      } catch { /* ignore */ }
    });

    es.addEventListener('done', (e: MessageEvent) => {
      lastEventIdRef.current = (e as MessageEvent & { lastEventId?: string }).lastEventId || '';
      doneRef.current = true;
      try {
        const data: SSEDoneEvent = JSON.parse(e.data);
        setState((prev) => ({ ...prev, done: data }));
      } catch { /* ignore */ }
      es.close();
      setState((prev) => ({ ...prev, connected: false }));
    });

    es.onerror = () => {
      setState((prev) => ({ ...prev, connected: false }));
      es.close();
      if (!doneRef.current) {
        setTimeout(() => {
          connect();
        }, 3000);
      }
    };
  }, [taskId, enabled]);

  useEffect(() => {
    doneRef.current = false;
    connect();
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }
    };
  }, [connect]);

  const reset = useCallback(() => {
    setState({
      status: null,
      entries: [],
      progress: null,
      done: null,
      error: null,
      connected: false,
    });
    lastEventIdRef.current = '';
    doneRef.current = false;
  }, []);

  return { ...state, reset };
}
