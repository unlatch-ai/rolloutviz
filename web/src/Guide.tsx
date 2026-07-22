import { useState } from "react";
import overviewMarkdown from "../../internal/guide/pages/overview.md?raw";
import installMarkdown from "../../internal/guide/pages/install.md?raw";
import workspaceMarkdown from "../../internal/guide/pages/workspace.md?raw";
import formatsMarkdown from "../../internal/guide/pages/formats.md?raw";
import agentsMarkdown from "../../internal/guide/pages/agents.md?raw";
import privacyMarkdown from "../../internal/guide/pages/privacy.md?raw";

type Block = { kind: "h1" | "h2" | "p" | "li" | "code"; text: string };
const pages = [
  { id: "overview", label: "Overview", markdown: overviewMarkdown },
  { id: "install", label: "Install", markdown: installMarkdown },
  { id: "workspace", label: "Workspace", markdown: workspaceMarkdown },
  { id: "formats", label: "Formats", markdown: formatsMarkdown },
  { id: "agents", label: "Agents", markdown: agentsMarkdown },
  { id: "privacy", label: "Privacy", markdown: privacyMarkdown },
] as const;

function blocks(markdown: string): Block[] {
  const result: Block[] = [];
  let code: string[] | undefined;
  markdown.split("\n").forEach((line) => {
    const text = line.trim();
    if (text.startsWith("```")) {
      if (code) { result.push({ kind: "code", text: code.join("\n") }); code = undefined; }
      else code = [];
      return;
    }
    if (code) { code.push(line); return; }
    if (!text) return;
    if (text.startsWith("# ")) result.push({ kind: "h1", text: text.slice(2) });
    else if (text.startsWith("## ")) result.push({ kind: "h2", text: text.slice(3) });
    else if (text.startsWith("- ")) result.push({ kind: "li", text: text.slice(2) });
    else result.push({ kind: "p", text });
  });
  return result;
}

function Inline({ text }: { text: string }) {
  const parts = text.split(/(`[^`]+`|\*\*[^*]+\*\*)/g);
  return <>{parts.map((part, index) => part.startsWith("`") ? <code key={index}>{part.slice(1, -1)}</code> : part.startsWith("**") ? <strong key={index}>{part.slice(2, -2)}</strong> : part)}</>;
}

export function Guide({ active, mode, shortcuts, onActivate, onClose }: { active: boolean; mode: "browser" | "cli"; shortcuts: Array<{ binding: string; label: string }>; onActivate: () => void; onClose: () => void }) {
  const [pageID, setPageID] = useState(mode === "browser" ? "install" : "workspace");
  const page = pages.find((candidate) => candidate.id === pageID) ?? pages[0];
  return <article className={`workspace-guide ${active ? "active-zone" : ""}`} tabIndex={0} onFocus={onActivate} onPointerDown={onActivate} aria-label="RLViz guide">
    <header><span>Guide · same source as public docs and <code>rlviz guide</code></span><button onClick={onClose}>close</button></header>
    <div className="guide-layout"><nav aria-label="Guide sections">{pages.map((candidate) => <button key={candidate.id} aria-current={candidate.id === page.id ? "page" : undefined} onClick={() => setPageID(candidate.id)}>{candidate.label}</button>)}</nav>
      <div className="guide-copy"><section className="guide-shortcuts" aria-label="Active module shortcuts"><h2>Active module shortcuts</h2><div>{shortcuts.map((shortcut) => <span key={`${shortcut.binding}:${shortcut.label}`}><kbd>{shortcut.binding}</kbd>{shortcut.label}</span>)}</div></section>{blocks(page.markdown).map((block, index) => {
        if (block.kind === "h1") return <h1 key={index}>{block.text}</h1>;
        if (block.kind === "h2") return <h2 key={index}>{block.text}</h2>;
        if (block.kind === "li") return <div className="guide-item" key={index}><span>•</span><p><Inline text={block.text} /></p></div>;
        if (block.kind === "code") return <pre key={index}><code>{block.text}</code></pre>;
        return <p key={index}><Inline text={block.text} /></p>;
      })}</div>
    </div>
  </article>;
}
