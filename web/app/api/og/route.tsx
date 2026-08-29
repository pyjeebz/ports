import { makeOgImage } from "@/lib/og";

// Per-doc OG cards render here because Next refuses an opengraph-image
// route inside the /docs/[[...slug]] catch-all. The docs page's
// generateMetadata points openGraph.images at this route.
export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const title = searchParams.get("title")?.slice(0, 120) || "Docs";
  // ~2 wrapped lines at card size before the layout crowds
  const description =
    searchParams.get("desc")?.slice(0, 100) || "Every port, by project.";
  const label = searchParams.get("label")?.slice(0, 40) || "ports · docs";

  return makeOgImage({
    label,
    lines: [title, description],
    // never echo the description back in the footnote
    footnote: "One static binary. Zero config.",
  });
}