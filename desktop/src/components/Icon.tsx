const glyphs = {
  dashboard: (
    <>
      <rect x="3.5" y="3.5" width="7" height="7" rx="1.8" />
      <rect x="13.5" y="3.5" width="7" height="7" rx="1.8" />
      <rect x="3.5" y="13.5" width="7" height="7" rx="1.8" />
      <rect x="13.5" y="13.5" width="7" height="7" rx="1.8" />
    </>
  ),
  devices: (
    <>
      <rect x="3" y="5" width="18" height="12" rx="2" />
      <path d="M9 20.5h6M12 17v3.5" />
    </>
  ),
  folder: (
    <path d="M3.5 7a2 2 0 0 1 2-2h3.9l2 2.4h7.1a2 2 0 0 1 2 2V17a2 2 0 0 1-2 2h-13a2 2 0 0 1-2-2z" />
  ),
  transfers: (
    <>
      <path d="M7.5 17.5v-11M7.5 6.5 4 10M7.5 6.5 11 10" />
      <path d="M16.5 6.5v11M16.5 17.5 13 14M16.5 17.5 20 14" />
    </>
  ),
  history: (
    <>
      <circle cx="12" cy="12" r="8.5" />
      <path d="M12 7.5V12l3.2 1.9" />
    </>
  ),
  versions: (
    <>
      <path d="m12 3.5 8.5 4.4-8.5 4.4-8.5-4.4z" />
      <path d="M3.5 12.4 12 16.8l8.5-4.4" />
      <path d="M3.5 16.4 12 20.8l8.5-4.4" />
    </>
  ),
  conflicts: (
    <>
      <path d="M10.4 4.9 3.2 17.6a1.8 1.8 0 0 0 1.6 2.7h14.4a1.8 1.8 0 0 0 1.6-2.7L13.6 4.9a1.85 1.85 0 0 0-3.2 0z" />
      <path d="M12 9.5v4.2M12 17h.01" />
    </>
  ),
  logs: (
    <>
      <rect x="3" y="4.5" width="18" height="15" rx="2" />
      <path d="m7 9.5 3 2.7-3 2.7M12.5 15.5H17" />
    </>
  ),
  settings: (
    <>
      <path d="M4 7.5h5M13 7.5h7M4 16.5h7M15 16.5h5" />
      <circle cx="11" cy="7.5" r="2" />
      <circle cx="13" cy="16.5" r="2" />
    </>
  ),
  plus: <path d="M12 5.5v13M5.5 12h13" />,
  trash: (
    <>
      <path d="M4.5 7h15" />
      <path d="M9.5 7V5.6A1.6 1.6 0 0 1 11.1 4h1.8a1.6 1.6 0 0 1 1.6 1.6V7" />
      <path d="m6.5 7 .7 11.5A1.6 1.6 0 0 0 8.8 20h6.4a1.6 1.6 0 0 0 1.6-1.5L17.5 7" />
      <path d="M10.2 10.5v6M13.8 10.5v6" />
    </>
  ),
  shield: (
    <>
      <path d="M12 3.5 19 6.2v5.2c0 4.6-3 7.7-7 9.1-4-1.4-7-4.5-7-9.1V6.2z" />
      <path d="m9 11.8 2.1 2.1 3.9-4.2" />
    </>
  ),
  sparkle: (
    <>
      <path d="M12 4.5 13.6 9l4.6 1.6-4.6 1.6L12 16.8l-1.6-4.6-4.6-1.6L10.4 9z" />
      <path d="M18.5 15.5v4M16.5 17.5h4" />
    </>
  ),
};

export type IconName = keyof typeof glyphs;

export function Icon({ name, size = 16 }: { name: IconName; size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.6}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {glyphs[name]}
    </svg>
  );
}
