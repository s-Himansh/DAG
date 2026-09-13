const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";

export interface TaskDefinition {
  id: string;
  execute: string;
  dependencies?: string[];
}

export interface TaskStatus {
  id: string;
  state: string;
  value?: unknown;
  error?: string;
  started_at?: string;
  ended_at?: string;
}

export interface Workflow {
  id: string;
  name: string;
  status: "pending" | "running" | "completed" | "failed" | "cancelled";
  tasks: TaskDefinition[];
  task_statuses?: Record<string, TaskStatus>;
  error?: string;
  created_at: string;
  started_at?: string;
  ended_at?: string;
}

export interface SubmitWorkflowRequest {
  name: string;
  tasks: TaskDefinition[];
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  });

  if (!res.ok) {
    const error = await res.text();
    throw new Error(error || `HTTP ${res.status}`);
  }

  return res.json();
}

export const api = {
  health: () => request<{ status: string }>("/api/health"),

  submitWorkflow: (data: SubmitWorkflowRequest) =>
    request<Workflow>("/api/workflows", {
      method: "POST",
      body: JSON.stringify(data),
    }),

  listWorkflows: () => request<Workflow[]>("/api/workflows"),

  getWorkflow: (id: string) => request<Workflow>(`/api/workflows/${id}`),

  getWorkflowTasks: (id: string) =>
    request<Record<string, TaskStatus>>(`/api/workflows/${id}/tasks`),

  cancelWorkflow: (id: string) =>
    request<{ status: string }>(`/api/workflows/${id}/cancel`, {
      method: "POST",
    }),
};
