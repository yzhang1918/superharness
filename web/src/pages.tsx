import MarkdownIt from "markdown-it";
import { useEffect, useMemo, useRef, useState } from "preact/hooks";

import {
  buildTimelineTabs,
  formatTimestamp,
  formatValue,
  humanizeLabel,
  pickDefaultTimelineEvent,
  reviewAggregateFindingSource,
  reviewArtifactKey,
  reviewArtifactLabel,
  reviewArtifactText,
  reviewCountLabel,
  reviewFindingBadgeTone,
  reviewFindingKey,
  reviewReviewerLabel,
  reviewReviewerStatusLabel,
  reviewReviewerStatusTone,
  reviewRoundAriaLabel,
  reviewRoundCompactMeta,
  reviewRoundCompactStatusLabel,
  reviewRoundListLabel,
  reviewRoundStatusLabel,
  reviewRoundStatusTone,
  reviewRoundSubtitle,
  reviewRoundTitle,
  sortTimelineEvents,
  timelineEventSubtitle,
  timelineEventTitle,
  timelineTabText,
} from "./helpers";
import type {
  ErrorDetail,
  NextAction,
  PlanDocument,
  PlanHeading,
  PlanNode,
  ReviewAggregateFinding,
  ReviewArtifact,
  ReviewFinding,
  ReviewRound,
  ReviewReviewer,
  TimelineEvent,
} from "./types";
import {
  EmptyState,
  ExplorerItem,
  ExplorerList,
  InspectorHeader,
  InspectorTab,
  InspectorTabs,
  Notice,
  StatusBadge,
  WorkbenchFrame,
} from "./workbench";

const markdownRenderer = new MarkdownIt({
  html: false,
  linkify: true,
});

function ReviewFindingCard(props: { finding: ReviewFinding; provenance?: string | null }) {
  const { finding, provenance } = props;
  return (
    <article class="review-finding">
      <div class="review-finding-head">
        <strong>{finding.title}</strong>
        <StatusBadge tone={reviewFindingBadgeTone(finding.severity)}>{humanizeLabel(finding.severity)}</StatusBadge>
      </div>
      {provenance ? <div class="review-finding-meta">from {provenance}</div> : null}
      <p>{finding.details}</p>
      {Array.isArray(finding.locations) && finding.locations.length > 0 ? <div class="review-finding-locations">{finding.locations.join("\n")}</div> : null}
    </article>
  );
}

function StatusOverviewMetrics(props: {
  currentNode: string;
  nextActionCount: number;
  warningCount: number;
  blockerCount: number;
}) {
  return (
    <section class="summary-metrics" aria-label="Status overview">
      <div class="summary-metric">
        <span class="label">Current node</span>
        <strong>{props.currentNode}</strong>
      </div>
      <div class="summary-metric">
        <span class="label">Next actions</span>
        <strong>{props.nextActionCount}</strong>
      </div>
      <div class="summary-metric">
        <span class="label">Warnings</span>
        <strong>{props.warningCount}</strong>
      </div>
      <div class="summary-metric">
        <span class="label">Blockers</span>
        <strong>{props.blockerCount}</strong>
      </div>
    </section>
  );
}

export function StatusWorkspace(props: {
  loading: boolean;
  error: string | null;
  summary: string;
  currentNode: string;
  nextActions: NextAction[];
  blockers: ErrorDetail[];
  warnings: string[];
  errors: ErrorDetail[];
  facts: Array<[string, unknown]>;
  artifacts: Array<[string, unknown]>;
  selectedSection: string;
  onSelectSection: (section: string) => void;
}) {
  const { loading, error, summary, currentNode, nextActions, blockers, warnings, errors, facts, artifacts, selectedSection, onSelectSection } = props;
  const sections = [
    { id: "summary", label: "Summary" },
    { id: "next-actions", label: "Next actions", meta: String(nextActions.length) },
    { id: "warnings", label: "Warnings", meta: String(warnings.length + blockers.length + errors.length) },
    { id: "facts", label: "Facts", meta: String(facts.length) },
    { id: "artifacts", label: "Artifacts", meta: String(artifacts.length) },
  ];
  const activeSectionLabel = sections.find((item) => item.id === selectedSection)?.label ?? "Summary";

  let inspectorTitle = "Summary";
  let inspectorSubtitle = "Workflow overview";
  let inspectorBody = (
    <div class="inspector-panel">
      <StatusOverviewMetrics
        currentNode={currentNode}
        nextActionCount={nextActions.length}
        warningCount={warnings.length}
        blockerCount={blockers.length}
      />
      <section class="content-section">
        <div class="section-head">
          <h2>Summary</h2>
        </div>
        <p class="detail-copy">{summary}</p>
      </section>
    </div>
  );

  if (selectedSection === "next-actions") {
    inspectorTitle = "Next actions";
    inspectorSubtitle = `${nextActions.length} action(s) surfaced`;
    inspectorBody = (
      <section class="content-section">
        <div class="section-head">
          <h2>Next actions</h2>
          <span class="muted">{nextActions.length}</span>
        </div>
        <ol class="stack-list">
          {nextActions.length > 0 ? (
            nextActions.map((action, index) => (
              <li key={`${action.description}-${index}`}>
                <div class="list-title">{action.description}</div>
                {action.command ? <code>{action.command}</code> : <span class="muted">No command available</span>}
              </li>
            ))
          ) : (
            <EmptyState>No next actions surfaced yet.</EmptyState>
          )}
        </ol>
      </section>
    );
  }

  if (selectedSection === "warnings") {
    inspectorTitle = "Warnings";
    inspectorSubtitle = "Warnings, blockers, and surfaced errors";
    inspectorBody = (
      <section class="content-section">
        <div class="section-head">
          <h2>Warnings and blockers</h2>
        </div>
        <div class="warning-stack">
          {warnings.length > 0 ? warnings.map((warning, index) => <div key={`warning-${index}`} class="warning-item is-warning">{warning}</div>) : null}
          {blockers.length > 0
            ? blockers.map((blocker, index) => (
                <div key={`${blocker.path}-${index}`} class="warning-item is-blocker">
                  <strong>{blocker.path}</strong>
                  <span>{blocker.message}</span>
                </div>
              ))
            : null}
          {errors.length > 0
            ? errors.map((item, index) => (
                <div key={`${item.path}-${index}`} class="warning-item is-blocker">
                  <strong>{item.path}</strong>
                  <span>{item.message}</span>
                </div>
              ))
            : null}
          {warnings.length === 0 && blockers.length === 0 && errors.length === 0 ? <EmptyState>No warnings or blockers.</EmptyState> : null}
        </div>
      </section>
    );
  }

  if (selectedSection === "facts") {
    inspectorTitle = "Facts";
    inspectorSubtitle = `${facts.length} fact value(s)`;
    inspectorBody = (
      <section class="content-section">
        <div class="section-head">
          <h2>Facts</h2>
          <span class="muted">{facts.length}</span>
        </div>
        {facts.length > 0 ? (
          <dl class="kv-list">
            {facts.map(([key, value]) => (
              <div key={key}>
                <dt>{key}</dt>
                <dd>{formatValue(value)}</dd>
              </div>
            ))}
          </dl>
        ) : (
          <EmptyState>No facts available.</EmptyState>
        )}
      </section>
    );
  }

  if (selectedSection === "artifacts") {
    inspectorTitle = "Artifacts";
    inspectorSubtitle = `${artifacts.length} artifact reference(s)`;
    inspectorBody = (
      <section class="content-section">
        <div class="section-head">
          <h2>Artifacts</h2>
          <span class="muted">{artifacts.length}</span>
        </div>
        {artifacts.length > 0 ? (
          <dl class="kv-list">
            {artifacts.map(([key, value]) => (
              <div key={key}>
                <dt>{key}</dt>
                <dd>{formatValue(value)}</dd>
              </div>
            ))}
          </dl>
        ) : (
          <EmptyState>No artifacts available.</EmptyState>
        )}
      </section>
    );
  }

  return (
    <WorkbenchFrame
      explorerLabel="Explorer"
      explorerTitle="Status"
      explorerCount={String(sections.length)}
      pageTitle="Status"
      detailLabel={activeSectionLabel}
      loading={loading}
      storageKey="status"
      defaultExplorerWidth={288}
      explorerContent={
        <ExplorerList ariaLabel="Status sections">
          {sections.map((item) => (
            <ExplorerItem
              key={item.id}
              selected={item.id === selectedSection}
              onSelect={() => onSelectSection(item.id)}
              title={item.label}
              meta={item.meta}
            />
          ))}
        </ExplorerList>
      }
    >
      {error ? <Notice tone="error">{error}</Notice> : null}
      <div class="inspector-panel">
        <InspectorHeader title={inspectorTitle} subtitle={inspectorSubtitle} />
        {inspectorBody}
      </div>
    </WorkbenchFrame>
  );
}

type FlattenedPlanHeading = PlanHeading & { nodeId: string };

export function PlanWorkspace(props: {
  loading: boolean;
  error: string | null;
  summary: string;
  document: PlanDocument | null;
  supplements: PlanNode | null;
  warnings: string[];
  artifacts: Array<[string, unknown]>;
}) {
  const { loading, error, summary, document, supplements, warnings, artifacts } = props;
  const documentRootId = document ? `document:${document.path}` : "document";
  const [selectedNodeId, setSelectedNodeId] = useState<string>(documentRootId);
  const [expandedNodeIds, setExpandedNodeIds] = useState<Set<string>>(new Set());
  const readerRef = useRef<HTMLDivElement | null>(null);

  const flattenedHeadings = useMemo(() => flattenPlanHeadings(document?.headings ?? []), [document?.headings]);
  const documentHTML = useMemo(() => (document ? markdownRenderer.render(document.markdown) : ""), [document]);

  useEffect(() => {
    setExpandedNodeIds(buildDefaultPlanExpanded(documentRootId, document?.headings ?? [], supplements));
  }, [documentRootId, document?.headings, supplements]);

  useEffect(() => {
    if (!document) {
      setSelectedNodeId(supplements ? planSupplementSelectionId(supplements) : "document");
      return;
    }

    setSelectedNodeId((current) => {
      if (current === documentRootId) return current;
      if (flattenedHeadings.some((heading) => heading.nodeId === current)) return current;
      if (supplements && findSupplementNodeBySelectionId(supplements, current)) return current;
      return documentRootId;
    });
  }, [document, documentRootId, flattenedHeadings, supplements]);

  const selectedHeading = flattenedHeadings.find((heading) => heading.nodeId === selectedNodeId) ?? null;
  const selectedSupplementNode = supplements ? findSupplementNodeBySelectionId(supplements, selectedNodeId) : null;
  const selectedFile = selectedSupplementNode?.kind === "file" ? selectedSupplementNode : null;
  const selectedDirectory = selectedSupplementNode?.kind === "directory" ? selectedSupplementNode : null;
  const detailLabel = selectedHeading?.label || selectedFile?.label || selectedDirectory?.label || document?.title || "Current plan";
  const explorerCount = document ? String((supplements ? 2 : 1)) : supplements ? "1" : "0";

  useEffect(() => {
    if (!document || selectedFile) return;
    const root = readerRef.current;
    if (!root) return;

    const renderedHeadings = Array.from(root.querySelectorAll("h1, h2, h3, h4, h5, h6"));
    const headingElements = renderedHeadings.filter(
      (element, index) => !(index === 0 && element.tagName === "H1" && normalizePlanText(element.textContent || "") === normalizePlanText(document.title)),
    );

    headingElements.forEach((element, index) => {
      const heading = flattenedHeadings[index];
      if (heading) {
        element.id = heading.anchor;
      }
    });

    if (selectedHeading) {
      const target = headingElements.find((element) => element.id === selectedHeading.anchor) as HTMLElement | undefined;
      if (target) {
        root.scrollTop = Math.max(0, target.offsetTop - 18);
        return;
      }
    }
    root.scrollTop = 0;
  }, [document, documentHTML, flattenedHeadings, selectedFile, selectedHeading]);

  const toggleNode = (id: string) => {
    setExpandedNodeIds((current) => {
      const next = new Set(current);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const renderHeadingNode = (heading: PlanHeading, depth: number) => {
    const nodeId = planHeadingSelectionId(heading);
    const isExpanded = expandedNodeIds.has(nodeId);
    const hasChildren = Array.isArray(heading.children) && heading.children.length > 0;

    return (
      <div key={nodeId} class="plan-tree-branch">
        <div class={`plan-tree-row${selectedNodeId === nodeId ? " is-active" : ""}`} style={{ "--plan-depth": String(depth) }}>
          <button
            type="button"
            class={`plan-tree-toggle${hasChildren ? "" : " is-placeholder"}`}
            onClick={() => hasChildren && toggleNode(nodeId)}
            aria-label={hasChildren ? `${isExpanded ? "Collapse" : "Expand"} ${heading.label}` : undefined}
            disabled={!hasChildren}
          >
            {hasChildren ? (isExpanded ? "v" : ">") : ""}
          </button>
          <button type="button" class="plan-tree-label" onClick={() => setSelectedNodeId(nodeId)}>
            <span class="plan-tree-text">{heading.label}</span>
            <span class="plan-tree-meta">H{heading.level}</span>
          </button>
        </div>
        {hasChildren && isExpanded ? <div class="plan-tree-children">{heading.children?.map((child) => renderHeadingNode(child, depth + 1))}</div> : null}
      </div>
    );
  };

  const renderSupplementNode = (node: PlanNode, depth: number) => {
    const nodeId = planSupplementSelectionId(node);
    const hasChildren = node.kind === "directory" && Array.isArray(node.children) && node.children.length > 0;
    const isExpanded = expandedNodeIds.has(nodeId);
    const previewStatus = node.preview?.status === "fallback" ? "TXT" : node.preview?.content_type?.toUpperCase() || "";

    return (
      <div key={nodeId} class="plan-tree-branch">
        <div class={`plan-tree-row${selectedNodeId === nodeId ? " is-active" : ""}`} style={{ "--plan-depth": String(depth) }}>
          <button
            type="button"
            class={`plan-tree-toggle${hasChildren ? "" : " is-placeholder"}`}
            onClick={() => hasChildren && toggleNode(nodeId)}
            aria-label={hasChildren ? `${isExpanded ? "Collapse" : "Expand"} ${node.label}` : undefined}
            disabled={!hasChildren}
          >
            {hasChildren ? (isExpanded ? "v" : ">") : ""}
          </button>
          <button type="button" class="plan-tree-label" onClick={() => setSelectedNodeId(nodeId)}>
            <span class="plan-tree-text">{node.kind === "directory" ? node.label : node.label}</span>
            {node.kind === "file" && previewStatus ? <span class="plan-tree-meta">{previewStatus}</span> : null}
          </button>
        </div>
        {hasChildren && isExpanded ? <div class="plan-tree-children">{node.children?.map((child) => renderSupplementNode(child, depth + 1))}</div> : null}
      </div>
    );
  };

  return (
    <WorkbenchFrame
      explorerLabel="Explorer"
      explorerTitle="Plan"
      explorerCount={explorerCount}
      pageTitle="Plan"
      detailLabel={detailLabel}
      loading={loading}
      storageKey="plan"
      defaultExplorerWidth={320}
      explorerContent={
        document || supplements ? (
          <div class="plan-tree" aria-label="Plan package explorer">
            {document ? (
              <div class="plan-tree-branch">
                <div class={`plan-tree-row${selectedNodeId === documentRootId ? " is-active" : ""}`} style={{ "--plan-depth": "0" }}>
                  <button
                    type="button"
                    class={`plan-tree-toggle${document.headings.length > 0 ? "" : " is-placeholder"}`}
                    onClick={() => document.headings.length > 0 && toggleNode(documentRootId)}
                    aria-label={document.headings.length > 0 ? `${expandedNodeIds.has(documentRootId) ? "Collapse" : "Expand"} ${document.title}` : undefined}
                    disabled={document.headings.length === 0}
                  >
                    {document.headings.length > 0 ? (expandedNodeIds.has(documentRootId) ? "v" : ">") : ""}
                  </button>
                  <button type="button" class="plan-tree-label" onClick={() => setSelectedNodeId(documentRootId)}>
                    <span class="plan-tree-text">{document.title}</span>
                    <span class="plan-tree-meta">PLAN</span>
                  </button>
                </div>
                {expandedNodeIds.has(documentRootId) ? (
                  <div class="plan-tree-children">{document.headings.map((heading) => renderHeadingNode(heading, 1))}</div>
                ) : null}
              </div>
            ) : null}

            {supplements ? (
              <div class="plan-tree-branch">
                <div class={`plan-tree-row${selectedNodeId === planSupplementSelectionId(supplements) ? " is-active" : ""}`} style={{ "--plan-depth": "0" }}>
                  <button
                    type="button"
                    class={`plan-tree-toggle${supplements.children?.length ? "" : " is-placeholder"}`}
                    onClick={() => supplements.children?.length && toggleNode(planSupplementSelectionId(supplements))}
                    aria-label={supplements.children?.length ? `${expandedNodeIds.has(planSupplementSelectionId(supplements)) ? "Collapse" : "Expand"} supplements` : undefined}
                    disabled={!supplements.children?.length}
                  >
                    {supplements.children?.length ? (expandedNodeIds.has(planSupplementSelectionId(supplements)) ? "v" : ">") : ""}
                  </button>
                  <button type="button" class="plan-tree-label" onClick={() => setSelectedNodeId(planSupplementSelectionId(supplements))}>
                    <span class="plan-tree-text">supplements</span>
                    <span class="plan-tree-meta">DIR</span>
                  </button>
                </div>
                {expandedNodeIds.has(planSupplementSelectionId(supplements)) ? (
                  <div class="plan-tree-children">{supplements.children?.map((node) => renderSupplementNode(node, 1))}</div>
                ) : null}
              </div>
            ) : null}
          </div>
        ) : (
          <EmptyState>{summary}</EmptyState>
        )
      }
    >
      {error ? <Notice tone="error">{error}</Notice> : null}
      {warnings.map((warning) => (
        <Notice key={warning} tone="warning">
          {warning}
        </Notice>
      ))}

      {document || supplements ? (
        <div class="inspector-panel">
          <InspectorHeader
            title={selectedHeading?.label || selectedFile?.label || selectedDirectory?.label || document?.title || "Plan"}
            subtitle={
              selectedHeading
                ? `${selectedHeading.label} · ${selectedHeading.level ? `H${selectedHeading.level}` : "heading"}`
                : selectedFile?.path || selectedDirectory?.path || document?.path || summary
            }
            meta={
              selectedFile?.preview ? (
                <>
                  <StatusBadge tone={selectedFile.preview.status === "supported" ? "good" : selectedFile.preview.status === "fallback" ? "warning" : "muted"}>
                    {selectedFile.preview.status === "fallback" ? "Plain Text" : humanizeLabel(selectedFile.preview.status)}
                  </StatusBadge>
                  <span>{selectedFile.preview.byte_size ? `${selectedFile.preview.byte_size} bytes` : ""}</span>
                </>
              ) : null
            }
          />

          {selectedFile ? (
            <PlanFilePreview file={selectedFile} />
          ) : selectedDirectory ? (
            <section class="content-section">
              <div class="section-head">
                <h2>{selectedDirectory.label === supplements?.label ? "Supplements" : selectedDirectory.label}</h2>
                <span class="muted">{selectedDirectory.children?.length ?? 0}</span>
              </div>
              <p class="detail-copy">
                {selectedDirectory.children?.length
                  ? "Choose a child file to preview its contents."
                  : "This folder is present but does not contain any previewable entries yet."}
              </p>
            </section>
          ) : document ? (
            <div class="plan-reader-shell">
              <div ref={readerRef} class="plan-reader" dangerouslySetInnerHTML={{ __html: documentHTML }} />
            </div>
          ) : (
            <EmptyState>{summary}</EmptyState>
          )}

          {artifacts.length > 0 ? (
            <section class="content-section content-section-secondary">
              <div class="section-head">
                <h2>Package metadata</h2>
                <span class="muted">{artifacts.length}</span>
              </div>
              <dl class="kv-list">
                {artifacts.map(([key, value]) => (
                  <div key={key}>
                    <dt>{key}</dt>
                    <dd>{formatValue(value)}</dd>
                  </div>
                ))}
              </dl>
            </section>
          ) : null}
        </div>
      ) : (
        <EmptyState>{summary}</EmptyState>
      )}
    </WorkbenchFrame>
  );
}

function PlanFilePreview(props: { file: PlanNode }) {
  const { file } = props;
  const preview = file.preview;
  if (!preview) {
    return <EmptyState>No preview information is available for this file.</EmptyState>;
  }

  if (preview.status === "not_supported") {
    return (
      <section class="content-section">
        <div class="section-head">
          <h2>Preview unavailable</h2>
          <span class="muted">{preview.extension || "file"}</span>
        </div>
        <p class="detail-copy">{preview.reason || "This file type is not supported yet."}</p>
      </section>
    );
  }

  if (preview.content_type === "markdown") {
    return (
      <div class="plan-reader-shell">
        {preview.reason ? <div class="plan-preview-note">{preview.reason}</div> : null}
        <div class="plan-reader" dangerouslySetInnerHTML={{ __html: markdownRenderer.render(preview.content || "") }} />
      </div>
    );
  }

  return (
    <section class="content-section">
      {preview.reason ? <div class="plan-preview-note">{preview.reason}</div> : null}
      <pre class="inspector-json plan-code-block">{preview.content || ""}</pre>
    </section>
  );
}

function flattenPlanHeadings(headings: PlanHeading[]): FlattenedPlanHeading[] {
  const flattened: FlattenedPlanHeading[] = [];
  const visit = (items: PlanHeading[]) => {
    items.forEach((item) => {
      flattened.push({ ...item, nodeId: planHeadingSelectionId(item) });
      if (Array.isArray(item.children) && item.children.length > 0) {
        visit(item.children);
      }
    });
  };
  visit(headings);
  return flattened;
}

function buildDefaultPlanExpanded(documentRootId: string, headings: PlanHeading[], supplements: PlanNode | null): Set<string> {
  const expanded = new Set<string>();
  if (headings.length > 0) {
    expanded.add(documentRootId);
  }
  const visit = (items: PlanHeading[]) => {
    items.forEach((item) => {
      if (Array.isArray(item.children) && item.children.length > 0 && item.level < 3) {
        expanded.add(planHeadingSelectionId(item));
        visit(item.children);
      }
    });
  };
  visit(headings);
  if (supplements) {
    expanded.add(planSupplementSelectionId(supplements));
  }
  return expanded;
}

function planHeadingSelectionId(heading: PlanHeading): string {
  return `heading:${heading.id}`;
}

function planSupplementSelectionId(node: PlanNode): string {
  return `${node.kind}:${node.path || node.id}`;
}

function findSupplementNodeBySelectionId(root: PlanNode, selectionId: string): PlanNode | null {
  if (planSupplementSelectionId(root) === selectionId) return root;
  if (!Array.isArray(root.children)) return null;
  for (const child of root.children) {
    const found = findSupplementNodeBySelectionId(child, selectionId);
    if (found) return found;
  }
  return null;
}

function normalizePlanText(value: string): string {
  return value.replace(/\s+/g, " ").trim().toLowerCase();
}

export function TimelineWorkspace(props: {
  loading: boolean;
  error: string | null;
  events: TimelineEvent[];
}) {
  const { loading, error, events } = props;
  const sortedEvents = useMemo(() => sortTimelineEvents(events), [events]);
  const [selectedEventId, setSelectedEventId] = useState<string | null>(null);
  const selectedEvent = useMemo(() => {
    if (sortedEvents.length === 0) return null;
    if (selectedEventId) {
      const found = sortedEvents.find((event) => event.event_id === selectedEventId);
      if (found) return found;
    }
    return pickDefaultTimelineEvent(sortedEvents);
  }, [selectedEventId, sortedEvents]);
  const [selectedTab, setSelectedTab] = useState<string>("event");
  const timelineTabs = useMemo(() => buildTimelineTabs(selectedEvent), [selectedEvent]);

  useEffect(() => {
    if (sortedEvents.length === 0) {
      setSelectedEventId(null);
      return;
    }
    setSelectedEventId((current) => {
      if (current && sortedEvents.some((event) => event.event_id === current)) {
        return current;
      }
      return pickDefaultTimelineEvent(sortedEvents)?.event_id ?? null;
    });
  }, [sortedEvents]);

  useEffect(() => {
    if (timelineTabs.length === 0) {
      setSelectedTab("event");
      return;
    }
    setSelectedTab((current) => (timelineTabs.some((tab) => tab.id === current) ? current : timelineTabs[0].id));
  }, [timelineTabs]);

  const transitionLabel =
    selectedEvent && (selectedEvent.from_node || selectedEvent.to_node)
      ? `${selectedEvent.from_node || "unknown"} → ${selectedEvent.to_node || "unknown"}`
      : null;
  const selectedTimelineTab = timelineTabs.find((tab) => tab.id === selectedTab) ?? timelineTabs[0];

  return (
    <WorkbenchFrame
      explorerLabel="Explorer"
      explorerTitle="Timeline"
      explorerCount={String(sortedEvents.length)}
      pageTitle="Timeline"
      detailLabel={selectedEvent ? timelineEventTitle(selectedEvent) : "Events"}
      loading={loading}
      storageKey="timeline"
      defaultExplorerWidth={304}
      explorerContent={
        <ExplorerList ariaLabel="Timeline events">
          {sortedEvents.length > 0 ? (
            sortedEvents.map((event) => (
              <ExplorerItem
                key={event.event_id}
                selected={event.event_id === selectedEvent?.event_id}
                onSelect={() => setSelectedEventId(event.event_id)}
                title={timelineEventTitle(event)}
                subtitle={timelineEventSubtitle(event)}
                meta={formatTimestamp(event.recorded_at)}
              />
            ))
          ) : (
            <EmptyState>No timeline events recorded yet for this plan.</EmptyState>
          )}
        </ExplorerList>
      }
    >
      {error ? <Notice tone="error">{error}</Notice> : null}
      <div class="inspector-panel">
        <InspectorHeader
          title={selectedEvent ? timelineEventTitle(selectedEvent) : "Timeline"}
          subtitle={selectedEvent ? selectedEvent.summary : "Select an event to inspect its payload."}
          meta={
            selectedEvent ? (
              <>
                <span>{selectedEvent.event_id}</span>
                <span>{formatTimestamp(selectedEvent.recorded_at)}</span>
              </>
            ) : null
          }
        />

        {selectedEvent ? (
          <>
            {transitionLabel ? <div class="inspector-transition">{transitionLabel}</div> : null}
            <InspectorTabs ariaLabel="Timeline event payloads">
              {timelineTabs.map((tab) => (
                <InspectorTab key={tab.id} selected={selectedTab === tab.id} onSelect={() => setSelectedTab(tab.id)}>
                  {tab.label}
                </InspectorTab>
              ))}
            </InspectorTabs>
            <pre class="inspector-json" aria-label={`${selectedTimelineTab?.label ?? "selected"} payload`}>
              {timelineTabText(selectedTimelineTab?.value ?? selectedEvent, selectedTimelineTab?.mode ?? "json")}
            </pre>
          </>
        ) : (
          <EmptyState>Select an event to inspect its raw payload.</EmptyState>
        )}
      </div>
    </WorkbenchFrame>
  );
}

export function ReviewWorkspace(props: {
  loading: boolean;
  error: string | null;
  summary: string;
  rounds: ReviewRound[];
  warnings: string[];
  artifacts: Array<[string, unknown]>;
}) {
  const { loading, error, summary, rounds, warnings, artifacts } = props;
  const [selectedRoundId, setSelectedRoundId] = useState<string | null>(null);
  const [selectedDetailTab, setSelectedDetailTab] = useState<string>("summary");
  const [selectedArtifactKey, setSelectedArtifactKey] = useState<string | null>(null);
  const [supportExpanded, setSupportExpanded] = useState(false);

  const selectedRound = useMemo(() => {
    if (rounds.length === 0) return null;
    if (selectedRoundId) {
      const found = rounds.find((round) => round.round_id === selectedRoundId);
      if (found) return found;
    }
    return rounds[0];
  }, [rounds, selectedRoundId]);

  const reviewers = Array.isArray(selectedRound?.reviewers) ? selectedRound.reviewers ?? [] : [];
  const supportArtifacts = Array.isArray(selectedRound?.artifacts) ? selectedRound.artifacts ?? [] : [];
  const selectedReviewer = useMemo(() => {
    if (reviewers.length === 0 || selectedDetailTab === "summary") return null;
    return reviewers.find((reviewer) => reviewer.slot === selectedDetailTab) ?? null;
  }, [reviewers, selectedDetailTab]);
  const selectedArtifact = useMemo(() => {
    if (supportArtifacts.length === 0) return null;
    if (selectedArtifactKey) {
      const found = supportArtifacts.find((artifact, index) => reviewArtifactKey(artifact, index) === selectedArtifactKey);
      if (found) return found;
    }
    return supportArtifacts[0];
  }, [supportArtifacts, selectedArtifactKey]);

  const blockingFindings = Array.isArray(selectedRound?.blocking_findings) ? selectedRound.blocking_findings ?? [] : [];
  const nonBlockingFindings = Array.isArray(selectedRound?.non_blocking_findings) ? selectedRound.non_blocking_findings ?? [] : [];
  const selectedRoundWarnings = Array.isArray(selectedRound?.warnings) ? selectedRound.warnings ?? [] : [];

  useEffect(() => {
    if (rounds.length === 0) {
      setSelectedRoundId(null);
      return;
    }
    setSelectedRoundId((current) => (current && rounds.some((round) => round.round_id === current) ? current : rounds[0]?.round_id ?? null));
  }, [rounds]);

  useEffect(() => {
    setSelectedDetailTab("summary");
    setSupportExpanded(false);
  }, [selectedRound?.round_id]);

  useEffect(() => {
    setSelectedDetailTab((current) => {
      if (current === "summary") return "summary";
      return reviewers.some((reviewer) => reviewer.slot === current) ? current : reviewers[0]?.slot ?? "summary";
    });
  }, [reviewers]);

  useEffect(() => {
    if (supportArtifacts.length === 0) {
      setSelectedArtifactKey(null);
      return;
    }
    setSelectedArtifactKey((current) =>
      current && supportArtifacts.some((artifact, index) => reviewArtifactKey(artifact, index) === current)
        ? current
        : reviewArtifactKey(supportArtifacts[0], 0),
    );
  }, [supportArtifacts]);

  return (
    <WorkbenchFrame
      explorerLabel="Explorer"
      explorerTitle="Review"
      explorerCount={String(rounds.length)}
      pageTitle="Review"
      detailLabel={selectedRound ? reviewRoundTitle(selectedRound) : "Rounds"}
      loading={loading}
      storageKey="review"
      defaultExplorerWidth={304}
      explorerContent={
        <ExplorerList ariaLabel="Review rounds">
          {rounds.length > 0 ? (
            rounds.map((round) => (
              <ExplorerItem
                key={round.round_id}
                selected={round.round_id === selectedRound?.round_id}
                onSelect={() => setSelectedRoundId(round.round_id)}
                ariaLabel={reviewRoundAriaLabel(round)}
                title={
                  <div class="review-explorer-title">
                    <span class="review-explorer-title-text">{reviewRoundTitle(round)}</span>
                    <span class={`review-round-indicator is-${reviewRoundStatusTone(round)}`} aria-hidden="true" />
                  </div>
                }
                subtitle={`${reviewRoundSubtitle(round)} · ${reviewCountLabel(round.submitted_slots)}/${reviewCountLabel(round.total_slots)} submitted`}
                trailing={<span class="review-round-status-text">{reviewRoundCompactStatusLabel(round)}</span>}
                tone={reviewRoundStatusTone(round)}
              />
            ))
          ) : (
            <EmptyState>{summary || "No review rounds recorded yet for the current plan."}</EmptyState>
          )}
        </ExplorerList>
      }
    >
      {error ? <Notice tone="error">{error}</Notice> : null}
      {warnings.map((warning) => (
        <Notice key={warning} tone="warning">
          {warning}
        </Notice>
      ))}

      {selectedRound ? (
        <div class="inspector-panel">
          <InspectorHeader
            title={reviewRoundTitle(selectedRound)}
            subtitle={reviewRoundListLabel(selectedRound)}
            meta={
              <>
                <StatusBadge tone={reviewRoundStatusTone(selectedRound)}>{reviewRoundStatusLabel(selectedRound)}</StatusBadge>
                <span>{formatTimestamp(selectedRound.aggregated_at || selectedRound.updated_at || selectedRound.created_at || "")}</span>
              </>
            }
          />

          <InspectorTabs ariaLabel="Review content tabs">
            <InspectorTab selected={selectedDetailTab === "summary"} onSelect={() => setSelectedDetailTab("summary")}>
              Summary
            </InspectorTab>
            {reviewers.map((reviewer) => (
              <InspectorTab key={reviewer.slot} selected={selectedDetailTab === reviewer.slot} onSelect={() => setSelectedDetailTab(reviewer.slot)}>
                {reviewReviewerLabel(reviewer)}
              </InspectorTab>
            ))}
          </InspectorTabs>

          {selectedDetailTab === "summary" ? (
            <div class="review-tab-panel">
              <section class="content-section">
                <div class="section-head">
                  <h2>Overview</h2>
                  <span class="muted">{reviewRoundCompactMeta(selectedRound)}</span>
                </div>
                <p class="detail-copy">{selectedRound.status_summary || summary}</p>
                <section class="summary-metrics review-summary-metrics" aria-label="Review summary">
                  <div class="summary-metric">
                    <span class="label">Decision</span>
                    <strong>{selectedRound.decision ? humanizeLabel(selectedRound.decision) : reviewRoundStatusLabel(selectedRound)}</strong>
                  </div>
                  <div class="summary-metric">
                    <span class="label">Progress</span>
                    <strong>{reviewCountLabel(selectedRound.submitted_slots)}/{reviewCountLabel(selectedRound.total_slots)} submitted</strong>
                  </div>
                  <div class="summary-metric">
                    <span class="label">Revision</span>
                    <strong>{selectedRound.revision ? `rev ${selectedRound.revision}` : "unknown"}</strong>
                  </div>
                  <div class="summary-metric">
                    <span class="label">Target</span>
                    <strong>{typeof selectedRound.step === "number" ? `Step ${selectedRound.step}` : selectedRound.review_title || "Finalize / unscoped"}</strong>
                  </div>
                </section>
              </section>

              {selectedRoundWarnings.length > 0 ? (
                <section class="content-section">
                  <div class="section-head">
                    <h2>Warnings</h2>
                    <span class="muted">{selectedRoundWarnings.length}</span>
                  </div>
                  <div class="warning-stack">
                    {selectedRoundWarnings.map((warning) => (
                      <div key={warning} class="warning-item is-warning">
                        {warning}
                      </div>
                    ))}
                  </div>
                </section>
              ) : null}

              <section class="content-section">
                <div class="section-head">
                  <h2>Blocking findings</h2>
                  <span class="muted">{blockingFindings.length}</span>
                </div>
                {blockingFindings.length > 0 ? (
                  <div class="review-finding-list">
                    {blockingFindings.map((finding, index) => (
                      <ReviewFindingCard key={reviewFindingKey(finding, index)} finding={finding} provenance={reviewAggregateFindingSource(finding as ReviewAggregateFinding)} />
                    ))}
                  </div>
                ) : (
                  <EmptyState>No blocking findings recorded.</EmptyState>
                )}
              </section>

              <section class="content-section">
                <div class="section-head">
                  <h2>Non-blocking findings</h2>
                  <span class="muted">{nonBlockingFindings.length}</span>
                </div>
                {nonBlockingFindings.length > 0 ? (
                  <div class="review-finding-list">
                    {nonBlockingFindings.map((finding, index) => (
                      <ReviewFindingCard key={reviewFindingKey(finding, index)} finding={finding} provenance={reviewAggregateFindingSource(finding as ReviewAggregateFinding)} />
                    ))}
                  </div>
                ) : (
                  <EmptyState>No non-blocking findings recorded.</EmptyState>
                )}
              </section>
            </div>
          ) : selectedReviewer ? (
            <ReviewerInspector
              reviewer={selectedReviewer}
              selectedRound={selectedRound}
              blockingFindings={blockingFindings}
              warningCount={selectedRoundWarnings.length}
            />
          ) : (
            <EmptyState>No reviewer slots are available for this round.</EmptyState>
          )}

          {supportArtifacts.length > 0 || artifacts.length > 0 ? (
            <section class="supporting-section">
              <button
                type="button"
                class={`supporting-toggle${supportExpanded ? " is-open" : ""}`}
                onClick={() => setSupportExpanded((current) => !current)}
                aria-expanded={supportExpanded}
              >
                <span>Supporting evidence</span>
                <span class="muted">{supportArtifacts.length + artifacts.length}</span>
              </button>
              {supportExpanded ? (
                <div class="supporting-stack">
                  <section class="content-section content-section-secondary">
                    <div class="section-head">
                      <h2>Artifact payloads</h2>
                      <span class="muted">{supportArtifacts.length}</span>
                    </div>
                    {supportArtifacts.length > 0 ? (
                      <>
                        <InspectorTabs ariaLabel="Supporting artifacts">
                          {supportArtifacts.map((artifact, index) => {
                            const artifactKey = reviewArtifactKey(artifact, index);
                            return (
                              <InspectorTab key={artifactKey} selected={selectedArtifactKey === artifactKey} onSelect={() => setSelectedArtifactKey(artifactKey)}>
                                {reviewArtifactLabel(artifact)}
                              </InspectorTab>
                            );
                          })}
                        </InspectorTabs>
                        {selectedArtifact ? <ArtifactInspector artifact={selectedArtifact} /> : null}
                      </>
                    ) : (
                      <EmptyState>No supporting artifacts available for this round.</EmptyState>
                    )}
                  </section>

                  {artifacts.length > 0 ? (
                    <section class="content-section content-section-secondary">
                      <div class="section-head">
                        <h2>Round metadata</h2>
                        <span class="muted">{artifacts.length}</span>
                      </div>
                      <dl class="kv-list">
                        {artifacts.map(([key, value]) => (
                          <div key={key}>
                            <dt>{key}</dt>
                            <dd>{formatValue(value)}</dd>
                          </div>
                        ))}
                      </dl>
                    </section>
                  ) : null}
                </div>
              ) : null}
            </section>
          ) : null}
        </div>
      ) : (
        <EmptyState>{summary || "No review rounds recorded yet for the current plan."}</EmptyState>
      )}
    </WorkbenchFrame>
  );
}

function ReviewerInspector(props: {
  reviewer: ReviewReviewer;
  selectedRound: ReviewRound;
  blockingFindings: ReviewFinding[];
  warningCount: number;
}) {
  const { reviewer, selectedRound, blockingFindings, warningCount } = props;
  return (
    <div class="review-tab-panel">
      <section class="content-section">
        <div class="section-head">
          <h2>{reviewReviewerLabel(reviewer)}</h2>
          <StatusBadge tone={reviewReviewerStatusTone(reviewer)}>{reviewReviewerStatusLabel(reviewer)}</StatusBadge>
        </div>
        <section class="summary-metrics review-summary-metrics" aria-label="Reviewer context">
          <div class="summary-metric">
            <span class="label">Round</span>
            <strong>{selectedRound.round_id}</strong>
          </div>
          <div class="summary-metric">
            <span class="label">Decision</span>
            <strong>{selectedRound.decision ? humanizeLabel(selectedRound.decision) : reviewRoundStatusLabel(selectedRound)}</strong>
          </div>
          <div class="summary-metric">
            <span class="label">Blocking</span>
            <strong>{blockingFindings.length}</strong>
          </div>
          <div class="summary-metric">
            <span class="label">Warnings</span>
            <strong>{warningCount}</strong>
          </div>
        </section>
      </section>

      <section class="content-section fold-section">
        <div class="section-head">
          <h2>Assigned task</h2>
          <span class="muted">{reviewer.instructions?.trim() ? "available" : "missing"}</span>
        </div>
        {reviewer.instructions?.trim() ? <p class="detail-copy">{reviewer.instructions}</p> : <EmptyState>Instructions are unavailable for this reviewer slot.</EmptyState>}
      </section>

      <section class="content-section fold-section">
        <div class="section-head">
          <h2>Returned result</h2>
          <span class="muted">
            {reviewer.summary?.trim() ? `${Array.isArray(reviewer.findings) ? reviewer.findings.length : 0} finding(s)` : reviewReviewerStatusLabel(reviewer)}
          </span>
        </div>
        {reviewer.summary?.trim() ? (
          <>
            <p class="detail-copy">{reviewer.summary}</p>
            <div class="review-finding-list">
              {Array.isArray(reviewer.findings) && reviewer.findings.length > 0 ? (
                reviewer.findings.map((finding, index) => <ReviewFindingCard key={reviewFindingKey(finding, index)} finding={finding} />)
              ) : (
                <EmptyState>No findings recorded for this reviewer.</EmptyState>
              )}
            </div>
          </>
        ) : (
          <EmptyState>This reviewer has not submitted a result yet.</EmptyState>
        )}
      </section>

      {Array.isArray(reviewer.warnings) && reviewer.warnings.length > 0 ? (
        <section class="content-section">
          <div class="section-head">
            <h2>Warnings</h2>
            <span class="muted">{reviewer.warnings.length}</span>
          </div>
          <div class="warning-stack">
            {reviewer.warnings.map((warning) => (
              <div key={warning} class="warning-item is-warning">
                {warning}
              </div>
            ))}
          </div>
        </section>
      ) : null}
    </div>
  );
}

function ArtifactInspector(props: { artifact: ReviewArtifact }) {
  const { artifact } = props;
  return (
    <div class="artifact-panel">
      <div class="artifact-meta">
        <StatusBadge tone={artifact.status === "available" ? "good" : artifact.status === "invalid" ? "danger" : "warning"}>
          {humanizeLabel(artifact.status || "unknown")}
        </StatusBadge>
        {artifact.path ? <span class="muted">{artifact.path}</span> : null}
      </div>
      {artifact.summary ? <p class="artifact-summary">{artifact.summary}</p> : null}
      <pre class="inspector-json">{reviewArtifactText(artifact)}</pre>
    </div>
  );
}
