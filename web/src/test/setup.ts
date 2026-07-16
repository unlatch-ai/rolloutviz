import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

afterEach(cleanup);

Object.defineProperty(Element.prototype, "scrollIntoView", { value: () => {}, writable: true });
globalThis.requestAnimationFrame = (callback: FrameRequestCallback) => { callback(0); return 0; };
