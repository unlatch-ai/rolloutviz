import { describe, expect, it } from "vitest";
import { diffStructuredJson, diffToolArguments, diffToolResults } from "./structuredDiff";

describe("structured JSON diff", () => {
  it("sorts object paths deterministically and reports all statuses", () => {
    const result = diffStructuredJson(
      { z: 1, same: true, removed: "left", nested: { value: 2 }, "odd.key": null },
      { added: "right", same: true, z: 3, nested: { value: 2 }, "odd.key": false },
    );

    expect(result.rows.map(({ path, status }) => [path, status])).toEqual([
      ["$.added", "right-only"],
      ["$.nested.value", "equal"],
      ["$[\"odd.key\"]", "changed"],
      ["$.removed", "left-only"],
      ["$.same", "equal"],
      ["$.z", "changed"],
    ]);
    expect(result.truncated).toBe(false);
  });

  it("compares arrays by index without guessing semantic alignment", () => {
    const result = diffStructuredJson(["a", "b"], ["a", "inserted", "b"]);
    expect(result.rows.map(({ path, status }) => [path, status])).toEqual([
      ["$[0]", "equal"],
      ["$[1]", "changed"],
      ["$[2]", "right-only"],
    ]);
  });

  it("bounds rows, depth, object keys, arrays, and string previews", () => {
    const rows = diffStructuredJson(
      Object.fromEntries(Array.from({ length: 20 }, (_, index) => [`field-${index}`, "l".repeat(100)])),
      Object.fromEntries(Array.from({ length: 20 }, (_, index) => [`field-${index}`, "r".repeat(100)])),
      { maxRows: 3, maxObjectKeys: 8, maxStringLength: 5 },
    );
    expect(rows.rows).toHaveLength(3);
    expect(rows.rows[0].left).toMatchObject({ preview: "lllll…", truncated: true });
    expect(rows.truncationReasons).toEqual(["object-keys", "rows"]);

    const depth = diffStructuredJson({ a: { b: 1 } }, { a: { b: 1 } }, { maxDepth: 1 });
    expect(depth.rows).toMatchObject([{ path: "$.a", status: "changed" }]);
    expect(depth.truncationReasons).toContain("depth");

    const array = diffStructuredJson([1, 2, 3], [1, 2, 4], { maxArrayItems: 2 });
    expect(array.rows.map(({ path }) => path)).toEqual(["$[0]", "$[1]"]);
    expect(array.truncationReasons).toContain("array-items");

    const arrayRows = diffStructuredJson([1, 2, 3], [4, 5, 6], { maxRows: 1 });
    expect(arrayRows.rows).toHaveLength(1);
    expect(arrayRows.truncationReasons).toContain("rows");
  });

  it("does not invoke accessors or recursively inspect unsupported values", () => {
    let reads = 0;
    const left = Object.defineProperty({}, "danger", { enumerable: true, get: () => { reads += 1; return { executable: true }; } });
    const right = Object.defineProperty({}, "danger", { enumerable: true, get: () => { reads += 1; return { executable: false }; } });

    const result = diffStructuredJson(left, right);
    expect(reads).toBe(0);
    expect(result.rows).toEqual([{
      path: "$.danger",
      depth: 1,
      status: "changed",
      left: { present: true, type: "unsupported", preview: "[accessor]" },
      right: { present: true, type: "unsupported", preview: "[accessor]" },
    }]);

    expect(diffStructuredJson(new Date(0), new Date(0)).rows[0]).toMatchObject({ path: "$", status: "changed", left: { type: "unsupported" } });
  });

  it("stops safely at cycles", () => {
    const left: Record<string, unknown> = {};
    const right: Record<string, unknown> = {};
    left.self = left;
    right.self = right;
    const result = diffStructuredJson(left, right);
    expect(result.rows).toMatchObject([{ path: "$.self", status: "changed" }]);
    expect(result.truncationReasons).toContain("cycle");
  });

  it("exposes separately rooted helpers for tool arguments and results", () => {
    expect(diffToolArguments({ query: "a" }, { query: "b" }).rows[0].path).toBe("$.arguments.query");
    expect(diffToolResults({ ok: true }, { ok: false }).rows[0].path).toBe("$.result.ok");
    expect(diffToolResults(1, 2, { rootPath: "$.output" }).rows[0].path).toBe("$.output");
  });
});
