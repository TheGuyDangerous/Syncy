import { StatusDot } from "./StatusDot";

export function Notice({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div className="notice" role="alert">
      <StatusDot tone="danger" />
      <span className="notice-text">{message}</span>
      {onRetry ? (
        <button className="btn btn--ghost btn--sm" onClick={onRetry}>
          Retry
        </button>
      ) : null}
    </div>
  );
}
