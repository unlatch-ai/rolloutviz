export type StructuredDiffStatus = "equal" | "changed" | "left-only" | "right-only";

export type StructuredDiffValue = {
  present: boolean;
  type: "missing" | "null" | "boolean" | "number" | "string" | "array" | "object" | "unsupported";
  preview: string;
  truncated?: boolean;
};

export type StructuredDiffRow = {
  path: string;
  depth: number;
  status: StructuredDiffStatus;
  left: StructuredDiffValue;
  right: StructuredDiffValue;
};

export type StructuredDiffOptions = {
  maxDepth?: number;
  maxRows?: number;
  maxStringLength?: number;
  maxArrayItems?: number;
  maxObjectKeys?: number;
  rootPath?: string;
};

export type StructuredDiffResult = {
  rows: StructuredDiffRow[];
  truncated: boolean;
  truncationReasons: Array<"depth" | "rows" | "array-items" | "object-keys" | "cycle">;
};

type Limits = Required<Omit<StructuredDiffOptions, "rootPath">> & { rootPath: string };
type Side = { present: boolean; value?: unknown; accessor?: boolean };

const DEFAULTS: Limits = {
  maxDepth: 6,
  maxRows: 200,
  maxStringLength: 240,
  maxArrayItems: 50,
  maxObjectKeys: 100,
  rootPath: "$",
};

const identifier = /^[A-Za-z_$][A-Za-z0-9_$]*$/;

function boundedInteger(value: number | undefined, fallback: number, minimum: number): number {
  if (value === undefined || !Number.isFinite(value)) return fallback;
  return Math.max(minimum, Math.floor(value));
}

function limits(options: StructuredDiffOptions): Limits {
  return {
    maxDepth: boundedInteger(options.maxDepth, DEFAULTS.maxDepth, 0),
    maxRows: boundedInteger(options.maxRows, DEFAULTS.maxRows, 1),
    maxStringLength: boundedInteger(options.maxStringLength, DEFAULTS.maxStringLength, 0),
    maxArrayItems: boundedInteger(options.maxArrayItems, DEFAULTS.maxArrayItems, 0),
    maxObjectKeys: boundedInteger(options.maxObjectKeys, DEFAULTS.maxObjectKeys, 0),
    rootPath: options.rootPath || DEFAULTS.rootPath,
  };
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  if (value === null || typeof value !== "object" || Array.isArray(value)) return false;
  const prototype = Object.getPrototypeOf(value);
  return prototype === Object.prototype || prototype === null;
}

function valueType(side: Side): StructuredDiffValue["type"] {
  if (!side.present) return "missing";
  if (side.accessor) return "unsupported";
  if (side.value === null) return "null";
  if (Array.isArray(side.value)) return "array";
  if (isPlainObject(side.value)) return "object";
  if (["boolean", "number", "string"].includes(typeof side.value)) return typeof side.value as "boolean" | "number" | "string";
  return "unsupported";
}

function preview(side: Side, maxStringLength: number): StructuredDiffValue {
  const type = valueType(side);
  if (type === "missing") return { present: false, type, preview: "—" };
  if (type === "unsupported") {
    const label = side.accessor
      ? "accessor"
      : side.value === undefined
        ? "undefined"
        : typeof side.value === "object"
          ? "unsupported object"
          : typeof side.value;
    return { present: true, type, preview: `[${label}]` };
  }
  if (type === "null") return { present: true, type, preview: "null" };
  if (type === "array") return { present: true, type, preview: `[${(side.value as unknown[]).length} items]` };
  if (type === "object") return { present: true, type, preview: `{${Object.keys(side.value as object).length} fields}` };
  if (type === "string") {
    const value = side.value as string;
    const clipped = value.length > maxStringLength;
    return { present: true, type, preview: clipped ? `${value.slice(0, maxStringLength)}…` : value, ...(clipped ? { truncated: true } : {}) };
  }
  if (type === "number") {
    const number = side.value as number;
    return { present: true, type, preview: Number.isFinite(number) ? String(number) : `[${String(number)}]` };
  }
  return { present: true, type, preview: String(side.value) };
}

function childPath(parent: string, key: string): string {
  return identifier.test(key) ? `${parent}.${key}` : `${parent}[${JSON.stringify(key)}]`;
}

function ownSide(parent: Side, key: string): Side {
  if (!parent.present || parent.value === null || typeof parent.value !== "object") return { present: false };
  const descriptor = Object.getOwnPropertyDescriptor(parent.value, key);
  if (!descriptor) return { present: false };
  if (!("value" in descriptor)) return { present: true, accessor: true };
  return { present: true, value: descriptor.value };
}

function scalarEqual(left: Side, right: Side): boolean {
  return left.present === right.present && (!left.present || (!left.accessor && !right.accessor && Object.is(left.value, right.value)));
}

/**
 * Produces display-only rows from JSON-like values. It never evaluates strings,
 * invokes accessors, or attempts semantic array alignment.
 */
export function diffStructuredJson(leftValue: unknown, rightValue: unknown, options: StructuredDiffOptions = {}): StructuredDiffResult {
  const config = limits(options);
  const rows: StructuredDiffRow[] = [];
  const reasons = new Set<StructuredDiffResult["truncationReasons"][number]>();
  const visited = new WeakMap<object, WeakSet<object>>();

  const add = (path: string, depth: number, status: StructuredDiffStatus, left: Side, right: Side) => {
    if (rows.length >= config.maxRows) {
      reasons.add("rows");
      return false;
    }
    rows.push({ path, depth, status, left: preview(left, config.maxStringLength), right: preview(right, config.maxStringLength) });
    return true;
  };

  const walk = (left: Side, right: Side, path: string, depth: number): void => {
    if (rows.length >= config.maxRows) {
      reasons.add("rows");
      return;
    }
    if (!left.present || !right.present) {
      add(path, depth, left.present ? "left-only" : "right-only", left, right);
      return;
    }

    const leftType = valueType(left);
    const rightType = valueType(right);
    const bothContainers = leftType === rightType && (leftType === "array" || leftType === "object");
    if (!bothContainers) {
      add(path, depth, leftType === rightType && scalarEqual(left, right) ? "equal" : "changed", left, right);
      return;
    }

    if (left.value === right.value) {
      add(path, depth, "equal", left, right);
      return;
    }
    if (depth >= config.maxDepth) {
      reasons.add("depth");
      add(path, depth, "changed", left, right);
      return;
    }

    const leftObject = left.value as object;
    const rightObject = right.value as object;
    const seenRights = visited.get(leftObject);
    if (seenRights?.has(rightObject)) {
      reasons.add("cycle");
      add(path, depth, "changed", left, right);
      return;
    }
    if (seenRights) seenRights.add(rightObject);
    else visited.set(leftObject, new WeakSet([rightObject]));

    if (leftType === "array") {
      const maximumLength = Math.max((left.value as unknown[]).length, (right.value as unknown[]).length);
      const count = Math.min(maximumLength, config.maxArrayItems);
      if (maximumLength > count) reasons.add("array-items");
      if (maximumLength === 0) add(path, depth, "equal", left, right);
      let processed = 0;
      for (let index = 0; index < count && rows.length < config.maxRows; index += 1) {
        walk(ownSide(left, String(index)), ownSide(right, String(index)), `${path}[${index}]`, depth + 1);
        processed += 1;
      }
      if (processed < count) reasons.add("rows");
      return;
    }

    const keys = [...new Set([...Object.keys(left.value as object), ...Object.keys(right.value as object)])].sort();
    const selectedKeys = keys.slice(0, config.maxObjectKeys);
    if (keys.length > selectedKeys.length) reasons.add("object-keys");
    if (keys.length === 0) add(path, depth, "equal", left, right);
    for (const key of selectedKeys) {
      if (rows.length >= config.maxRows) {
        reasons.add("rows");
        break;
      }
      walk(ownSide(left, key), ownSide(right, key), childPath(path, key), depth + 1);
    }
  };

  walk({ present: true, value: leftValue }, { present: true, value: rightValue }, config.rootPath, 0);
  return { rows, truncated: reasons.size > 0, truncationReasons: [...reasons].sort() };
}

export function diffToolArguments(left: unknown, right: unknown, options: StructuredDiffOptions = {}): StructuredDiffResult {
  return diffStructuredJson(left, right, { ...options, rootPath: options.rootPath ?? "$.arguments" });
}

export function diffToolResults(left: unknown, right: unknown, options: StructuredDiffOptions = {}): StructuredDiffResult {
  return diffStructuredJson(left, right, { ...options, rootPath: options.rootPath ?? "$.result" });
}
