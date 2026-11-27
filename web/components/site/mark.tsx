type IconProps = { className?: string; mono?: boolean };

// The ports mark: a grid of ports, one of which has become a name. It's
// drawn on a 13-unit grid (cells 3, gaps 2), so it lands on whole pixels
// at 13, 26 and 39 px. The name is blue unless `mono`.
export function PortsMark({ className, mono = false }: IconProps) {
  return (
    <svg viewBox="0 0 13 13" aria-hidden="true" className={className} fill="currentColor">
      <rect x="0" y="0" width="3" height="3" rx="0.8" />
      <rect x="5" y="0" width="3" height="3" rx="0.8" />
      <rect x="10" y="0" width="3" height="3" rx="0.8" />
      <rect x="0" y="5" width="3" height="3" rx="0.8" />
      <rect x="5" y="5" width="8" height="3" rx="1.5" className={mono ? undefined : "fill-blue"} />
      <rect x="0" y="10" width="3" height="3" rx="0.8" />
      <rect x="5" y="10" width="3" height="3" rx="0.8" />
      <rect x="10" y="10" width="3" height="3" rx="0.8" />
    </svg>
  );
}
