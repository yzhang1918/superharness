import type { ComponentChildren, JSX } from "preact";
import { useEffect, useState } from "preact/hooks";

type Tone = "good" | "danger" | "warning" | "muted";

export function RailIcon(props: { page: "status" | "timeline" | "review" }) {
  switch (props.page) {
    case "status":
      return (
        <svg viewBox="0 0 16 16" aria-hidden="true">
          <path d="M3 3.5h10v3H3zM3 8.5h10v4H3z" fill="none" stroke="currentColor" stroke-width="1.2" />
        </svg>
      );
    case "timeline":
      return (
        <svg viewBox="0 0 16 16" aria-hidden="true">
          <path d="M3 4.5h3v3H3zM10 9.5h3v3h-3z" fill="none" stroke="currentColor" stroke-width="1.2" />
          <path d="M6 6h2.5v5H10" fill="none" stroke="currentColor" stroke-width="1.2" />
        </svg>
      );
    case "review":
      return (
        <svg viewBox="0 0 16 16" aria-hidden="true">
          <path d="M4 3.5h8v9H4z" fill="none" stroke="currentColor" stroke-width="1.2" />
          <path d="M6 6.5h4M6 9.5h3" fill="none" stroke="currentColor" stroke-width="1.2" />
        </svg>
      );
  }
}

export function MetricIcon(props: { kind: "node" | "actions" | "warnings" | "blockers" }) {
  switch (props.kind) {
    case "node":
      return (
        <svg viewBox="0 0 16 16" aria-hidden="true">
          <path d="M3 4.5h4v3H3zM9 8.5h4v3H9z" fill="none" stroke="currentColor" stroke-width="1.2" />
          <path d="M7 6h2v4h0" fill="none" stroke="currentColor" stroke-width="1.2" />
        </svg>
      );
    case "actions":
      return (
        <svg viewBox="0 0 16 16" aria-hidden="true">
          <path d="M3.5 8h8" fill="none" stroke="currentColor" stroke-width="1.2" />
          <path d="M8.5 5l3 3-3 3" fill="none" stroke="currentColor" stroke-width="1.2" />
        </svg>
      );
    case "warnings":
      return (
        <svg viewBox="0 0 16 16" aria-hidden="true">
          <path d="M8 3.2l4.8 8.6H3.2z" fill="none" stroke="currentColor" stroke-width="1.2" />
          <path d="M8 6.1v3.2M8 11.2h.01" fill="none" stroke="currentColor" stroke-width="1.2" />
        </svg>
      );
    case "blockers":
      return (
        <svg viewBox="0 0 16 16" aria-hidden="true">
          <path d="M4 4.5h8v7H4z" fill="none" stroke="currentColor" stroke-width="1.2" />
          <path d="M5.8 6.3l4.4 4.4M10.2 6.3l-4.4 4.4" fill="none" stroke="currentColor" stroke-width="1.2" />
        </svg>
      );
  }
}

export function TopbarMetric(props: {
  kind: "node" | "actions" | "warnings" | "blockers";
  label: string;
  value: string;
  tone?: Tone;
  onClick?: () => void;
}) {
  const { kind, label, value, tone = "muted", onClick } = props;
  const content = (
    <>
      <span class="topbar-metric-icon">
        <MetricIcon kind={kind} />
      </span>
      <span class="topbar-metric-copy">
        <span class="topbar-metric-label">{label}</span>
        <strong class="topbar-metric-value">{value}</strong>
      </span>
    </>
  );

  return onClick ? (
    <button type="button" class={`topbar-metric is-${tone}`} onClick={onClick}>
      {content}
    </button>
  ) : (
    <div class={`topbar-metric is-${tone}`}>{content}</div>
  );
}

export function WorkbenchFrame(props: {
  explorerLabel: string;
  explorerTitle: string;
  explorerCount?: string;
  pageTitle: string;
  detailLabel: string;
  loading?: boolean;
  storageKey: string;
  defaultExplorerWidth?: number;
  explorerContent: ComponentChildren;
  children: ComponentChildren;
}) {
  const {
    explorerLabel,
    explorerTitle,
    explorerCount,
    pageTitle,
    detailLabel,
    loading,
    storageKey,
    defaultExplorerWidth = 288,
    explorerContent,
    children,
  } = props;
  const widthStorageKey = `harness-ui:explorer-width:${storageKey}`;
  const minExplorerWidth = 220;
  const maxExplorerWidth = 420;
  const clampExplorerWidth = (nextWidth: number) => Math.min(maxExplorerWidth, Math.max(minExplorerWidth, nextWidth));
  const readStoredExplorerWidth = () => {
    if (typeof window === "undefined") {
      return defaultExplorerWidth;
    }
    try {
      const stored = window.localStorage.getItem(widthStorageKey);
      if (!stored) {
        return defaultExplorerWidth;
      }
      const parsed = Number.parseInt(stored, 10);
      if (Number.isNaN(parsed)) {
        return defaultExplorerWidth;
      }
      return clampExplorerWidth(parsed);
    } catch {
      return defaultExplorerWidth;
    }
  };
  const [explorerWidth, setExplorerWidth] = useState(() => {
    return readStoredExplorerWidth();
  });
  const [dragState, setDragState] = useState<{ pointerId: number; startX: number; startWidth: number } | null>(null);

  useEffect(() => {
    try {
      window.localStorage.setItem(widthStorageKey, String(explorerWidth));
    } catch {
      // Storage-denied environments should still render and resize normally.
    }
  }, [explorerWidth, widthStorageKey]);

  useEffect(() => {
    if (!dragState) return;

    const onPointerMove = (event: PointerEvent) => {
      if (event.pointerId !== dragState.pointerId) return;
      const nextWidth = dragState.startWidth + (event.clientX - dragState.startX);
      setExplorerWidth(clampExplorerWidth(nextWidth));
    };

    const stopDragging = () => {
      setDragState(null);
      document.body.classList.remove("is-resizing-workbench");
    };

    document.body.classList.add("is-resizing-workbench");
    window.addEventListener("pointermove", onPointerMove);
    window.addEventListener("pointerup", stopDragging);
    window.addEventListener("pointercancel", stopDragging);
    return () => {
      window.removeEventListener("pointermove", onPointerMove);
      window.removeEventListener("pointerup", stopDragging);
      window.removeEventListener("pointercancel", stopDragging);
      document.body.classList.remove("is-resizing-workbench");
    };
  }, [dragState]);

  const frameStyle: JSX.CSSProperties = {
    gridTemplateColumns: `${explorerWidth}px 8px minmax(0, 1fr)`,
  };

  return (
    <section class="workbench-page" style={frameStyle}>
      <aside class="workbench-explorer">
        <div class="workbench-explorer-header">
          <span class="sidebar-label">{explorerLabel}</span>
          <strong>{explorerTitle}</strong>
          {explorerCount ? <span class="workbench-explorer-count">{explorerCount}</span> : null}
        </div>
        <div class="workbench-explorer-body">{explorerContent}</div>
      </aside>

      <div
        class="workbench-divider"
        role="separator"
        aria-orientation="vertical"
        aria-label={`${explorerTitle} width`}
        aria-valuemin={minExplorerWidth}
        aria-valuemax={maxExplorerWidth}
        aria-valuenow={explorerWidth}
        tabIndex={0}
        onPointerDown={(event) => {
          if (event.pointerType === "mouse" && event.button !== 0) return;
          setDragState({ pointerId: event.pointerId, startX: event.clientX, startWidth: explorerWidth });
          event.preventDefault();
        }}
        onKeyDown={(event) => {
          if (event.key === "ArrowLeft") {
            setExplorerWidth((current) => clampExplorerWidth(current - 16));
            event.preventDefault();
            return;
          }
          if (event.key === "ArrowRight") {
            setExplorerWidth((current) => clampExplorerWidth(current + 16));
            event.preventDefault();
          }
        }}
      />

      <section class="workbench-inspector">
        <header class="workbench-header">
          <div class="workbench-header-title">
            <div class="editor-tab is-active">{pageTitle}</div>
            <div class="editor-section-label">{detailLabel}</div>
          </div>
          {loading ? <span class="muted">loading</span> : null}
        </header>
        <div class="workbench-body">{children}</div>
      </section>
    </section>
  );
}

export function ExplorerList(props: { ariaLabel: string; children: ComponentChildren }) {
  return (
    <div class="explorer-list" aria-label={props.ariaLabel}>
      {props.children}
    </div>
  );
}

export function ExplorerItem(props: {
  selected: boolean;
  onSelect: () => void;
  ariaLabel?: string;
  meta?: string;
  subtitle?: string;
  tone?: Tone;
  title: ComponentChildren;
  trailing?: ComponentChildren;
}) {
  const { selected, onSelect, ariaLabel, meta, subtitle, tone = "muted", title, trailing } = props;
  return (
    <button
      type="button"
      class={`explorer-item is-${tone}${selected ? " is-active" : ""}`}
      onClick={onSelect}
      aria-pressed={selected}
      aria-label={ariaLabel}
    >
      <div class="explorer-item-main">
        <div class="explorer-item-row">
          <div class="explorer-item-title">{title}</div>
          {trailing ? <div class="explorer-item-trailing">{trailing}</div> : null}
        </div>
        {subtitle ? <div class="explorer-item-subtitle">{subtitle}</div> : null}
      </div>
      {meta ? <div class="explorer-item-meta">{meta}</div> : null}
    </button>
  );
}

export function EmptyState(props: { children: ComponentChildren }) {
  return <div class="empty-row">{props.children}</div>;
}

export function Notice(props: { tone: "warning" | "error"; children: ComponentChildren }) {
  return <div class={`notice notice-${props.tone}`}>{props.children}</div>;
}

export function InspectorTabs(props: { ariaLabel: string; children: ComponentChildren }) {
  return (
    <div class="inspector-tabs" role="tablist" aria-label={props.ariaLabel}>
      {props.children}
    </div>
  );
}

export function InspectorTab(props: {
  selected: boolean;
  onSelect: () => void;
  children: ComponentChildren;
}) {
  return (
    <button
      type="button"
      class={`inspector-tab${props.selected ? " is-active" : ""}`}
      onClick={props.onSelect}
      role="tab"
      aria-selected={props.selected}
    >
      {props.children}
    </button>
  );
}

export function InspectorHeader(props: {
  title: ComponentChildren;
  subtitle?: ComponentChildren;
  meta?: ComponentChildren;
}) {
  return (
    <div class="inspector-head">
      <div>
        <div class="inspector-title">{props.title}</div>
        {props.subtitle ? <div class="inspector-subtitle">{props.subtitle}</div> : null}
      </div>
      {props.meta ? <div class="inspector-meta">{props.meta}</div> : null}
    </div>
  );
}

export function StatusBadge(props: { tone: Tone; children: ComponentChildren }) {
  return <span class={`status-badge is-${props.tone}`}>{props.children}</span>;
}
