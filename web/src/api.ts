import { FACILITY_API_BASE } from "./config";

/** RFC 7807 problem+json body every error response from facility-layout
 *  returns, per CLAUDE.md's "go straight to RFC 7807" mandate. */
export interface ProblemDetails {
  type: string;
  title: string;
  status: number;
  detail: string;
  instance?: string;
}

export class ApiError extends Error {
  problem: ProblemDetails | null;
  status: number;

  constructor(status: number, problem: ProblemDetails | null, fallbackMessage: string) {
    super(problem?.detail || problem?.title || fallbackMessage);
    this.status = status;
    this.problem = problem;
  }
}

/**
 * POST/PATCH-style JSON call shared by every config form on this screen.
 * Parses an RFC 7807 problem+json body on failure so forms can surface the
 * exact domain-error detail (e.g. "placement-rule-violated: ...") instead
 * of a generic "request failed".
 */
export async function apiPost<TResponse>(
  path: string,
  body: unknown,
): Promise<TResponse> {
  const res = await fetch(`${FACILITY_API_BASE}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    let problem: ProblemDetails | null = null;
    try {
      problem = (await res.json()) as ProblemDetails;
    } catch {
      // non-JSON error body -- fall through with problem = null
    }
    throw new ApiError(res.status, problem, `${res.status} ${res.statusText}`);
  }
  if (res.status === 204) return undefined as TResponse;
  return (await res.json()) as TResponse;
}
