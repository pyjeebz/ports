import Link from "next/link";

export default function NotFound() {
  return (
    <main className="flex min-h-dvh flex-col items-center justify-center px-6 text-center">
      <p className="font-mono text-xs uppercase tracking-[0.18em] text-faint">
        404 · no listener
      </p>
      <h1 className="mt-3 font-mono text-3xl">Contact not found</h1>
      <p className="mt-3 max-w-md text-sm text-muted">
        Nothing is listening at this URL. Check the address, or head back to
        the radar.
      </p>
      <Link
        href="/"
        className="mt-8 font-mono text-sm text-accent underline-offset-4 hover:underline"
      >
        ← back to ports.tools
      </Link>
    </main>
  );
}