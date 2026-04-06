import { render } from "preact";
import { useEffect, useMemo, useState } from "preact/hooks";

import "./styles.css";

import {
  formatReviewError,
  formatStatusError,
  formatTimelineError,
  pickEntries,
  productNameLabel,
  workdirLabel,
} from "./helpers";
import { ReviewWorkspace, StatusWorkspace, TimelineWorkspace } from "./pages";
import type { Page, PageDef, ReviewResult, StatusResult, TimelineResult } from "./types";
import { RailIcon, TopbarMetric } from "./workbench";

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

function sectionIDsForPage(page: Page): string[] {
  if (page === "status") {
    return ["summary", "next-actions", "warnings", "facts", "artifacts"];
  }
  return ["overview"];
}

function readSectionFromLocation(page: Page): string {
  const section = window.location.hash.replace(/^#/, "");
  return sectionIDsForPage(page).includes(section) ? section : sectionIDsForPage(page)[0];
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

  useEffect(() => {
    if (pageFromPathname(window.location.pathname) === null && !window.location.hash) {
      window.history.replaceState({}, "", `${pageDefinition(page).href}#${sectionIDsForPage(page)[0]}`);
    }
  }, [page]);

  const navigateToPage = (nextPage: Page, nextSection = sectionIDsForPage(nextPage)[0]) => {
    const nextURL = `${pageDefinition(nextPage).href}#${nextSection}`;
    if (`${window.location.pathname}${window.location.hash}` !== nextURL) {
      window.history.pushState({}, "", nextURL);
    }
    setPage(nextPage);
    setSection(nextSection);
  };

  const navigateToSection = (nextSection: string) => {
    navigateToPage(page, nextSection);
  };

  useEffect(() => {
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
  }, []);

  useEffect(() => {
    if (page !== "timeline") return;

    const controller = new AbortController();
    setTimelineLoading(true);
    setTimelineError(null);

    fetch("/api/timeline", { signal: controller.signal })
      .then(async (response) => {
        const payload = (await response.json()) as TimelineResult;
        if (!response.ok || payload.ok === false) {
          throw new Error(formatTimelineError(payload?.summary, payload?.errors, response.status));
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

  const activeStatus = useMemo(
    () => ({
      summary: status?.summary ?? "Waiting for status data.",
      currentNode: status?.state?.current_node ?? "unknown",
      nextActions: Array.isArray(status?.next_actions) ? status.next_actions ?? [] : [],
      blockers: Array.isArray(status?.blockers) ? status.blockers ?? [] : [],
      warnings: Array.isArray(status?.warnings) ? status.warnings ?? [] : [],
      errors: Array.isArray(status?.errors) ? status.errors ?? [] : [],
      facts: pickEntries(status?.facts),
      artifacts: pickEntries(status?.artifacts),
    }),
    [status],
  );

  const activeTimeline = useMemo(
    () => ({
      events: Array.isArray(timeline?.events) ? timeline.events ?? [] : [],
    }),
    [timeline],
  );

  const activeReview = useMemo(
    () => ({
      rounds: Array.isArray(review?.rounds) ? review.rounds ?? [] : [],
      warnings: Array.isArray(review?.warnings) ? review.warnings ?? [] : [],
      artifacts: pickEntries((review?.artifacts as Record<string, unknown>) ?? null),
      summary: review?.summary ?? "Waiting for review data.",
    }),
    [review],
  );

  return (
    <div class="app-shell">
      <header class="topbar">
        <div class="brand">
          <span class="brand-mark">{productNameLabel()}</span>
        </div>
        <div class="workspace-path" title={workdirLabel()}>
          {workdirLabel()}
        </div>
        <div class="topbar-summary">
          <TopbarMetric kind="node" label="Node" value={activeStatus.currentNode} onClick={() => navigateToPage("status", "summary")} />
          {activeStatus.blockers.length > 0 ? (
            <TopbarMetric
              kind="blockers"
              label="Blockers"
              value={String(activeStatus.blockers.length)}
              tone="danger"
              onClick={() => navigateToPage("status", "warnings")}
            />
          ) : null}
          <TopbarMetric
            kind="warnings"
            label="Warnings"
            value={String(activeStatus.warnings.length)}
            tone={activeStatus.warnings.length > 0 ? "warning" : "muted"}
            onClick={() => navigateToPage("status", "warnings")}
          />
          <TopbarMetric
            kind="actions"
            label="Actions"
            value={String(activeStatus.nextActions.length)}
            tone={activeStatus.nextActions.length > 0 ? "good" : "muted"}
            onClick={() => navigateToPage("status", "next-actions")}
          />
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

        <main class="main-stage">
          {page === "timeline" ? (
            <TimelineWorkspace loading={timelineLoading} error={timelineError} events={activeTimeline.events} />
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
            <StatusWorkspace
              loading={statusLoading}
              error={statusError}
              summary={activeStatus.summary}
              currentNode={activeStatus.currentNode}
              nextActions={activeStatus.nextActions}
              blockers={activeStatus.blockers}
              warnings={activeStatus.warnings}
              errors={activeStatus.errors}
              facts={activeStatus.facts}
              artifacts={activeStatus.artifacts}
              selectedSection={section}
              onSelectSection={navigateToSection}
            />
          )}
        </main>
      </div>
    </div>
  );
}

render(<App />, document.getElementById("app") as HTMLElement);
