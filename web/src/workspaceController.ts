import { useCallback, useEffect, useMemo, useReducer, useRef, useState } from "react";
import type { MutableRefObject } from "react";
import {
  emptyWorkspace,
  legacyWorkspace,
  normalizeWorkspace,
  snapshotLabel,
  workspaceFromSearch,
  workspaceStorageKey,
  workspaceTopologyKey,
  workspaceURL,
  workspaceWithoutLayout,
} from "./workspace";
import type { WorkspaceState } from "./workspace";
import { loadRemoteWorkspace, remoteWorkspaceID, saveRemoteWorkspace } from "./workspaceRemote";

type WorkspaceAction = { type: "replace"; workspace: WorkspaceState };

function workspaceReducer(_state: WorkspaceState, action: WorkspaceAction): WorkspaceState {
  return action.type === "replace" ? action.workspace : _state;
}

function savedWorkspace(): WorkspaceState | undefined {
  try {
    const value = localStorage.getItem(workspaceStorageKey);
    return value ? normalizeWorkspace(JSON.parse(value)) : undefined;
  } catch { return undefined; }
}

function initialWorkspace(): WorkspaceState {
  const saved = savedWorkspace();
  const linked = workspaceFromSearch(window.location.search) ?? legacyWorkspace(window.location.search);
  if (!linked) return saved ?? emptyWorkspace();
  // Reuse exact geometry only for the same local panel topology. A shared link
  // from another arrangement or viewport receives a deterministic default.
  if (saved?.layout && workspaceTopologyKey(saved) === workspaceTopologyKey(linked)) return { ...linked, layout: saved.layout };
  return linked;
}

export type WorkspaceController = {
  workspace: WorkspaceState;
  workspaceRef: MutableRefObject<WorkspaceState>;
  breadcrumb: string;
  applyWorkspace: (next: WorkspaceState, snapshot?: boolean) => void;
  change: (update: (current: WorkspaceState) => WorkspaceState, snapshot?: boolean) => void;
  jump: (delta: number) => void;
};

export function useWorkspaceController(): WorkspaceController {
  const [workspace, dispatch] = useReducer(workspaceReducer, undefined, initialWorkspace);
  const workspaceRef = useRef(workspace);
  workspaceRef.current = workspace;
  const restoring = useRef(false);
  const jumpList = useRef<WorkspaceState[]>([workspace]);
  const jumpIndex = useRef(0);
  const pendingReplace = useRef<WorkspaceState | undefined>(undefined);
  const replaceFrame = useRef<number | undefined>(undefined);
  const remoteID = useMemo(() => remoteWorkspaceID(), []);
  const remoteRevision = useRef(-1);
  const remoteSerialized = useRef("");
  const [remoteReady, setRemoteReady] = useState(!remoteID);

  const writeURL = useCallback((next: WorkspaceState, push: boolean) => {
    try { localStorage.setItem(workspaceStorageKey, JSON.stringify(next)); } catch { /* persistence is optional */ }
    const state = { rlvizWorkspace: next };
    if (push) {
      if (replaceFrame.current !== undefined) cancelAnimationFrame(replaceFrame.current);
      replaceFrame.current = undefined;
      pendingReplace.current = undefined;
      window.history.pushState(state, "", workspaceURL(next));
      return;
    }
    pendingReplace.current = next;
    if (replaceFrame.current !== undefined) return;
    replaceFrame.current = requestAnimationFrame(() => {
      replaceFrame.current = undefined;
      const latest = pendingReplace.current;
      pendingReplace.current = undefined;
      if (latest) window.history.replaceState({ rlvizWorkspace: latest }, "", workspaceURL(latest));
    });
  }, []);

  const applyWorkspace = useCallback((next: WorkspaceState, snapshot = true) => {
    const normalized = normalizeWorkspace(next);
    if (!normalized || JSON.stringify(normalized) === JSON.stringify(workspaceRef.current)) return;
    workspaceRef.current = normalized;
    dispatch({ type: "replace", workspace: normalized });
    writeURL(normalized, snapshot && !restoring.current);
    if (snapshot && !restoring.current) {
      const serialized = JSON.stringify(normalized);
      const current = JSON.stringify(jumpList.current[jumpIndex.current]);
      if (serialized !== current) {
        jumpList.current = [...jumpList.current.slice(0, jumpIndex.current + 1), normalized];
        jumpIndex.current = jumpList.current.length - 1;
      }
    }
  }, [writeURL]);

  const change = useCallback((update: (current: WorkspaceState) => WorkspaceState, snapshot = true) => {
    applyWorkspace(update(workspaceRef.current), snapshot);
  }, [applyWorkspace]);

  const jump = useCallback((delta: number) => {
    const nextIndex = jumpIndex.current + delta;
    if (nextIndex < 0 || nextIndex >= jumpList.current.length) return;
    jumpIndex.current = nextIndex;
    restoring.current = true;
    applyWorkspace(jumpList.current[nextIndex], false);
    restoring.current = false;
  }, [applyWorkspace]);

  useEffect(() => {
    writeURL(workspaceRef.current, false);
    return () => {
      if (replaceFrame.current !== undefined) cancelAnimationFrame(replaceFrame.current);
    };
  }, [writeURL]);

  useEffect(() => {
    const onPop = (event: PopStateEvent) => {
      const next = normalizeWorkspace((event.state as { rlvizWorkspace?: unknown } | null)?.rlvizWorkspace)
        ?? workspaceFromSearch(location.search)
        ?? legacyWorkspace(location.search);
      if (!next) return;
      const serialized = JSON.stringify(next);
      for (let index = jumpList.current.length - 1; index >= 0; index--) {
        if (JSON.stringify(jumpList.current[index]) === serialized) {
          jumpIndex.current = index;
          break;
        }
      }
      restoring.current = true;
      workspaceRef.current = next;
      dispatch({ type: "replace", workspace: next });
      restoring.current = false;
    };
    window.addEventListener("popstate", onPop);
    return () => window.removeEventListener("popstate", onPop);
  }, []);

  useEffect(() => {
    if (!remoteID) return;
    const controller = new AbortController();
    const follow = async () => {
      while (!controller.signal.aborted) {
        try {
          const remote = await loadRemoteWorkspace(remoteID, remoteRevision.current, controller.signal);
          remoteRevision.current = remote.revision;
          remoteSerialized.current = JSON.stringify(workspaceWithoutLayout(remote.workspace));
          restoring.current = true;
          applyWorkspace(remote.workspace, false);
          restoring.current = false;
          setRemoteReady(true);
        } catch (error) {
          if (controller.signal.aborted) return;
          setRemoteReady(true);
          await new Promise((resolve) => window.setTimeout(resolve, 1000));
        }
      }
    };
    void follow();
    return () => controller.abort();
  }, [applyWorkspace, remoteID]);

  useEffect(() => {
    if (!remoteID || !remoteReady) return;
    const serialized = JSON.stringify(workspaceWithoutLayout(workspace));
    if (serialized === remoteSerialized.current) return;
    const controller = new AbortController();
    const timer = window.setTimeout(() => {
      void saveRemoteWorkspace(remoteID, workspace, controller.signal).then((remote) => {
        remoteRevision.current = remote.revision;
        remoteSerialized.current = JSON.stringify(workspaceWithoutLayout(remote.workspace));
      }).catch(() => undefined);
    }, 80);
    return () => {
      window.clearTimeout(timer);
      controller.abort();
    };
  }, [remoteID, remoteReady, workspace]);

  return {
    workspace,
    workspaceRef,
    breadcrumb: useMemo(() => snapshotLabel(workspace), [workspace]),
    applyWorkspace,
    change,
    jump,
  };
}
