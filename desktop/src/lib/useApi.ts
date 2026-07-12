import { useCallback, useEffect, useState } from "react";
import { errorMessage } from "./api";

export interface ApiResult<T> {
  data: T | null;
  error: string | null;
  loading: boolean;
  reload: () => void;
}

interface ApiState<T> {
  data: T | null;
  error: string | null;
  loading: boolean;
}

export function useApi<T>(fetcher: () => Promise<T>): ApiResult<T> {
  const [state, setState] = useState<ApiState<T>>({
    data: null,
    error: null,
    loading: true,
  });
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let live = true;
    setState((s) => ({ ...s, loading: true }));
    fetcher()
      .then((data) => {
        if (live) setState({ data, error: null, loading: false });
      })
      .catch((e: unknown) => {
        if (live) setState((s) => ({ data: s.data, error: errorMessage(e), loading: false }));
      });
    return () => {
      live = false;
    };
  }, [tick]);

  const reload = useCallback(() => setTick((t) => t + 1), []);

  return { data: state.data, error: state.error, loading: state.loading, reload };
}
