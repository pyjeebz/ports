"use client";

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <main className="flex min-h-dvh flex-col items-center justify-center px-6 text-center">
      <p className="font-mono text-xs uppercase tracking-[0.18em] text-faint">
        500 · signal lost
      </p>
      <h1 className="mt-3 font-mono text-3xl">The sweep cut out</h1>
      <p className="mt-3 max-w-md text-sm text-muted">
        Something failed on this page{error.digest ? ` (${error.digest})` : ""}
        . You can retry the request.
      </p>
      <button
        onClick={reset}
        className="mt-8 font-mono text-sm text-accent underline-offset-4 hover:underline"
      >
        → sweep again
      </button>
    </main>
  );
}