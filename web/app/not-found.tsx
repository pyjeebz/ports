import Link from "next/link";

// The site's 404, worded like the router's own page for a name it
// doesn't know.
export default function NotFound() {
  return (
    <main className="mx-auto flex min-h-dvh max-w-[560px] flex-col justify-center px-6">
      <p className="text-sm text-muted-foreground">404</p>
      <h1 className="mt-2 text-2xl font-semibold tracking-[-0.015em]">Nothing is called that yet</h1>
      <p className="mt-3 text-[15px] leading-7 text-muted-foreground">
        There&apos;s no page at this address. It may have moved, or never existed.
      </p>
      <Link href="/" className="mt-6 text-sm underline underline-offset-4 hover:text-muted-foreground">
        Back to ports.tools
      </Link>
    </main>
  );
}
