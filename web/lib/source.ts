import { loader } from "fumadocs-core/source";

// Relative import: the `@/` alias fails to resolve into the dot-prefixed
// `.source` output directory.
import { docs } from "../.source/server";

export const source = loader({
  baseUrl: "/docs",
  source: docs.toFumadocsSource(),
});