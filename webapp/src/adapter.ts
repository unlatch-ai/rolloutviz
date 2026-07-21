import { decodeAdapter } from "./wasm";

interface AdapterExports extends WebAssembly.Exports {
  memory: WebAssembly.Memory;
  rlviz_alloc(size: number): number;
  rlviz_adapt(pointer: number, length: number): number;
  rlviz_result_len(): number;
  rlviz_free(pointer: number, length: number): void;
}

export async function adapterIdentity(bytes: Uint8Array): Promise<{ digest: string; size: number }> {
  const hash = await crypto.subtle.digest("SHA-256", bytes);
  return { digest: [...new Uint8Array(hash)].map((value) => value.toString(16).padStart(2, "0")).join(""), size: bytes.byteLength };
}

/** Execute a confirmed, import-free adapter module and validate its canonical output. */
export async function runAdapter(moduleBytes: Uint8Array, sourceBytes: Uint8Array, sourceName: string): Promise<string> {
  const { instance } = await WebAssembly.instantiate(moduleBytes, {});
  const exports = instance.exports as AdapterExports;
  for (const name of ["memory", "rlviz_alloc", "rlviz_adapt", "rlviz_result_len", "rlviz_free"] as const) {
    if (!(name in exports)) throw new Error(`Adapter is missing export ${name}`);
  }
  const inputPointer = exports.rlviz_alloc(sourceBytes.byteLength);
  new Uint8Array(exports.memory.buffer, inputPointer, sourceBytes.byteLength).set(sourceBytes);
  const outputPointer = exports.rlviz_adapt(inputPointer, sourceBytes.byteLength);
  const outputLength = exports.rlviz_result_len();
  if (outputPointer <= 0 || outputLength <= 0 || outputLength > 64 * 1024 * 1024) throw new Error("Adapter returned an invalid result buffer");
  const output = new Uint8Array(exports.memory.buffer, outputPointer, outputLength).slice();
  exports.rlviz_free(inputPointer, sourceBytes.byteLength);
  exports.rlviz_free(outputPointer, outputLength);
  return decodeAdapter(output, sourceName, sourceBytes.byteLength);
}
