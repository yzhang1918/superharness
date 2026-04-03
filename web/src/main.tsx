import { render } from "preact";
import { useEffect, useMemo, useState } from "preact/hooks";

import "./styles.css";

type Page = "status" | "timeline" | "review";
type PageDef = { id: Page; label: string; href: string };
type SectionLink = { id: string; label: string; meta?: string };

type NextAction = {
  command: string | null;
  description: string;
};

type ErrorDetail = {
  path: string;
  message: string;
};

type StatusResult = {
  ok: boolean;
  command: string;
  summary: string;
  state?: {
    current_node?: string;
  };
  facts?: Record<string, unknown> | null;
  artifacts?: Record<string, unknown> | null;
  next_actions?: NextAction[] | null;
  blockers?: ErrorDetail[] | null;
  warnings?: string[] | null;
  errors?: ErrorDetail[] | null;
};

type TimelineDetail = {
  key: string;
  value: string;
};

type TimelineArtifactRef = {
  label: string;
  value: string;
  path?: string;
  content_type?: string;
  content?: unknown;
};

type TimelineEvent = {
  event_id: string;
  sequence: number;
  recorded_at: string;
  kind: string;
  command: string;
  summary: string;
  synthetic?: boolean;
  plan_path?: string;
  plan_stem: string;
  revision?: number;
  from_node?: string;
  to_node?: string;
  details?: TimelineDetail[] | null;
  artifact_refs?: TimelineArtifactRef[] | null;
  input?: unknown;
  output?: unknown;
  artifacts?: unknown;
  payload?: unknown;
  raw_input?: unknown;
  raw_output?: unknown;
  raw_artifacts?: unknown;
  [key: string]: unknown;
};

type TimelineResult = {
  ok: boolean;
  resource: string;
  summary: string;
  artifacts?: {
    plan_path?: string;
    local_state_path?: string;
    event_index_path?: string;
  } | null;
  events?: TimelineEvent[] | null;
  errors?: ErrorDetail[] | null;
};

type ReviewArtifact = {
  label: string;
  path?: string;
  status?: string;
  summary?: string;
  content_type?: string;
  content?: unknown;
};

type ReviewFinding = {
  severity: string;
  title: string;
  details: string;
  locations?: string[] | null;
};

type ReviewAggregateFinding = ReviewFinding & {
  slot?: string;
  dimension?: string;
};

type ReviewReviewer = {
  name?: string;
  slot: string;
  instructions?: string;
  status?: string;
  submission_path?: string;
  submitted_at?: string;
  summary?: string;
  findings?: ReviewFinding[] | null;
  warnings?: string[] | null;
};

type ReviewRound = {
  round_id: string;
  kind?: string;
  step?: number;
  revision?: number;
  review_title?: string;
  status?: string;
  status_summary?: string;
  decision?: string;
  created_at?: string;
  updated_at?: string;
  aggregated_at?: string;
  is_active?: boolean;
  total_slots?: number;
  submitted_slots?: number;
  pending_slots?: number;
  reviewers?: ReviewReviewer[] | null;
  blocking_findings?: ReviewAggregateFinding[] | null;
  non_blocking_findings?: ReviewAggregateFinding[] | null;
  artifacts?: ReviewArtifact[] | null;
  warnings?: string[] | null;
};

type ReviewResult = {
  ok: boolean;
  resource: string;
  summary: string;
  artifacts?: {
    plan_path?: string;
    local_state_path?: string;
    reviews_dir?: string;
    active_round_id?: string;
  } | null;
  rounds?: ReviewRound[] | null;
  warnings?: string[] | null;
  errors?: ErrorDetail[] | null;
};

declare global {
  interface Window {
    __HARNESS_UI__?: {
      workdir?: string;
      repoName?: string;
      productName?: string;
    };
  }
}

const pages: PageDef[] = [
  { id: "status", label: "Status", href: "/status" },
  { id: "timeline", label: "Timeline", href: "/timeline" },
  { id: "review", label: "Review", href: "/review" },
];

function isPage(value: string | null): value is Page {
  return value === "status" || value === "timeline" || value === "review";
}

function pageFromPathname(pathname: string): Page | null {
  const trimmed = pathname.replace(/\/+$/, "");
  const value = trimmed.split("/").filter(Boolean).pop() ?? "";
  return isPage(value) ? value : null;
}

function readPageFromLocation(): Page {
  const pathnamePage = pageFromPathname(window.location.pathname);
  if (pathnamePage) return pathnamePage;
  const hashValue = window.location.hash.replace(/^#/, "");
  return isPage(hashValue) ? hashValue : "status";
}

function pageDefinition(page: Page): PageDef {
  return pages.find((item) => item.id === page) ?? pages[0];
}

function RailIcon(props: { page: Page }) {
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

function metadataValue(value: string | undefined): string {
  const trimmed = value?.trim() ?? "";
  if (!trimmed || /^__HARNESS_UI_[A-Z0-9_]+__$/.test(trimmed)) {
    return "";
  }
  return trimmed;
}

function workdirLabel(): string {
  return metadataValue(window.__HARNESS_UI__?.workdir) || "unknown worktree";
}

function productNameLabel(): string {
  return metadataValue(window.__HARNESS_UI__?.productName) || "easyharness";
}

function formatValue(value: unknown): string {
  if (value === null) return "null";
  if (value === undefined) return "undefined";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  if (Array.isArray(value)) return `[${value.map(formatValue).join(", ")}]`;
  if (typeof value === "object") return JSON.stringify(value, null, 2);
  return String(value);
}

function pickEntries(value: Record<string, unknown> | null | undefined): Array<[string, unknown]> {
  if (!value || typeof value !== "object" || Array.isArray(value)) return [];
  return Object.entries(value);
}

function formatTimestamp(value: string): string {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  return parsed.toLocaleString(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  });
}

function humanizeLabel(value: string): string {
  const normalized = value.replace(/[_-]+/g, " ").trim();
  return normalized ? normalized.charAt(0).toUpperCase() + normalized.slice(1) : value;
}

function titleizeLabel(value: string): string {
  return value
    .replace(/[_-]+/g, " ")
    .trim()
    .split(/\s+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function timelineEventTitle(event: TimelineEvent): string {
  const command = event.command.trim();
  if (command) return command;
  const kind = event.kind.trim();
  if (kind) return humanizeLabel(kind);
  return `event ${event.sequence}`;
}

function timelineEventSubtitle(event: TimelineEvent): string {
  const parts = [event.synthetic ? "bootstrap" : humanizeLabel(event.kind)];
  if (event.revision !== undefined) {
    parts.push(`rev ${event.revision}`);
  }
  return parts.join(" · ");
}

function sortTimelineEvents(events: TimelineEvent[]): TimelineEvent[] {
  return [...events].sort((left, right) => {
    const leftTime = Date.parse(left.recorded_at);
    const rightTime = Date.parse(right.recorded_at);
    if (!Number.isNaN(leftTime) && !Number.isNaN(rightTime) && leftTime !== rightTime) {
      return rightTime - leftTime;
    }
    if (!Number.isNaN(leftTime) && Number.isNaN(rightTime)) return -1;
    if (Number.isNaN(leftTime) && !Number.isNaN(rightTime)) return 1;
    if (left.sequence !== right.sequence) return right.sequence - left.sequence;
    if (left.synthetic !== right.synthetic) return left.synthetic ? 1 : -1;
    return right.event_id.localeCompare(left.event_id);
  });
}

function pickDefaultTimelineEvent(events: TimelineEvent[]): TimelineEvent | null {
  if (events.length === 0) return null;
  for (const event of events) {
    if (!event.synthetic) return event;
  }
  return events[0];
}

function jsonStringify(value: unknown): string {
  if (value === undefined) return "";
  if (typeof value === "string") return JSON.stringify(value, null, 2);
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

function firstDefinedValue(event: TimelineEvent, keys: string[]): unknown {
  for (const key of keys) {
    const next = event[key];
    if (next !== undefined && next !== null) {
      if (typeof next === "string" && next.trim() === "") continue;
      return next;
    }
  }
  return undefined;
}

type TimelineTab = {
  id: string;
  label: string;
  value: unknown;
  mode: "json" | "text";
};

const hiddenArtifactTabLabels = new Set([
  "plan_path",
  "local_state_path",
  "current_plan_path",
  "from_plan_path",
  "to_plan_path",
]);

function artifactTabLabel(artifactRef: TimelineArtifactRef, index: number): string {
  const rawLabel = artifactRef.label?.trim();
  if (!rawLabel) return `Artifact ${index + 1}`;
  const normalized = rawLabel.replace(/_path$/i, "");
  return titleizeLabel(normalized);
}

function timelineEventRecord(event: TimelineEvent): Record<string, unknown> {
  const artifactRefs = Array.isArray(event.artifact_refs)
    ? event.artifact_refs.map((artifactRef) => ({
        label: artifactRef.label,
        value: artifactRef.value,
        ...(artifactRef.path ? { path: artifactRef.path } : {}),
      }))
    : undefined;
  const value: Record<string, unknown> = {};
  Object.entries(event).forEach(([key, next]) => {
    if (next === undefined) return;
    if (key === "artifact_refs") {
      if (artifactRefs && artifactRefs.length > 0) value.artifact_refs = artifactRefs;
      return;
    }
    value[key] = next;
  });
  return value;
}

function buildTimelineTabs(event: TimelineEvent | null): TimelineTab[] {
  if (!event) return [];

  const tabs: TimelineTab[] = [{ id: "event", label: "Event", value: timelineEventRecord(event), mode: "json" }];

  const inputValue = firstDefinedValue(event, ["input", "raw_input"]);
  if (inputValue !== undefined) {
    tabs.push({ id: "input", label: "Input", value: inputValue, mode: "json" });
  }

  const outputValue = firstDefinedValue(event, ["output", "raw_output"]);
  if (outputValue !== undefined) {
    tabs.push({ id: "output", label: "Output", value: outputValue, mode: "json" });
  }

  const artifactsValue = firstDefinedValue(event, ["artifacts", "raw_artifacts"]);
  if (artifactsValue !== undefined) {
    tabs.push({ id: "artifacts", label: "Artifacts", value: artifactsValue, mode: "json" });
  }

  const payloadValue = firstDefinedValue(event, ["payload"]);
  if (payloadValue !== undefined) {
    tabs.push({ id: "payload", label: "Payload", value: payloadValue, mode: "json" });
  }

  if (Array.isArray(event.artifact_refs)) {
    event.artifact_refs.forEach((artifactRef, index) => {
      if (hiddenArtifactTabLabels.has(artifactRef.label)) return;
      if (!artifactRef.path || artifactRef.content === undefined) return;
      tabs.push({
        id: `artifact-ref-${index}`,
        label: artifactTabLabel(artifactRef, index),
        value: artifactRef.content,
        mode: artifactRef.content_type === "text" ? "text" : "json",
      });
    });
  }

  return tabs;
}

function timelineTabText(value: unknown, mode: "json" | "text"): string {
  if (mode === "text" && typeof value === "string") return value;
  return jsonStringify(value);
}

function reviewRoundTitle(round: ReviewRound): string {
  const title = round.review_title?.trim();
  if (title) return title;
  if (typeof round.step === "number") return `Step ${round.step} review`;
  if (round.kind?.trim() === "full") return "Finalize review";
  return round.round_id;
}

function reviewRoundSequenceLabel(round: ReviewRound): string {
  const match = round.round_id.trim().match(/^review-(\d+)(?:-.+)?$/i);
  if (!match) return round.round_id;
  return `Round ${match[1]}`;
}

function reviewRoundListLabel(round: ReviewRound): string {
  const parts = [reviewRoundSequenceLabel(round)];
  if (round.kind?.trim()) parts.push(humanizeLabel(round.kind));
  if (typeof round.step === "number") {
    parts.push(`step ${round.step}`);
  } else if (round.kind?.trim() === "full") {
    parts.push("finalize");
  }
  if (typeof round.revision === "number" && round.revision > 0) parts.push(`rev ${round.revision}`);
  return parts.join(" · ");
}

function reviewRoundCompactMeta(round: ReviewRound): string {
  const parts = [reviewRoundSequenceLabel(round)];
  if (round.kind?.trim()) parts.push(humanizeLabel(round.kind));
  if (round.kind?.trim() === "full") {
    parts.push("finalize");
  }
  if (typeof round.revision === "number" && round.revision > 0) parts.push(`rev ${round.revision}`);
  return parts.join(" · ");
}

function reviewRoundCompactStatusLabel(round: ReviewRound): string {
  switch (round.status) {
    case "pass":
      return "PASS";
    case "changes_requested":
      return "CHG";
    case "waiting_for_submissions":
      return "WAIT";
    case "waiting_for_aggregation":
      return "AGGR";
    case "degraded":
      return "DEG";
    case "in_progress":
      return "WIP";
    case "complete":
      return "DONE";
    case "aggregated":
      return "AGGR";
    case "incomplete":
      return "PART";
    default:
      return reviewRoundStatusLabel(round).toUpperCase();
  }
}

function reviewRoundSubtitle(round: ReviewRound): string {
  const parts: string[] = [];
  if (typeof round.step === "number") {
    parts.push(`Step ${round.step}`);
  } else if (round.kind?.trim() === "full") {
    parts.push("Finalize scope");
  }
  return parts.join(" · ") || round.round_id;
}

function reviewRoundAriaLabel(round: ReviewRound): string {
  const parts = [
    reviewRoundTitle(round),
    reviewRoundStatusLabel(round),
    reviewRoundCompactMeta(round),
    reviewRoundSubtitle(round),
  ].filter(Boolean);
  const timestamp = round.created_at || round.updated_at || round.aggregated_at;
  if (timestamp) {
    parts.push(formatTimestamp(timestamp));
  }
  parts.push(`${reviewCountLabel(round.submitted_slots)}/${reviewCountLabel(round.total_slots)} submitted`);
  return parts.join(" ");
}

function reviewRoundStatusLabel(round: ReviewRound): string {
  const status = round.status?.trim();
  if (!status) return "Unknown";
  switch (status) {
    case "pass":
      return "Pass";
    case "changes_requested":
      return "Changes requested";
    case "waiting_for_submissions":
      return "Waiting for submissions";
    case "waiting_for_aggregation":
      return "Waiting for aggregation";
    case "degraded":
      return "Degraded";
    case "in_progress":
      return "In progress";
    case "complete":
      return "Complete";
    case "aggregated":
      return "Aggregated";
    case "incomplete":
      return "Incomplete";
    default:
      return humanizeLabel(status);
  }
}

function reviewRoundStatusTone(round: ReviewRound): "good" | "danger" | "warning" | "muted" {
  switch (round.status) {
    case "pass":
    case "complete":
      return "good";
    case "changes_requested":
    case "degraded":
      return "danger";
    case "waiting_for_submissions":
    case "waiting_for_aggregation":
    case "incomplete":
      return "warning";
    default:
      return "muted";
  }
}

function reviewReviewerLabel(reviewer: ReviewReviewer): string {
  return reviewer.name?.trim() || reviewer.slot;
}

function reviewReviewerStatusLabel(reviewer: ReviewReviewer): string {
  const status = reviewer.status?.trim();
  if (!status) return reviewer.summary?.trim() ? "Submitted" : "Pending";
  return humanizeLabel(status);
}

function reviewReviewerStatusTone(reviewer: ReviewReviewer): "good" | "danger" | "warning" | "muted" {
  const status = reviewer.status?.trim().toLowerCase();
  const hasWarnings = Array.isArray(reviewer.warnings) && reviewer.warnings.length > 0;
  if (!status) {
    if (reviewer.summary?.trim()) return hasWarnings ? "warning" : "good";
    return "warning";
  }
  if (status === "submitted") return hasWarnings ? "warning" : "good";
  if (status === "pending") return "warning";
  return hasWarnings ? "danger" : "warning";
}

function reviewFindingBadgeTone(severity: string): "danger" | "warning" {
  return severity === "minor" ? "warning" : "danger";
}

function reviewFindingKey(finding: ReviewFinding, index: number): string {
  const aggregateFinding = finding as ReviewAggregateFinding;
  return [aggregateFinding.slot, aggregateFinding.dimension, finding.title, finding.details, String(index)].filter(Boolean).join("::");
}

function reviewAggregateFindingSource(finding: ReviewAggregateFinding): string | null {
  const dimension = finding.dimension?.trim() ? humanizeLabel(finding.dimension) : "";
  const slot = finding.slot?.trim() ? humanizeLabel(finding.slot) : "";
  if (dimension && slot && dimension.toLowerCase() !== slot.toLowerCase()) {
    return `${dimension} · slot ${slot}`;
  }
  if (dimension) return dimension;
  if (slot) return `slot ${slot}`;
  return null;
}

function ReviewFindingCard(props: { finding: ReviewFinding; provenance?: string | null }) {
  const { finding, provenance } = props;
  return (
    <article class="review-finding">
      <div class="review-finding-head">
        <strong>{finding.title}</strong>
        <span class={`status-badge is-${reviewFindingBadgeTone(finding.severity)}`}>{humanizeLabel(finding.severity)}</span>
      </div>
      {provenance ? <div class="review-finding-meta">from {provenance}</div> : null}
      <p>{finding.details}</p>
      {Array.isArray(finding.locations) && finding.locations.length > 0 ? <div class="review-finding-locations">{finding.locations.join("\n")}</div> : null}
    </article>
  );
}

function reviewArtifactLabel(artifact: ReviewArtifact): string {
  return artifact.label?.trim() || "Artifact";
}

function reviewArtifactKey(artifact: ReviewArtifact, index: number): string {
  const path = artifact.path?.trim();
  if (path) return path;
  return `${reviewArtifactLabel(artifact)}-${index}`;
}

function reviewArtifactText(artifact: ReviewArtifact | null): string {
  if (!artifact) return "";
  if (artifact.content_type === "text" && typeof artifact.content === "string") return artifact.content;
  return jsonStringify(artifact.content ?? { status: artifact.status, summary: artifact.summary, path: artifact.path });
}

function reviewCountLabel(value: number | undefined): string {
  if (typeof value !== "number") return "0";
  return String(value);
}

function formatReviewError(result: ReviewResult | null, statusCode?: number): string {
  const details = Array.isArray(result?.errors)
    ? result.errors
        ?.map((item) => {
          const path = item.path?.trim();
          const message = item.message?.trim();
          if (path && message) return `${path}: ${message}`;
          return message || path || "";
        })
        .filter(Boolean)
    : [];
  const summary = result?.summary?.trim();
  if (summary && details.length > 0) return `${summary} ${details.join("; ")}`;
  if (summary) return summary;
  if (details.length > 0) return details.join("; ");
  if (statusCode) return `GET /api/review failed with ${statusCode}`;
  return "Unable to load review";
}

function formatStatusError(result: StatusResult | null, statusCode?: number): string {
  const details = Array.isArray(result?.errors)
    ? result?.errors
        ?.map((item) => {
          const path = item.path?.trim();
          const message = item.message?.trim();
          if (path && message) return `${path}: ${message}`;
          return message || path || "";
        })
        .filter(Boolean)
    : [];
  const summary = result?.summary?.trim();
  if (summary && details.length > 0) return `${summary} ${details.join("; ")}`;
  if (summary) return summary;
  if (details.length > 0) return details.join("; ");
  if (statusCode) return `GET /api/status failed with ${statusCode}`;
  return "Unable to load status";
}

function sectionIDsForPage(page: Page): string[] {
  if (page === "status") {
    return ["summary", "next-actions", "warnings", "facts", "artifacts"];
  }
  if (page === "timeline") {
    return ["events"];
  }
  if (page === "review") {
    return ["rounds"];
  }
  return ["overview", "status"];
}

function readSectionFromLocation(page: Page): string {
  const section = window.location.hash.replace(/^#/, "");
  return sectionIDsForPage(page).includes(section) ? section : sectionIDsForPage(page)[0];
}

function sectionsForPage(page: Page, status: {
  nextActions: NextAction[];
  blockers: ErrorDetail[];
  warnings: string[];
  facts: Array<[string, unknown]>;
  artifacts: Array<[string, unknown]>;
}, timeline: {
  events: TimelineEvent[];
  artifacts: Array<[string, unknown]>;
}): SectionLink[] {
  if (page === "status") {
    return [
      { id: "summary", label: "Summary" },
      { id: "next-actions", label: "Next actions", meta: String(status.nextActions.length) },
      { id: "warnings", label: "Warnings", meta: String(status.warnings.length + status.blockers.length) },
      { id: "facts", label: "Facts", meta: String(status.facts.length) },
      { id: "artifacts", label: "Artifacts", meta: String(status.artifacts.length) },
    ];
  }

  if (page === "timeline") {
    return [
      { id: "events", label: "Events", meta: String(timeline.events.length) },
    ];
  }

  return [
    { id: "overview", label: "Overview" },
    { id: "status", label: "Status" },
  ];
}

function App() {
  const [page, setPage] = useState<Page>(() => readPageFromLocation());
  const [section, setSection] = useState<string>(() => readSectionFromLocation(readPageFromLocation()));
  const [status, setStatus] = useState<StatusResult | null>(null);
  const [statusError, setStatusError] = useState<string | null>(null);
  const [statusLoading, setStatusLoading] = useState(false);
  const [timeline, setTimeline] = useState<TimelineResult | null>(null);
  const [timelineError, setTimelineError] = useState<string | null>(null);
  const [timelineLoading, setTimelineLoading] = useState(false);
  const [review, setReview] = useState<ReviewResult | null>(null);
  const [reviewError, setReviewError] = useState<string | null>(null);
  const [reviewLoading, setReviewLoading] = useState(false);

  useEffect(() => {
    const onLocationChange = () => {
      const nextPage = readPageFromLocation();
      setPage(nextPage);
      setSection(readSectionFromLocation(nextPage));
    };
    window.addEventListener("popstate", onLocationChange);
    window.addEventListener("hashchange", onLocationChange);
    return () => {
      window.removeEventListener("popstate", onLocationChange);
      window.removeEventListener("hashchange", onLocationChange);
    };
  }, []);

  const navigateToPage = (nextPage: Page) => {
    const next = pageDefinition(nextPage);
    const nextSection = sectionIDsForPage(nextPage)[0];
    const nextURL = `${next.href}#${nextSection}`;
    if (`${window.location.pathname}${window.location.hash}` !== nextURL) {
      window.history.pushState({}, "", nextURL);
    }
    setPage(nextPage);
    setSection(nextSection);
  };

  const navigateToSection = (nextSection: string) => {
    const nextURL = `${pageDefinition(page).href}#${nextSection}`;
    if (`${window.location.pathname}${window.location.hash}` !== nextURL) {
      window.history.pushState({}, "", nextURL);
    }
    setSection(nextSection);
  };

  useEffect(() => {
    if (pageFromPathname(window.location.pathname) === null && !window.location.hash) {
      window.history.replaceState({}, "", `${pageDefinition(page).href}#${sectionIDsForPage(page)[0]}`);
    }
  }, [page]);

  useEffect(() => {
    if (page !== "status") return;

    const controller = new AbortController();
    setStatusLoading(true);
    setStatusError(null);

    fetch("/api/status", { signal: controller.signal })
      .then(async (response) => {
        const payload = (await response.json()) as StatusResult;
        if (!response.ok || payload.ok === false) {
          throw new Error(formatStatusError(payload, response.status));
        }
        return payload;
      })
      .then((nextStatus) => {
        setStatus(nextStatus);
        setStatusLoading(false);
      })
      .catch((error: unknown) => {
        if (controller.signal.aborted) return;
        setStatus(null);
        setStatusError(error instanceof Error ? error.message : "Unable to load status");
        setStatusLoading(false);
      });

    return () => controller.abort();
  }, [page]);

  useEffect(() => {
    if (page !== "timeline") return;

    const controller = new AbortController();
    setTimelineLoading(true);
    setTimelineError(null);

    fetch("/api/timeline", { signal: controller.signal })
      .then(async (response) => {
        const payload = (await response.json()) as TimelineResult;
        if (!response.ok || payload.ok === false) {
          const summary = payload?.summary?.trim();
          const details = Array.isArray(payload?.errors)
            ? payload.errors
                ?.map((item) => {
                  const path = item.path?.trim();
                  const message = item.message?.trim();
                  if (path && message) return `${path}: ${message}`;
                  return message || path || "";
                })
                .filter(Boolean)
            : [];
          const fallback = summary || (response.status ? `GET /api/timeline failed with ${response.status}` : "Unable to load timeline");
          throw new Error(details.length > 0 ? `${fallback} ${details.join("; ")}` : fallback);
        }
        return payload;
      })
      .then((nextTimeline) => {
        setTimeline(nextTimeline);
        setTimelineLoading(false);
      })
      .catch((error: unknown) => {
        if (controller.signal.aborted) return;
        setTimeline(null);
        setTimelineError(error instanceof Error ? error.message : "Unable to load timeline");
        setTimelineLoading(false);
      });

    return () => controller.abort();
  }, [page]);

  useEffect(() => {
    if (page !== "review") return;

    const controller = new AbortController();
    setReviewLoading(true);
    setReviewError(null);

    fetch("/api/review", { signal: controller.signal })
      .then(async (response) => {
        const payload = (await response.json()) as ReviewResult;
        if (!response.ok || payload.ok === false) {
          throw new Error(formatReviewError(payload, response.status));
        }
        return payload;
      })
      .then((nextReview) => {
        setReview(nextReview);
        setReviewLoading(false);
      })
      .catch((error: unknown) => {
        if (controller.signal.aborted) return;
        setReview(null);
        setReviewError(error instanceof Error ? error.message : "Unable to load review");
        setReviewLoading(false);
      });

    return () => controller.abort();
  }, [page]);

  const activeStatus = useMemo(() => {
    return {
      summary: status?.summary ?? "Waiting for status data.",
      currentNode: status?.state?.current_node ?? "unknown",
      nextActions: Array.isArray(status?.next_actions) ? status?.next_actions ?? [] : [],
      blockers: Array.isArray(status?.blockers) ? status?.blockers ?? [] : [],
      warnings: Array.isArray(status?.warnings) ? status?.warnings ?? [] : [],
      errors: Array.isArray(status?.errors) ? status?.errors ?? [] : [],
      facts: pickEntries(status?.facts),
      artifacts: pickEntries(status?.artifacts),
    };
  }, [status]);

  const activeTimeline = useMemo(() => {
    const events = sortTimelineEvents(Array.isArray(timeline?.events) ? timeline?.events ?? [] : []);
    const artifacts = pickEntries((timeline?.artifacts as Record<string, unknown>) ?? null);
    return {
      events,
      artifacts,
      latestEvent: events.length > 0 ? events[0] : null,
    };
  }, [timeline]);

  const activeReview = useMemo(() => {
    return {
      rounds: Array.isArray(review?.rounds) ? review.rounds ?? [] : [],
      warnings: Array.isArray(review?.warnings) ? review.warnings ?? [] : [],
      artifacts: pickEntries((review?.artifacts as Record<string, unknown>) ?? null),
      summary: review?.summary ?? "Waiting for review data.",
    };
  }, [review]);
  const activeSectionLabel =
    sectionsForPage(page, activeStatus, activeTimeline).find((item) => item.id === section)?.label ?? pageDefinition(page).label;

  return (
    <div class="app-shell">
      <header class="topbar">
        <div class="brand">
          <span class="brand-mark">{productNameLabel()}</span>
        </div>
        <div class="workspace-path" title={workdirLabel()}>{workdirLabel()}</div>
        <div class="topbar-meta">
          <span>read-only</span>
          <span>local</span>
        </div>
      </header>

      <div class="layout">
        <aside class="rail" aria-label="Pages">
          {pages.map((item) => {
            const selected = page === item.id;
            return (
              <a
                key={item.id}
                class={`rail-item${selected ? " is-active" : ""}`}
                href={item.href}
                aria-current={selected ? "page" : undefined}
                aria-label={item.label}
                title={item.label}
                onClick={(event) => {
                  event.preventDefault();
                  navigateToPage(item.id);
                }}
              >
                <span class="rail-icon">
                  <RailIcon page={item.id} />
                </span>
                <span class="sr-only">{item.label}</span>
              </a>
            );
          })}
        </aside>

        {page === "timeline" ? (
          <TimelineWorkspace
            loading={timelineLoading}
            error={timelineError}
            events={activeTimeline.events}
          />
        ) : page === "review" ? (
          <ReviewWorkspace
            loading={reviewLoading}
            error={reviewError}
            summary={activeReview.summary}
            rounds={activeReview.rounds}
            warnings={activeReview.warnings}
            artifacts={activeReview.artifacts}
          />
        ) : (
          <main class="content">
            <aside class="sidebar" aria-label={`${pageDefinition(page).label} sidebar`}>
              <div class="sidebar-header">
                <span class="sidebar-label">Explorer</span>
                <strong>{pageDefinition(page).label}</strong>
              </div>
              <nav class="sidebar-group" aria-label={`${pageDefinition(page).label} sections`}>
                {sectionsForPage(page, activeStatus, activeTimeline).map((item) => (
                  <a
                    key={item.id}
                    class={`sidebar-link${item.id === section ? " is-active" : ""}`}
                    href={`#${item.id}`}
                    onClick={(event) => {
                      event.preventDefault();
                      navigateToSection(item.id);
                    }}
                  >
                    <span>{item.label}</span>
                    {item.meta ? <span class="sidebar-meta">{item.meta}</span> : null}
                  </a>
                ))}
              </nav>
            </aside>

            <section class="editor">
              <section class="page-header">
                <div class="editor-tabs">
                  <div class="editor-tab is-active">{pageDefinition(page).label}</div>
                  <div class="editor-section-label">{activeSectionLabel}</div>
                </div>
                {page === "status" && statusLoading ? <span class="muted">loading</span> : null}
              </section>

              {page === "status" ? (
                <StatusPage
                  loading={statusLoading}
                  error={statusError}
                  summary={activeStatus.summary}
                  currentNode={activeStatus.currentNode}
                  nextActions={activeStatus.nextActions}
                  blockers={activeStatus.blockers}
                  warnings={activeStatus.warnings}
                  errors={Array.isArray(status?.errors) ? status?.errors ?? [] : []}
                  facts={activeStatus.facts}
                  artifacts={activeStatus.artifacts}
                  selectedSection={section}
                />
              ) : null}
            </section>
          </main>
        )}
      </div>
    </div>
  );
}

function StatusPage(props: {
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
}) {
  const { loading, error, summary, currentNode, nextActions, blockers, warnings, errors, facts, artifacts, selectedSection } = props;

  let detailPane = (
    <section id="summary" class="pane">
      <div class="section-head">
        <h2>Summary</h2>
        {loading ? <span class="muted">loading</span> : null}
      </div>
      <div class="detail-copy">{summary}</div>
    </section>
  );

  if (selectedSection === "next-actions") {
    detailPane = (
      <section id="next-actions" class="pane">
        <div class="section-head">
          <h2>Next actions</h2>
          <span class="muted">{nextActions.length}</span>
        </div>
        <ol class="stack-list">
          {nextActions.length > 0 ? (
            nextActions.map((action, index) => (
              <li key={`${action.description}-${index}`}>
                <div class="list-title">{action.description}</div>
                {action.command ? <code>{action.command}</code> : <span class="muted">no command</span>}
              </li>
            ))
          ) : (
            <li class="empty-row">No next actions surfaced yet.</li>
          )}
        </ol>
      </section>
    );
  }

  if (selectedSection === "warnings") {
    detailPane = (
      <section id="warnings" class="pane">
        <div class="section-head">
          <h2>Warnings & blockers</h2>
        </div>
        <div class="stack-list">
          {warnings.length > 0 ? warnings.map((warning, index) => <div key={`warning-${index}`} class="pill pill-warn">{warning}</div>) : <div class="empty-row">No warnings.</div>}
          {blockers.length > 0 ? (
            blockers.map((blocker, index) => (
              <div key={`${blocker.path}-${index}`} class="pill pill-blocker">
                <strong>{blocker.path}</strong>
                <span>{blocker.message}</span>
              </div>
            ))
          ) : (
            <div class="empty-row">No blockers.</div>
          )}
          {errors.length > 0 ? (
            errors.map((item, index) => (
              <div key={`${item.path}-${index}`} class="pill pill-blocker">
                <strong>{item.path}</strong>
                <span>{item.message}</span>
              </div>
            ))
          ) : null}
        </div>
      </section>
    );
  }

  if (selectedSection === "facts") {
    detailPane = (
      <section id="facts" class="pane">
        <div class="section-head">
          <h2>Facts</h2>
          <span class="muted">{facts.length}</span>
        </div>
        <dl class="kv-list">
          {facts.length > 0 ? (
            facts.map(([key, value]) => (
              <div key={key}>
                <dt>{key}</dt>
                <dd>{formatValue(value)}</dd>
              </div>
            ))
          ) : (
            <div class="empty-row">No facts available.</div>
          )}
        </dl>
      </section>
    );
  }

  if (selectedSection === "artifacts") {
    detailPane = (
      <section id="artifacts" class="pane">
        <div class="section-head">
          <h2>Artifacts</h2>
          <span class="muted">{artifacts.length}</span>
        </div>
        <dl class="kv-list">
          {artifacts.length > 0 ? (
            artifacts.map(([key, value]) => (
              <div key={key}>
                <dt>{key}</dt>
                <dd>{formatValue(value)}</dd>
              </div>
            ))
          ) : (
            <div class="empty-row">No artifacts available.</div>
          )}
        </dl>
      </section>
    );
  }

  return (
    <section class="workspace">
      <div class="workspace-inner">
        <section class="status-grid" aria-label="Status overview">
          <div class="status-block">
            <span class="label">current node</span>
            <strong>{currentNode}</strong>
          </div>
          <div class="status-block">
            <span class="label">next actions</span>
            <strong>{nextActions.length}</strong>
          </div>
          <div class="status-block">
            <span class="label">warnings</span>
            <strong>{warnings.length}</strong>
          </div>
          <div class="status-block">
            <span class="label">blockers</span>
            <strong>{blockers.length}</strong>
          </div>
        </section>

        {error ? <div class="notice notice-error">{error}</div> : null}

        {detailPane}
      </div>
    </section>
  );
}

function TimelineWorkspace(props: {
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
    setSelectedTab((current) => {
      if (timelineTabs.some((tab) => tab.id === current)) {
        return current;
      }
      return timelineTabs[0].id;
    });
  }, [timelineTabs, selectedEvent?.event_id]);

  const selectedTabValue =
    timelineTabs.find((tab) => tab.id === selectedTab)?.value ?? timelineTabs[0]?.value ?? selectedEvent ?? null;
  const selectedTabMode = timelineTabs.find((tab) => tab.id === selectedTab)?.mode ?? timelineTabs[0]?.mode ?? "json";
  const transitionLabel =
    selectedEvent && (selectedEvent.from_node || selectedEvent.to_node)
      ? `${selectedEvent.from_node || "unknown"} → ${selectedEvent.to_node || "unknown"}`
      : null;

  return (
    <section class="timeline-shell">
      <aside class="timeline-nav" aria-label="Timeline events">
        <div class="timeline-nav-header">
          <span class="sidebar-label">Explorer</span>
          <strong>Timeline</strong>
          <span class="timeline-nav-meta">{sortedEvents.length}</span>
        </div>
        <div class="timeline-nav-list">
          {sortedEvents.length > 0 ? (
            sortedEvents.map((event) => {
              const selected = event.event_id === selectedEvent?.event_id;
              return (
                <button
                  key={event.event_id}
                  class={`timeline-stream-item${selected ? " is-active" : ""}`}
                  type="button"
                  onClick={() => setSelectedEventId(event.event_id)}
                  aria-pressed={selected}
                >
                  <div class="timeline-stream-row">
                    <div class="timeline-stream-title">{timelineEventTitle(event)}</div>
                    <div class="timeline-stream-meta">
                      <span>{formatTimestamp(event.recorded_at)}</span>
                    </div>
                  </div>
                  <div class="timeline-stream-subtitle">{timelineEventSubtitle(event)}</div>
                </button>
              );
            })
          ) : (
            <div class="empty-row">No timeline events recorded yet for this plan.</div>
          )}
        </div>
      </aside>

      <section class="editor timeline-editor">
        <section class="page-header">
          <div class="editor-tabs">
            <div class="editor-tab is-active">Timeline</div>
            <div class="editor-section-label">{selectedEvent ? timelineEventTitle(selectedEvent) : "Events"}</div>
          </div>
          {loading ? <span class="muted">loading</span> : null}
        </section>

        <section class="workspace workspace-timeline">
          {error ? <div class="notice notice-error">{error}</div> : null}

          <section class="timeline-inspector" aria-label="Selected event details">
            <div class="timeline-inspector-tabs" role="tablist" aria-label="Timeline event payloads">
              {timelineTabs.map((tab) => (
                <button
                  key={tab.id}
                  type="button"
                  class={`timeline-inspector-tab${selectedTab === tab.id ? " is-active" : ""}`}
                  onClick={() => setSelectedTab(tab.id)}
                  role="tab"
                  aria-selected={selectedTab === tab.id}
                >
                  {tab.label}
                </button>
              ))}
            </div>

            <div class="timeline-inspector-body">
              {selectedEvent ? (
                <>
                  <div class="timeline-inspector-head">
                    <div>
                      <div class="timeline-inspector-command">{timelineEventTitle(selectedEvent)}</div>
                      {transitionLabel ? <div class="timeline-inspector-transition">{transitionLabel}</div> : null}
                      <div class="timeline-inspector-subtitle">{selectedEvent.summary}</div>
                    </div>
                    <div class="timeline-inspector-refs">
                      <span>{selectedEvent.event_id}</span>
                      <span>{formatTimestamp(selectedEvent.recorded_at)}</span>
                    </div>
                  </div>

                  <pre class="timeline-json" aria-label={`${selectedTab} payload`}>
                    {timelineTabText(selectedTabValue, selectedTabMode)}
                  </pre>
                </>
              ) : (
                <div class="empty-row">Select an event to inspect its raw payload.</div>
              )}
            </div>
          </section>
        </section>
      </section>
    </section>
  );
}

function ReviewWorkspace(props: {
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
  const selectedRound = useMemo(() => {
    if (rounds.length === 0) return null;
    if (selectedRoundId) {
      const found = rounds.find((round) => round.round_id === selectedRoundId);
      if (found) return found;
    }
    return rounds[0];
  }, [rounds, selectedRoundId]);
  const [selectedArtifactKey, setSelectedArtifactKey] = useState<string | null>(null);
  const [supportExpanded, setSupportExpanded] = useState(false);
  const reviewers = Array.isArray(selectedRound?.reviewers) ? selectedRound.reviewers ?? [] : [];
  const supportArtifacts = Array.isArray(selectedRound?.artifacts) ? selectedRound.artifacts ?? [] : [];
  const selectedReviewer = useMemo(() => {
    if (reviewers.length === 0 || selectedDetailTab === "summary") return null;
    const found = reviewers.find((reviewer) => reviewer.slot === selectedDetailTab);
    if (found) {
      return found;
    }
    return null;
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
    setSelectedRoundId((current) => {
      if (current && rounds.some((round) => round.round_id === current)) {
        return current;
      }
      return rounds[0]?.round_id ?? null;
    });
  }, [rounds]);

  useEffect(() => {
    setSelectedDetailTab("summary");
  }, [selectedRound?.round_id]);

  useEffect(() => {
    setSelectedDetailTab((current) => {
      if (current === "summary") {
        return "summary";
      }
      if (reviewers.some((reviewer) => reviewer.slot === current)) {
        return current;
      }
      return reviewers[0]?.slot ?? "summary";
    });
  }, [reviewers]);

  useEffect(() => {
    if (supportArtifacts.length === 0) {
      setSelectedArtifactKey(null);
      return;
    }
    setSelectedArtifactKey((current) => {
      if (current && supportArtifacts.some((artifact, index) => reviewArtifactKey(artifact, index) === current)) {
        return current;
      }
      return reviewArtifactKey(supportArtifacts[0], 0);
    });
  }, [supportArtifacts, selectedRound?.round_id]);

  useEffect(() => {
    setSupportExpanded(false);
  }, [selectedRound?.round_id]);

  return (
    <section class="review-shell">
      <aside class="review-nav" aria-label="Review rounds">
        <div class="timeline-nav-header">
          <span class="sidebar-label">Explorer</span>
          <strong>Review rounds</strong>
          <span class="timeline-nav-meta">{rounds.length}</span>
        </div>
        <div class="timeline-nav-list">
          {rounds.length > 0 ? (
            rounds.map((round) => {
              const selected = round.round_id === selectedRound?.round_id;
              return (
                <button
                  key={round.round_id}
                  class={`review-round-item is-${reviewRoundStatusTone(round)}${selected ? " is-active" : ""}`}
                  type="button"
                  onClick={() => setSelectedRoundId(round.round_id)}
                  aria-pressed={selected}
                  aria-label={reviewRoundAriaLabel(round)}
                >
                  <div class="review-round-main">
                    <div class="review-round-row">
                      <div class="review-round-title">{reviewRoundTitle(round)}</div>
                      <span class={`review-round-indicator is-${reviewRoundStatusTone(round)}`} aria-hidden="true" title={reviewRoundStatusLabel(round)} />
                    </div>
                    <div class={`review-round-id is-${reviewRoundStatusTone(round)}`}>
                      <span class="review-round-id-text">{reviewRoundCompactMeta(round)}</span>
                      <span class="review-round-status-text">{reviewRoundCompactStatusLabel(round)}</span>
                    </div>
                    <div class="review-round-subtitle">{reviewRoundSubtitle(round)}</div>
                  </div>
                  <div class="review-round-meta">
                    <span>{round.created_at || round.updated_at || round.aggregated_at ? formatTimestamp(round.created_at ?? round.updated_at ?? round.aggregated_at ?? "") : "time unknown"}</span>
                    <span>
                      {reviewCountLabel(round.submitted_slots)}/{reviewCountLabel(round.total_slots)} submitted
                    </span>
                  </div>
                </button>
              );
            })
          ) : (
            <div class="empty-row">{summary || "No review rounds recorded yet for the current plan."}</div>
          )}
        </div>
      </aside>

      <section class="editor timeline-editor">
        <section class="page-header">
          <div class="editor-tabs">
            <div class="editor-tab is-active">Review</div>
            <div class="editor-section-label">{selectedRound ? reviewRoundTitle(selectedRound) : "Rounds"}</div>
          </div>
          {loading ? <span class="muted">loading</span> : null}
        </section>

        <section class="workspace workspace-review">
          {error ? <div class="notice notice-error">{error}</div> : null}
          {warnings.map((warning) => (
            <div key={warning} class="notice notice-warning">
              {warning}
            </div>
          ))}

          {selectedRound ? (
            <div class="review-body">
              <section class="review-section review-content-pane" aria-label="Review content">
                <div class="timeline-inspector-tabs review-detail-tabs" role="tablist" aria-label="Review content tabs">
                  <button
                    type="button"
                    class={`timeline-inspector-tab${selectedDetailTab === "summary" ? " is-active" : ""}`}
                    onClick={() => setSelectedDetailTab("summary")}
                    role="tab"
                    aria-selected={selectedDetailTab === "summary"}
                  >
                    Summary
                  </button>
                  {reviewers.map((reviewer) => (
                    <button
                      key={reviewer.slot}
                      type="button"
                      class={`timeline-inspector-tab${selectedDetailTab === reviewer.slot ? " is-active" : ""}`}
                      onClick={() => setSelectedDetailTab(reviewer.slot)}
                      role="tab"
                      aria-selected={selectedDetailTab === reviewer.slot}
                    >
                      {reviewReviewerLabel(reviewer)}
                    </button>
                  ))}
                </div>

                <div class="timeline-inspector-body review-detail-body">
                  {selectedDetailTab === "summary" ? (
                  <div class="review-tab-panel">
                    {selectedRoundWarnings.length > 0 ? (
                      <section class="review-subsection">
                        <div class="section-head">
                          <h3>Warnings</h3>
                          <span class="muted">{selectedRoundWarnings.length}</span>
                        </div>
                        <div class="review-warning-list">
                          {selectedRoundWarnings.map((warning) => (
                            <div key={warning} class="review-warning-item">
                              {warning}
                            </div>
                          ))}
                        </div>
                      </section>
                    ) : null}

                    <section class="review-subsection">
                      <div class="section-head">
                        <h3>Overview</h3>
                        <span class="muted">{selectedRound.round_id}</span>
                      </div>
                      <div class="review-summary-panel">
                        <div class="review-summary-headline">
                          <div>
                            <div class="review-overview-title">{reviewRoundTitle(selectedRound)}</div>
                            <div class="review-overview-subtitle">{reviewRoundListLabel(selectedRound)}</div>
                          </div>
                          <div class="review-badges">
                            {selectedRound.is_active ? <span class="status-badge is-muted">Active</span> : null}
                            {selectedRound.kind ? <span class="status-badge is-muted">{humanizeLabel(selectedRound.kind)}</span> : null}
                            <span class={`status-badge is-${reviewRoundStatusTone(selectedRound)}`}>{reviewRoundStatusLabel(selectedRound)}</span>
                          </div>
                        </div>
                        <p class="detail-copy">{selectedRound.status_summary || summary}</p>
                        <section class="status-grid review-status-grid" aria-label="Review round summary">
                          <div class="status-block">
                            <span class="label">decision</span>
                            <strong>{selectedRound.decision ? humanizeLabel(selectedRound.decision) : reviewRoundStatusLabel(selectedRound)}</strong>
                          </div>
                          <div class="status-block">
                            <span class="label">progress</span>
                            <strong>
                              {reviewCountLabel(selectedRound.submitted_slots)}/{reviewCountLabel(selectedRound.total_slots)} submitted
                            </strong>
                          </div>
                          <div class="status-block">
                            <span class="label">revision</span>
                            <strong>{selectedRound.revision ? `rev ${selectedRound.revision}` : "unknown"}</strong>
                          </div>
                          <div class="status-block">
                            <span class="label">updated</span>
                            <strong>{formatTimestamp(selectedRound.aggregated_at || selectedRound.updated_at || selectedRound.created_at || "unknown")}</strong>
                          </div>
                        </section>
                        <dl class="kv-list">
                          <div>
                            <dt>Kind</dt>
                            <dd>{selectedRound.kind ? humanizeLabel(selectedRound.kind) : "unknown"}</dd>
                          </div>
                          <div>
                            <dt>Target</dt>
                            <dd>{typeof selectedRound.step === "number" ? `Step ${selectedRound.step}` : selectedRound.review_title || "Finalize / unscoped"}</dd>
                          </div>
                          <div>
                            <dt>Created</dt>
                            <dd>{formatTimestamp(selectedRound.created_at || "unknown")}</dd>
                          </div>
                        </dl>
                      </div>
                    </section>

                    <section class="review-subsection">
                      <div class="section-head">
                        <h3>Blocking findings</h3>
                        <span class="muted">{blockingFindings.length}</span>
                      </div>
                      {blockingFindings.length > 0 ? (
                        <div class="review-finding-list">
                          {blockingFindings.map((finding, index) => (
                            <ReviewFindingCard
                              key={reviewFindingKey(finding, index)}
                              finding={finding}
                              provenance={reviewAggregateFindingSource(finding)}
                            />
                          ))}
                        </div>
                      ) : (
                        <div class="empty-row">No blocking findings recorded.</div>
                      )}
                    </section>

                    <section class="review-subsection">
                      <div class="section-head">
                        <h3>Non-blocking findings</h3>
                        <span class="muted">{nonBlockingFindings.length}</span>
                      </div>
                      {nonBlockingFindings.length > 0 ? (
                        <div class="review-finding-list">
                          {nonBlockingFindings.map((finding, index) => (
                            <ReviewFindingCard
                              key={reviewFindingKey(finding, index)}
                              finding={finding}
                              provenance={reviewAggregateFindingSource(finding)}
                            />
                          ))}
                        </div>
                      ) : (
                        <div class="empty-row">No non-blocking findings recorded.</div>
                      )}
                    </section>
                  </div>
                ) : selectedReviewer ? (
                  <div class="reviewer-panel">
                    <div class="reviewer-panel-head">
                      <div>
                        <div class="review-overview-title">{reviewReviewerLabel(selectedReviewer)}</div>
                        <div class="review-overview-subtitle">{reviewReviewerStatusLabel(selectedReviewer)}</div>
                      </div>
                      <div class="review-badges">
                        <span class={`status-badge is-${reviewReviewerStatusTone(selectedReviewer)}`}>
                          {reviewReviewerStatusLabel(selectedReviewer)}
                        </span>
                        {selectedReviewer.submitted_at ? <span class="status-badge is-muted">{formatTimestamp(selectedReviewer.submitted_at)}</span> : null}
                      </div>
                    </div>

                    <div class="review-context-strip" aria-label="Selected round context">
                      <div class="review-context-item">
                        <span class="label">round</span>
                        <strong>{selectedRound.round_id}</strong>
                      </div>
                      <div class="review-context-item">
                        <span class="label">decision</span>
                        <strong>{selectedRound.decision ? humanizeLabel(selectedRound.decision) : reviewRoundStatusLabel(selectedRound)}</strong>
                      </div>
                      <div class="review-context-item">
                        <span class="label">blocking</span>
                        <strong>{blockingFindings.length}</strong>
                      </div>
                      <div class="review-context-item">
                        <span class="label">warnings</span>
                        <strong>{selectedRoundWarnings.length}</strong>
                      </div>
                    </div>

                    <div class="review-tab-panel reviewer-tab-panel">
                      <details class="review-fold review-fold-task" open>
                        <summary class="review-fold-summary">
                          <span>Assigned task</span>
                          <span class="muted">{selectedReviewer.instructions?.trim() ? "available" : "missing"}</span>
                        </summary>
                        <div class="review-fold-body">
                          {selectedReviewer.instructions?.trim() ? (
                            <p class="detail-copy">{selectedReviewer.instructions}</p>
                          ) : (
                            <div class="empty-row">Instructions are unavailable for this reviewer slot.</div>
                          )}
                        </div>
                      </details>

                      <details class="review-fold review-fold-result" open>
                        <summary class="review-fold-summary">
                          <span>Returned result</span>
                          <span class="muted">
                            {selectedReviewer.summary?.trim()
                              ? `${Array.isArray(selectedReviewer.findings) ? selectedReviewer.findings.length : 0} finding(s)`
                              : reviewReviewerStatusLabel(selectedReviewer)}
                          </span>
                        </summary>
                        <div class="review-fold-body">
                          {selectedReviewer.summary?.trim() ? (
                            <>
                              <p class="detail-copy">{selectedReviewer.summary}</p>
                              <div class="review-finding-list">
                                {Array.isArray(selectedReviewer.findings) && selectedReviewer.findings.length > 0 ? (
                                  selectedReviewer.findings.map((finding, index) => (
                                    <ReviewFindingCard key={reviewFindingKey(finding, index)} finding={finding} />
                                  ))
                                ) : (
                                  <div class="empty-row">No findings recorded for this reviewer.</div>
                                )}
                              </div>
                            </>
                          ) : (
                            <div class="empty-row">This reviewer has not submitted a result yet.</div>
                          )}
                        </div>
                      </details>

                      {Array.isArray(selectedReviewer.warnings) && selectedReviewer.warnings.length > 0 ? (
                        <div class="review-warning-list">
                          {selectedReviewer.warnings.map((warning) => (
                            <div key={warning} class="review-warning-item">
                              {warning}
                            </div>
                          ))}
                        </div>
                      ) : null}
                    </div>
                  </div>
                  ) : (
                    <div class="empty-row">No reviewer slots are available for this round.</div>
                  )}

                  {supportArtifacts.length > 0 || artifacts.length > 0 ? (
                    <section class="review-support-wrap" aria-label="Supporting evidence">
                      <button
                        type="button"
                        class={`review-support-toggle${supportExpanded ? " is-open" : ""}`}
                        onClick={() => setSupportExpanded((current) => !current)}
                        aria-expanded={supportExpanded}
                      >
                        <span>Supporting evidence</span>
                        <span class="muted">{supportArtifacts.length + artifacts.length}</span>
                      </button>
                      <p class="supporting-copy">Use raw artifacts and round metadata only when you need to debug damaged or incomplete review state.</p>
                      {supportExpanded ? (
                        <div class="review-support-stack">
                          <section class="review-section secondary-pane review-support-section" aria-label="Supporting artifacts">
                            <div class="section-head">
                              <h2>Artifact payloads</h2>
                              <span class="muted">{supportArtifacts.length}</span>
                            </div>
                            {supportArtifacts.length > 0 ? (
                              <>
                                <div class="reviewer-tabs" role="tablist" aria-label="Supporting artifacts">
                                  {supportArtifacts.map((artifact, index) => {
                                    const artifactKey = reviewArtifactKey(artifact, index);
                                    const label = reviewArtifactLabel(artifact);
                                    return (
                                      <button
                                        key={artifactKey}
                                        type="button"
                                        class={`timeline-inspector-tab${selectedArtifactKey === artifactKey ? " is-active" : ""}`}
                                        onClick={() => setSelectedArtifactKey(artifactKey)}
                                        role="tab"
                                        aria-selected={selectedArtifactKey === artifactKey}
                                      >
                                        {label}
                                      </button>
                                    );
                                  })}
                                </div>

                                {selectedArtifact ? (
                                  <div class="review-artifact-panel">
                                    <div class="review-artifact-meta">
                                      <span class={`status-badge is-${selectedArtifact.status === "available" ? "good" : selectedArtifact.status === "invalid" ? "danger" : "warning"}`}>
                                        {humanizeLabel(selectedArtifact.status || "unknown")}
                                      </span>
                                      {selectedArtifact.path ? <span class="muted">{selectedArtifact.path}</span> : null}
                                    </div>
                                    {selectedArtifact.summary ? <p class="review-artifact-summary">{selectedArtifact.summary}</p> : null}
                                    <pre class="timeline-json">{reviewArtifactText(selectedArtifact)}</pre>
                                  </div>
                                ) : null}
                              </>
                            ) : (
                              <div class="empty-row">No supporting artifacts available for this round.</div>
                            )}
                          </section>

                          {artifacts.length > 0 ? (
                            <section class="review-section secondary-pane review-support-section">
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
              </section>
            </div>
          ) : (
            <div class="workspace-inner">
              <div class="empty-row">{summary || "No review rounds recorded yet for the current plan."}</div>
            </div>
          )}
        </section>
      </section>
    </section>
  );
}

render(<App />, document.getElementById("app") as HTMLElement);
