export type Page = "status" | "plan" | "timeline" | "review";

export type PageDef = { id: Page; label: string; href: string };

export type SectionLink = { id: string; label: string; meta?: string };

export type NextAction = {
  command: string | null;
  description: string;
};

export type ErrorDetail = {
  path: string;
  message: string;
};

export type StatusResult = {
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

export type PlanHeading = {
  id: string;
  label: string;
  level: number;
  anchor: string;
  children?: PlanHeading[] | null;
};

export type PlanPreview = {
  status: string;
  content_type?: string;
  content?: string;
  reason?: string;
  byte_size?: number;
  extension?: string;
};

export type PlanNode = {
  id: string;
  kind: "directory" | "file";
  label: string;
  path?: string;
  children?: PlanNode[] | null;
  preview?: PlanPreview | null;
};

export type PlanDocument = {
  title: string;
  path: string;
  markdown: string;
  headings: PlanHeading[];
};

export type PlanResult = {
  ok: boolean;
  resource: string;
  summary: string;
  artifacts?: {
    plan_path?: string;
    supplements_path?: string;
    local_state_path?: string;
  } | null;
  document?: PlanDocument | null;
  supplements?: PlanNode | null;
  warnings?: string[] | null;
  errors?: ErrorDetail[] | null;
};

export type TimelineDetail = {
  key: string;
  value: string;
};

export type TimelineArtifactRef = {
  label: string;
  value: string;
  path?: string;
  content_type?: string;
  content?: unknown;
};

export type TimelineEvent = {
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

export type TimelineResult = {
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

export type ReviewArtifact = {
  label: string;
  path?: string;
  status?: string;
  summary?: string;
  content_type?: string;
  content?: unknown;
};

export type ReviewFinding = {
  severity: string;
  title: string;
  details: string;
  locations?: string[] | null;
};

export type ReviewAggregateFinding = ReviewFinding & {
  slot?: string;
  dimension?: string;
};

export type ReviewReviewer = {
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

export type ReviewRound = {
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

export type ReviewResult = {
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
