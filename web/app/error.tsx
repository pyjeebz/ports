"use client";

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <main className="mx-auto flex min-h-dvh max-w-[560px] flex-col justify-center px-6">
      <p className="text-sm text-muted-foreground">500</p>
      <h1 className="mt-2 text-2xl font-semibold tracking-[-0.015em]">This page isn&apos;t answering</h1>
      <p className="mt-3 text-[15px] leading-7 text-muted-foreground">
        Something failed while loading it{error.digest ? ` (${error.digest})` : ""}. Try again.
      </p>
      <button
        type="button"
        onClick={reset}
        className="mt-6 w-fit rounded-md border px-3 py-1.5 text-sm transition-[transform,background-color] duration-150 ease-out hover:bg-accent active:scale-[0.97]"
      >
        Try again
      </button>
    </main>
  );
}
