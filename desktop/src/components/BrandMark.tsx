import { useId } from "react";

export function BrandMark({ size = 26, className }: { size?: number; className?: string }) {
  const gradient = `${useId()}g`;
  const small = size < 24;
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 96 96"
      className={className ? `brand-glyph ${className}` : "brand-glyph"}
      aria-hidden="true"
    >
      <defs>
        <linearGradient id={gradient} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#5b8cff" />
          <stop offset="1" stopColor="#7c5cff" />
        </linearGradient>
      </defs>
      <rect width="96" height="96" rx="22" fill="#0c0f16" />
      <line
        x1="36"
        y1="60"
        x2="60"
        y2="36"
        stroke={`url(#${gradient})`}
        strokeWidth={small ? 8 : 6}
        strokeLinecap="round"
      />
      <circle cx="36" cy="60" r="13" fill={`url(#${gradient})`} />
      <circle cx="60" cy="36" r="12.5" fill="none" stroke="#ffffff" strokeWidth={small ? 5.5 : 4} />
    </svg>
  );
}
