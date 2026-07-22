import { describe, expect, it } from "vitest";
import { collectionMetadataKey, normalizeViewerMetadata } from "./viewerMetadata";

describe("viewer presentation metadata", () => {
  it("normalizes bounded collection and trajectory labels", () => {
    const normalized = normalizeViewerMetadata({
      collections: { demo: { title: "  Checkout eval  ", description: " local notes " } },
      trajectories: { lane: { title: "Preferred run", description: "x".repeat(700) }, empty: { title: " " } },
    });
    expect(normalized.collections.demo).toEqual({ title: "Checkout eval", description: "local notes" });
    expect(normalized.trajectories.lane.description).toHaveLength(500);
    expect(normalized.trajectories.empty).toBeUndefined();
  });

  it("uses source identity rather than display order for collection metadata", () => {
    expect(collectionMetadataKey(["source-b", "source-a", "source-a"])).toBe("source-a|source-b");
  });
});
