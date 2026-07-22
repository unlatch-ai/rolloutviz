export type ViewerSetup = {
  mode: "browser" | "cli";
  status?: string;
  samples?: Array<{ label: string; value: string }>;
  selectedSample?: string;
  onSample?: (value: string) => void;
  onOpenTrace?: () => void;
  onOpenAdapter?: () => void;
  onAdapterHelp?: () => void;
};

export function Settings({ active, theme, setup, onTheme, onActivate, onClose }: {
  active: boolean;
  theme: "light" | "dark";
  setup: ViewerSetup;
  onTheme: (theme: "light" | "dark") => void;
  onActivate: () => void;
  onClose: () => void;
}) {
  return <section className={`workspace-settings ${active ? "active-zone" : ""}`} tabIndex={0} onFocus={onActivate} onPointerDown={onActivate} aria-label="RLViz settings">
    <header><span>viewer settings</span><button onClick={onClose}>close</button></header>
    <div className="settings-copy">
      <section><h2>Appearance</h2><div className="settings-options" role="group" aria-label="Color theme"><button aria-pressed={theme === "light"} onClick={() => onTheme("light")}>Light</button><button aria-pressed={theme === "dark"} onClick={() => onTheme("dark")}>Dark</button></div></section>
      <section><h2>Open data</h2>{setup.mode === "browser" ? <>
        <p>Files are parsed in this tab. Trace bytes are not uploaded or persisted.</p>
        {setup.samples?.length && <label>Example data<select aria-label="Example data" value={setup.selectedSample} onChange={(event) => setup.onSample?.(event.target.value)}>{setup.samples.map((sample) => <option value={sample.value} key={sample.value}>{sample.label}</option>)}</select></label>}
        <div className="settings-actions"><button onClick={setup.onOpenTrace}>Open local trace</button><button onClick={setup.onOpenAdapter}>Upload WASM adapter</button><button onClick={setup.onAdapterHelp}>Adapter instructions</button></div>
      </> : <>
        <p>The CLI indexes source files read-only and opens this browser workspace. Use <code>rlviz open PATH</code>, or ask a coding agent to inspect and open the trajectories you need.</p>
        <pre>rlviz trajectories PATH --json{`\n`}rlviz workspace open PATH --trajectory ID --json</pre>
      </>}</section>
      {setup.status && <section><h2>Status</h2><p role="status">{setup.status}</p></section>}
      <section><h2>Unsupported formats</h2><p>Ask a coding agent to inspect the source first. In the browser, have it build a sandboxed WASM adapter and upload it here. With the CLI, have it scaffold a local plugin, show you the executable files, and ask before trusting or running them.</p></section>
    </div>
  </section>;
}
