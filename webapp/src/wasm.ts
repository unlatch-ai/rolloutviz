type WasmResult = { ok: true; value: unknown } | { ok: false; error: string };

declare global {
  class Go {
    importObject: WebAssembly.Imports;
    run(instance: WebAssembly.Instance): Promise<void>;
  }
  var rlvizWasmReady: boolean | undefined;
  var rlvizParse: ((bytes: Uint8Array, name: string) => string) | undefined;
  var rlvizDecodeAdapter: ((bytes: Uint8Array, name: string, sourceSize: number) => string) | undefined;
  var rlvizAnalyze: ((collection: string, trajectoryId: string) => string) | undefined;
  var rlvizCompare: ((collection: string, left: string, right: string) => string) | undefined;
}

let ready: Promise<void> | undefined;

export function loadCore(): Promise<void> {
  if (ready) return ready;
  ready = (async () => {
    if (typeof Go === "undefined") {
      await new Promise<void>((resolve, reject) => {
        const script = document.createElement("script");
        script.src = "/wasm_exec.js";
        script.onload = () => resolve();
        script.onerror = () => reject(new Error("Could not load the local Go WebAssembly runtime"));
        document.head.append(script);
      });
    }
    const go = new Go();
    const response = await fetch("/rlviz.wasm");
    if (!response.ok) throw new Error(`Could not load viewer core (${response.status})`);
    const bytes = await response.arrayBuffer();
    const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
    void go.run(instance);
    for (let attempt = 0; attempt < 100 && !globalThis.rlvizWasmReady; attempt++) {
      await new Promise((resolve) => setTimeout(resolve, 5));
    }
    if (!globalThis.rlvizWasmReady) throw new Error("Viewer core did not initialize");
  })();
  return ready;
}

function unwrap<T>(encoded: string): T {
  const result = JSON.parse(encoded) as WasmResult;
  if (!result.ok) throw new Error(result.error);
  return result.value as T;
}

export async function parseTrace(bytes: Uint8Array, name: string): Promise<string> {
  await loadCore();
  return unwrap<string>(globalThis.rlvizParse!(bytes, name));
}

export async function decodeAdapter(bytes: Uint8Array, name: string, sourceSize: number): Promise<string> {
  await loadCore();
  return unwrap<string>(globalThis.rlvizDecodeAdapter!(bytes, name, sourceSize));
}

export async function analyze(collection: string, trajectoryId: string): Promise<unknown> {
  await loadCore();
  return unwrap(globalThis.rlvizAnalyze!(collection, trajectoryId));
}

export async function compare(collection: string, left: string, right: string): Promise<unknown> {
  await loadCore();
  return unwrap(globalThis.rlvizCompare!(collection, left, right));
}
