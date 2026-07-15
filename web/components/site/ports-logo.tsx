export function PortsLogo({
  className,
  maskId = "ports-slash",
}: {
  className?: string;
  maskId?: string;
}) {
  return (
    <svg viewBox="0 0 100 100" fill="currentColor" aria-hidden="true" className={className}>
      <mask id={maskId}>
        <rect width="100" height="100" fill="white" />
        <line
          x1="12"
          y1="88"
          x2="88"
          y2="12"
          stroke="black"
          strokeWidth="14"
          strokeLinecap="square"
        />
      </mask>
      <circle
        cx="50"
        cy="50"
        r="35"
        stroke="currentColor"
        strokeWidth="14"
        fill="none"
        mask={`url(#${maskId})`}
      />
      <circle cx="38" cy="38" r="6" fill="currentColor" />
      <circle cx="62" cy="62" r="6" fill="currentColor" />
    </svg>
  );
}
