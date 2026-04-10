import type { ComponentChildren } from "preact";
import { useEffect, useMemo, useState } from "preact/hooks";

import {
  buildTimelineTabs,
  formatTimestamp,
  formatValue,
  humanizeLabel,
  pickDefaultTimelineEvent,
  reviewAggregateFindingLabels,
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
  reviewRawSubmissionText,
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
  ReviewAggregateFinding,
  ReviewArtifact,
  ReviewFinding,
  ReviewRound,
  ReviewReviewer,
  ReviewWorklog,
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

function ReviewFindingCard(props: { finding: ReviewFinding; provenance?: string | null; provenanceLabels?: string[] }) {
  const { finding, provenance, provenanceLabels = [] } = props;
  return (
    <article class="review-finding">
      <div class="review-finding-head">
        <strong>{finding.title}</strong>
        <StatusBadge tone={reviewFindingBadgeTone(finding.severity)}>{humanizeLabel(finding.severity)}</StatusBadge>
      </div>
      {provenanceLabels.length > 0 ? (
        <div class="review-finding-provenance">
          {provenanceLabels.map((label) => (
            <span key={label} class="provenance-pill">
              {label}
            </span>
          ))}
        </div>
      ) : null}
      {provenance ? <div class="review-finding-meta">from {provenance}</div> : null}
      <p>{finding.details}</p>
      {Array.isArray(finding.locations) && finding.locations.length > 0 ? <div class="review-finding-locations">{finding.locations.join("\n")}</div> : null}
    </article>
  );
}

function ReviewCollapsibleSection(props: {
  title: string;
  meta?: ComponentChildren;
  defaultOpen?: boolean;
  children: ComponentChildren;
}) {
  const { title, meta, defaultOpen = true, children } = props;
  return (
    <details class="review-collapsible" open={defaultOpen}>
      <summary class="review-collapsible-summary">
        <span class="review-collapsible-title">
          <span class="review-collapsible-caret" aria-hidden="true">
            ▾
          </span>
          <span>{title}</span>
        </span>
        {meta ? <span class="review-collapsible-meta">{meta}</span> : null}
      </summary>
      <div class="review-collapsible-body">{children}</div>
    </details>
  );
}

function RawSubmissionOverlay(props: {
  title: string;
  value: unknown;
  onClose: () => void;
}) {
  const { title, value, onClose } = props;

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onClose();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [onClose]);

  return (
    <div class="raw-json-overlay" role="dialog" aria-modal="true" aria-label={title} onClick={onClose}>
      <div class="raw-json-dialog" onClick={(event) => event.stopPropagation()}>
        <div class="raw-json-header">
          <div>
            <div class="inspector-title">{title}</div>
            <div class="inspector-subtitle">Raw reviewer submission payload</div>
          </div>
          <button type="button" class="secondary-button" onClick={onClose}>
            Close
          </button>
        </div>
        <pre class="inspector-json raw-json-pre">{reviewRawSubmissionText(value)}</pre>
      </div>
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

function RoundArtifactsOverlay(props: {
  title: string;
  artifacts: ReviewArtifact[];
  metadata: Array<[string, unknown]>;
  selectedArtifactKey: string | null;
  onSelectArtifact: (key: string) => void;
  onClose: () => void;
}) {
  const { title, artifacts, metadata, selectedArtifactKey, onSelectArtifact, onClose } = props;
  const selectedArtifact =
    artifacts.find((artifact, index) => reviewArtifactKey(artifact, index) === selectedArtifactKey) ?? artifacts[0] ?? null;

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onClose();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [onClose]);

  return (
    <div class="raw-json-overlay" role="dialog" aria-modal="true" aria-label={title} onClick={onClose}>
      <div class="raw-json-dialog artifact-overlay-dialog" onClick={(event) => event.stopPropagation()}>
        <div class="raw-json-header">
          <div>
            <div class="inspector-title">{title}</div>
            <div class="inspector-subtitle">Round artifacts and supporting metadata</div>
          </div>
          <button type="button" class="secondary-button" onClick={onClose}>
            Close
          </button>
        </div>
        <div class="artifact-overlay-body">
          {artifacts.length > 0 ? (
            <>
              <InspectorTabs ariaLabel="Round artifacts">
                {artifacts.map((artifact, index) => {
                  const artifactKey = reviewArtifactKey(artifact, index);
                  return (
                    <InspectorTab key={artifactKey} selected={selectedArtifactKey === artifactKey} onSelect={() => onSelectArtifact(artifactKey)}>
                      {reviewArtifactLabel(artifact)}
                    </InspectorTab>
                  );
                })}
              </InspectorTabs>
              {selectedArtifact ? <ArtifactInspector artifact={selectedArtifact} /> : null}
            </>
          ) : (
            <EmptyState>No round artifacts available.</EmptyState>
          )}

          {metadata.length > 0 ? (
            <section class="content-section content-section-secondary artifact-overlay-section">
              <div class="section-head">
                <h2>Round metadata</h2>
              </div>
              <dl class="kv-list">
                {metadata.map(([key, value]) => (
                  <div key={key}>
                    <dt>{key}</dt>
                    <dd>{formatValue(value)}</dd>
                  </div>
                ))}
              </dl>
            </section>
          ) : null}
        </div>
      </div>
    </div>
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
  const [showArtifacts, setShowArtifacts] = useState(false);

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
    setShowArtifacts(false);
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
                {supportArtifacts.length > 0 || artifacts.length > 0 ? (
                  <button type="button" class="subtle-button" onClick={() => setShowArtifacts(true)}>
                    Artifacts
                  </button>
                ) : null}
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
                      <ReviewFindingCard
                        key={reviewFindingKey(finding, index)}
                        finding={finding}
                        provenanceLabels={reviewAggregateFindingLabels(finding as ReviewAggregateFinding)}
                      />
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
                      <ReviewFindingCard
                        key={reviewFindingKey(finding, index)}
                        finding={finding}
                        provenanceLabels={reviewAggregateFindingLabels(finding as ReviewAggregateFinding)}
                      />
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

        </div>
      ) : (
        <EmptyState>{summary || "No review rounds recorded yet for the current plan."}</EmptyState>
      )}

      {selectedRound && showArtifacts ? (
        <RoundArtifactsOverlay
          title={`${reviewRoundTitle(selectedRound)} artifacts`}
          artifacts={supportArtifacts}
          metadata={artifacts}
          selectedArtifactKey={selectedArtifactKey}
          onSelectArtifact={setSelectedArtifactKey}
          onClose={() => setShowArtifacts(false)}
        />
      ) : null}
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
  const [showRawSubmission, setShowRawSubmission] = useState(false);
  const worklog: ReviewWorklog | null = reviewer.worklog ?? null;
  const checkedAreas = Array.isArray(worklog?.checked_areas) ? worklog?.checked_areas ?? [] : [];
  const openQuestions = Array.isArray(worklog?.open_questions) ? worklog?.open_questions ?? [] : [];
  const candidateFindings = Array.isArray(worklog?.candidate_findings) ? worklog?.candidate_findings ?? [] : [];
  const reviewKind = worklog?.review_kind?.trim() || selectedRound.kind?.trim() || "";
  const anchorSHA = selectedRound.anchor_sha?.trim() || worklog?.anchor_sha?.trim() || "";
  const hasRawSubmission = reviewer.raw_submission !== undefined;
  const findings = Array.isArray(reviewer.findings) ? reviewer.findings ?? [] : [];
  const fullPlanReadLabel =
    worklog?.full_plan_read === true ? "Confirmed" : worklog?.full_plan_read === false ? "Not yet confirmed" : "Unknown";

  return (
    <div class="review-tab-panel">
      <section class="content-section">
        <div class="section-head">
          <h2>{reviewReviewerLabel(reviewer)}</h2>
          <div class="section-head-actions">
            {hasRawSubmission ? (
              <button type="button" class="subtle-button" onClick={() => setShowRawSubmission(true)}>
                Raw JSON
              </button>
            ) : null}
            <StatusBadge tone={reviewReviewerStatusTone(reviewer)}>{reviewReviewerStatusLabel(reviewer)}</StatusBadge>
          </div>
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

      <section class="content-section">
        <div class="section-head">
          <h2>Assigned task</h2>
        </div>
        {reviewer.instructions?.trim() ? <p class="detail-copy">{reviewer.instructions}</p> : <EmptyState>Instructions are unavailable for this reviewer slot.</EmptyState>}
      </section>

      <section class="content-section">
        <div class="section-head">
          <h2>Returned result</h2>
        </div>
        {reviewer.summary?.trim() ? (
          <>
            <p class="detail-copy">{reviewer.summary}</p>
            <div class="review-finding-list">
              {findings.length > 0 ? (
                findings.map((finding, index) => <ReviewFindingCard key={reviewFindingKey(finding, index)} finding={finding} />)
              ) : (
                <EmptyState>No findings recorded for this reviewer.</EmptyState>
              )}
            </div>
          </>
        ) : (
          <EmptyState>This reviewer has not submitted a result yet.</EmptyState>
        )}
      </section>

      <section class="content-section review-process-section">
        <div class="section-head">
          <h2>Review process</h2>
        </div>
        <ReviewCollapsibleSection
          title="Review context"
          defaultOpen={false}
          meta={reviewKind ? humanizeLabel(reviewKind) : reviewReviewerStatusLabel(reviewer)}
        >
          <dl class="kv-list">
            <div>
              <dt>Review kind</dt>
              <dd>{reviewKind ? humanizeLabel(reviewKind) : "Unknown"}</dd>
            </div>
            <div>
              <dt>Anchor</dt>
              <dd>{anchorSHA || "Not recorded"}</dd>
            </div>
            <div>
              <dt>Full plan read</dt>
              <dd>{fullPlanReadLabel}</dd>
            </div>
            <div>
              <dt>Submitted</dt>
              <dd>{reviewer.submitted_at ? formatTimestamp(reviewer.submitted_at) : "Not submitted"}</dd>
            </div>
          </dl>
        </ReviewCollapsibleSection>

        <ReviewCollapsibleSection title="Covered areas" defaultOpen={false} meta={`${checkedAreas.length} item(s)`}>
          {checkedAreas.length > 0 ? (
            <ul class="compact-list">
              {checkedAreas.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          ) : (
            <EmptyState>No covered areas recorded yet.</EmptyState>
          )}
        </ReviewCollapsibleSection>

        <ReviewCollapsibleSection title="Open questions" defaultOpen={openQuestions.length > 0} meta={`${openQuestions.length} item(s)`}>
          {openQuestions.length > 0 ? (
            <ul class="compact-list">
              {openQuestions.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          ) : (
            <EmptyState>No open questions recorded.</EmptyState>
          )}
        </ReviewCollapsibleSection>

        <ReviewCollapsibleSection
          title="Candidate findings"
          defaultOpen={candidateFindings.length > 0}
          meta={`${candidateFindings.length} item(s)`}
        >
          {candidateFindings.length > 0 ? (
            <ul class="compact-list">
              {candidateFindings.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          ) : (
            <EmptyState>No candidate findings recorded.</EmptyState>
          )}
        </ReviewCollapsibleSection>
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

      {showRawSubmission ? (
        <RawSubmissionOverlay title={`${reviewReviewerLabel(reviewer)} raw submission`} value={reviewer.raw_submission} onClose={() => setShowRawSubmission(false)} />
      ) : null}
    </div>
  );
}
