"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api, Workflow } from "@/lib/api";

const statusConfig: Record<string, { badge: string; icon: string; color: string }> = {
  pending: { badge: "badge-pending", icon: "⏳", color: "from-amber-400 to-orange-500" },
  running: { badge: "badge-running", icon: "⚡", color: "from-blue-400 to-indigo-500" },
  completed: { badge: "badge-completed", icon: "✓", color: "from-emerald-400 to-teal-500" },
  failed: { badge: "badge-failed", icon: "✗", color: "from-red-400 to-rose-500" },
  cancelled: { badge: "badge-cancelled", icon: "○", color: "from-slate-400 to-slate-500" },
};

export default function Dashboard() {
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchWorkflows = async () => {
    try {
      const data = await api.listWorkflows();
      setWorkflows(data);
      setError(null);
    } catch {
      setError("Failed to connect to API. Is the server running on port 8080?");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchWorkflows();
    const interval = setInterval(fetchWorkflows, 2000);
    return () => clearInterval(interval);
  }, []);

  const stats = [
    { label: "Total Workflows", value: workflows.length, gradient: "from-indigo-500 via-purple-500 to-pink-500", icon: (
      <svg className="w-6 h-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z" /></svg>
    )},
    { label: "Running", value: workflows.filter(w => w.status === "running").length, gradient: "from-blue-500 via-cyan-500 to-teal-400", icon: (
      <svg className="w-6 h-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" /></svg>
    )},
    { label: "Completed", value: workflows.filter(w => w.status === "completed").length, gradient: "from-emerald-500 via-green-500 to-lime-400", icon: (
      <svg className="w-6 h-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
    )},
    { label: "Failed", value: workflows.filter(w => w.status === "failed").length, gradient: "from-red-500 via-rose-500 to-pink-500", icon: (
      <svg className="w-6 h-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
    )},
  ];

  return (
    <div className="min-h-screen relative">
      {/* Animated Background */}
      <div className="bg-blob bg-blob-1"></div>
      <div className="bg-blob bg-blob-2"></div>
      <div className="bg-blob bg-blob-3"></div>

      {/* Header */}
      <header className="sticky top-0 z-50 glass-static" style={{ borderRadius: 0, borderTop: "none", borderLeft: "none", borderRight: "none" }}>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-5">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="relative">
                <div className="w-12 h-12 rounded-2xl bg-gradient-to-br from-indigo-500 via-purple-500 to-pink-500 flex items-center justify-center shadow-lg shadow-indigo-500/30 animate-float">
                  <svg className="w-7 h-7 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z" />
                  </svg>
                </div>
                <div className="absolute -bottom-1 -right-1 w-4 h-4 rounded-full bg-emerald-400 border-2 border-white animate-pulse"></div>
              </div>
              <div>
                <h1 className="text-2xl font-extrabold bg-gradient-to-r from-indigo-600 via-purple-600 to-pink-600 bg-clip-text text-transparent">
                  DAG Orchestrator
                </h1>
                <p className="text-sm text-slate-500 font-medium">Concurrent Workflow Engine</p>
              </div>
            </div>
            <Link
              href="/workflows/new"
              className="btn-gradient flex items-center gap-2"
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M12 4v16m8-8H4" />
              </svg>
              New Workflow
            </Link>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-10 relative z-10">
        {/* Stats Grid */}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-5 mb-10">
          {stats.map((stat, i) => (
            <div
              key={stat.label}
              className="glass p-5 animate-fade-in-up group"
              style={{ animationDelay: `${i * 100}ms` }}
            >
              <div className="flex items-center gap-4">
                <div className={`w-14 h-14 rounded-2xl bg-gradient-to-br ${stat.gradient} flex items-center justify-center shadow-lg group-hover:scale-110 transition-transform duration-300`}>
                  {stat.icon}
                </div>
                <div>
                  <p className="text-3xl font-extrabold text-slate-900 animate-count-up" style={{ animationDelay: `${i * 100 + 200}ms` }}>
                    {stat.value}
                  </p>
                  <p className="text-sm text-slate-500 font-medium">{stat.label}</p>
                </div>
              </div>
            </div>
          ))}
        </div>

        {error && (
          <div className="mb-8 p-5 glass border-l-4 border-red-400 bg-red-50/50 text-red-700 animate-fade-in-up flex items-center gap-4 rounded-2xl">
            <div className="w-10 h-10 rounded-xl bg-red-100 flex items-center justify-center flex-shrink-0">
              <svg className="w-5 h-5 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
            <span className="font-medium">{error}</span>
          </div>
        )}

        {loading ? (
          <div className="flex flex-col items-center justify-center py-24 gap-6">
            <div className="relative w-16 h-16">
              <div className="absolute inset-0 rounded-full border-4 border-indigo-200"></div>
              <div className="absolute inset-0 rounded-full border-4 border-transparent border-t-indigo-500 animate-spin"></div>
              <div className="absolute inset-2 rounded-full border-4 border-transparent border-t-purple-400 animate-spin" style={{ animationDirection: "reverse", animationDuration: "1.5s" }}></div>
            </div>
            <p className="text-slate-500 font-medium">Loading workflows...</p>
          </div>
        ) : workflows.length === 0 ? (
          <div className="text-center py-20 animate-fade-in-up">
            <div className="relative mx-auto w-28 h-28 mb-8">
              <div className="absolute inset-0 rounded-3xl bg-gradient-to-br from-indigo-100 via-purple-100 to-pink-100 animate-blob"></div>
              <div className="absolute inset-0 flex items-center justify-center">
                <svg className="w-14 h-14 text-indigo-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z" />
                </svg>
              </div>
            </div>
            <h3 className="text-2xl font-extrabold text-slate-900 mb-3">No workflows yet</h3>
            <p className="text-slate-500 mb-8 text-lg">Create your first workflow to see the magic</p>
            <Link href="/workflows/new" className="btn-gradient inline-flex items-center gap-2 text-base">
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M12 4v16m8-8H4" />
              </svg>
              Create Workflow
            </Link>
          </div>
        ) : (
          <div className="space-y-4">
            <h2 className="text-lg font-bold text-slate-900 mb-5 flex items-center gap-2">
              <div className="w-1.5 h-6 rounded-full bg-gradient-to-b from-indigo-500 to-purple-500"></div>
              Recent Workflows
            </h2>
            {workflows.map((wf, i) => {
              const config = statusConfig[wf.status] || statusConfig.pending;
              return (
                <Link
                  key={wf.id}
                  href={`/workflows/${wf.id}`}
                  className="glass p-6 block animate-fade-in-up group relative overflow-hidden"
                  style={{ animationDelay: `${i * 80}ms` }}
                >
                  {/* Hover gradient bar */}
                  <div className="absolute left-0 top-0 bottom-0 w-1.5 bg-gradient-to-b from-indigo-500 via-purple-500 to-pink-500 opacity-0 group-hover:opacity-100 transition-opacity duration-300 rounded-r-full"></div>

                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-5">
                      <div className="w-14 h-14 rounded-2xl bg-gradient-to-br from-indigo-50 via-purple-50 to-pink-50 border border-indigo-100/60 flex items-center justify-center group-hover:from-indigo-100 group-hover:via-purple-100 group-hover:to-pink-100 transition-all duration-300 group-hover:scale-110">
                        <svg className="w-7 h-7 text-indigo-500 group-hover:text-indigo-600 transition-colors" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z" />
                        </svg>
                      </div>
                      <div>
                        <h3 className="text-lg font-bold text-slate-900 group-hover:text-indigo-600 transition-colors">{wf.name}</h3>
                        <p className="text-sm text-slate-400 font-mono">ID: {wf.id.slice(0, 8)}...</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-5">
                      <div className="text-right">
                        <div className="flex items-center gap-2 mb-1">
                          <span className="text-2xl font-extrabold text-slate-900">{wf.tasks?.length || 0}</span>
                          <span className="text-sm text-slate-500">tasks</span>
                        </div>
                        <p className="text-xs text-slate-400">
                          {wf.started_at ? new Date(wf.started_at).toLocaleTimeString() : "Not started"}
                        </p>
                      </div>
                      <span className={`badge-fancy ${config.badge}`}>
                        {wf.status === "running" && (
                          <span className="relative flex h-2 w-2">
                            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75"></span>
                            <span className="relative inline-flex rounded-full h-2 w-2 bg-blue-500"></span>
                          </span>
                        )}
                        {config.icon} {wf.status}
                      </span>
                    </div>
                  </div>

                  {/* Task chips */}
                  {wf.task_statuses && Object.keys(wf.task_statuses).length > 0 && (
                    <div className="mt-5 pt-4 border-t border-slate-100 flex flex-wrap gap-2">
                      {Object.values(wf.task_statuses).map((task, j) => {
                        const taskState = task.state;
                        return (
                          <span
                            key={task.id}
                            className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold animate-scale-in transition-all duration-200 ${
                              taskState === "completed"
                                ? "bg-emerald-50 text-emerald-700 border border-emerald-200/60 shadow-sm shadow-emerald-100"
                                : taskState === "running"
                                  ? "bg-blue-50 text-blue-700 border border-blue-200/60 shadow-sm shadow-blue-100"
                                  : taskState === "failed"
                                    ? "bg-red-50 text-red-700 border border-red-200/60 shadow-sm shadow-red-100"
                                    : "bg-slate-50 text-slate-600 border border-slate-200/60"
                            }`}
                            style={{ animationDelay: `${j * 40}ms` }}
                          >
                            <span className={`w-2 h-2 rounded-full ${
                              taskState === "completed" ? "bg-emerald-500" :
                              taskState === "running" ? "bg-blue-500 animate-pulse" :
                              taskState === "failed" ? "bg-red-500" :
                              "bg-slate-300"
                            }`}></span>
                            {task.id}
                          </span>
                        );
                      })}
                    </div>
                  )}
                </Link>
              );
            })}
          </div>
        )}
      </main>
    </div>
  );
}
