import type { ReactNode } from "react";

export function Screen({
  title,
  actions,
  children,
}: {
  title: string;
  actions?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="screen">
      <header className="topbar">
        <h1 className="topbar-title">{title}</h1>
        {actions ? <div className="topbar-actions">{actions}</div> : null}
      </header>
      <div className="content">{children}</div>
    </div>
  );
}
