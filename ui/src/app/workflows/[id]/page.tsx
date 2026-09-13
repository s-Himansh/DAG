"use client";

import { useEffect, useState, use } from "react";
import Link from "next/link";
import { api, Workflow } from "@/lib/api";

const nodeStyles: Record<string, { fill: string; stroke: string; text: string; gradient: string; glow: string }> = {
  pending:   { fill: "#f8fafc", stroke: "#cbd5e1", text: "#64748b", gradient: "from-slate-100 to-slate-50",   glow: "none" },
  running:   { fill: "#eff6ff", stroke: "#3b82f6", text: "#1e40af", gradient: "from-blue-100 to-cyan-50",     glow: "0 0 20px rgba(59,130,246,0.4)" },
  completed: { fill: "#ecfdf5", stroke: "#10b981", text: "#065f46", gradient: "from-emerald-100 to-green-50",  glow: "0 0 16px rgba(16,185,129,0.3)" },
  failed:    { fill: "#fef2f2", stroke: "#ef4444", text: "#991b1b", gradient: "from-red-100 to-rose-50",       glow: "0 0 16px rgba(239,68,68,0.3)" },
  skipped:   { fill: "#fffbeb", stroke: "#f59e0b", text: "#92400e", gradient: "from-amber-100 to-yellow-50",   glow: "0 0 16px rgba(245,158,11,0.3)" },
};

const statusConfig: Record<string, { badge: string; icon: string }> = {
  pending:   { badge: "badge-pending",   icon: "⏳" },
  running:   { badge: "badge-running",   icon: "⚡" },
  completed: { badge: "badge-completed", icon: "✓" },
  failed:    { badge: "badge-failed",    icon: "✗" },
  cancelled: { badge: "badge-cancelled", icon: "○" },
};

interface NodePosition {
  id: string;
  x: number;
  y: number;
  state: string;
  dependencies: string[];
}

function computeLayout(tasks: { id: string; dependencies?: string[] }[]): NodePosition[] {
  const levels: string[][] = [];
  const taskMap = new Map(tasks.map((t) => [t.id, t]));
  const assigned = new Set<string>();

  let currentLevel = tasks.filter((t) => !t.dependencies || t.dependencies.length === 0).map((t) => t.id);
  while (currentLevel.length > 0) {
    levels.push(currentLevel);
    currentLevel.forEach((id) => assigned.add(id));

    const nextLevel: string[] = [];
    for (const task of tasks) {
      if (assigned.has(task.id)) continue;
      const deps = task.dependencies || [];
      if (deps.every((d) => assigned.has(d))) {
        nextLevel.push(task.id);
      }
    }
    currentLevel = nextLevel;
  }

  const nodeWidth = 160;
  const nodeHeight = 80;
  const levelGap = 140;
  const nodeGap = 60;

  const positions: NodePosition[] = [];

  for (let levelIdx = 0; levelIdx < levels.length; levelIdx++) {
    const level = levels[levelIdx];
    const totalWidth = level.length * nodeWidth + (level.length - 1) * nodeGap;
    const startX = -totalWidth / 2;

    for (let i = 0; i < level.length; i++) {
      const id = level[i];
      const task = taskMap.get(id);
      positions.push({
        id,
        x: startX + i * (nodeWidth + nodeGap) + nodeWidth / 2,
        y: levelIdx * (nodeHeight + levelGap) + nodeHeight / 2,
        state: "pending",
        dependencies: task?.dependencies || [],
      });
    }
  }

  return positions;
}

export default function WorkflowDetail({ params }: { params: Promise<{ id: string }> }) {
  const resolvedParams = use(params);
  const [workflow, setWorkflow] = useState<Workflow | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchWorkflow = async () => {
    try {
      const data = await api.getWorkflow(resolvedParams.id);
      setWorkflow(data);
      setError(null);
    } catch {
      setError("Failed to fetch workflow");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchWorkflow();
    const interval = setInterval(fetchWorkflow, 1000);
    return () => clearInterval(interval);
  }, [resolvedParams.id]);

  const handleCancel = async () => {
    try {
      await api.cancelWorkflow(resolvedParams.id);
      fetchWorkflow();
    } catch {
      setError("Failed to cancel workflow");
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="bg-blob bg-blob-1"></div>
        <div className="bg-blob bg-blob-2"></div>
        <div className="flex flex-col items-center gap-5 relative z-10">
          <div className="relative w-16 h-16">
            <div className="absolute inset-0 rounded-full border-4 border-indigo-200"></div>
            <div className="absolute inset-0 rounded-full border-4 border-transparent border-t-indigo-500 animate-spin"></div>
            <div className="absolute inset-2 rounded-full border-4 border-transparent border-t-purple-400 animate-spin" style={{ animationDirection: "reverse", animationDuration: "1.5s" }}></div>
          </div>
          <p className="text-slate-500 font-medium">Loading workflow...</p>
        </div>
      </div>
    );
  }

  if (!workflow) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="bg-blob bg-blob-1"></div>
        <div className="bg-blob bg-blob-2"></div>
        <div className="text-center relative z-10">
          <p className="text-2xl font-bold text-slate-900 mb-2">Workflow not found</p>
          <Link href="/" className="text-indigo-600 hover:underline font-medium">← Back to Dashboard</Link>
        </div>
      </div>
    );
  }

  const taskPositions = computeLayout(
    workflow.tasks.map((t) => ({ id: t.id, dependencies: t.dependencies }))
  );

  if (workflow.task_statuses) {
    for (const pos of taskPositions) {
      const status = workflow.task_statuses[pos.id];
      if (status) pos.state = status.state;
    }
  }

  const padding = 100;
  const nodeWidth = 160;
  const nodeHeight = 80;

  let minX = Infinity, maxX = -Infinity, minY = Infinity, maxY = -Infinity;
  for (const pos of taskPositions) {
    minX = Math.min(minX, pos.x - nodeWidth / 2);
    maxX = Math.max(maxX, pos.x + nodeWidth / 2);
    minY = Math.min(minY, pos.y - nodeHeight / 2);
    maxY = Math.max(maxY, pos.y + nodeHeight / 2);
  }

  const svgWidth = maxX - minX + padding * 2;
  const svgHeight = maxY - minY + padding * 2;
  const offsetX = -minX + padding;
  const offsetY = -minY + padding;

  const wfConfig = statusConfig[workflow.status] || statusConfig.pending;

  return (
    <div className="min-h-screen relative">
      <div className="bg-blob bg-blob-1"></div>
      <div className="bg-blob bg-blob-2"></div>
      <div className="bg-blob bg-blob-3"></div>

      {/* Header */}
      <header className="sticky top-0 z-50 glass-static" style={{ borderRadius: 0, borderTop: "none", borderLeft: "none", borderRight: "none" }}>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-5">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <Link href="/" className="w-10 h-10 rounded-xl bg-slate-100 hover:bg-slate-200 flex items-center justify-center transition-all hover:scale-110">
                <svg className="w-5 h-5 text-slate-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M15 19l-7-7 7-7" />
                </svg>
              </Link>
              <div>
                <h1 className="text-xl font-extrabold text-slate-900">{workflow.name}</h1>
                <p className="text-xs text-slate-400 font-mono">ID: {workflow.id.slice(0, 12)}...</p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <span className={`badge-fancy ${wfConfig.badge}`}>
                {workflow.status === "running" && (
                  <span className="relative flex h-2 w-2">
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75"></span>
                    <span className="relative inline-flex rounded-full h-2 w-2 bg-blue-500"></span>
                  </span>
                )}
                {wfConfig.icon} {workflow.status}
              </span>
              {(workflow.status === "running" || workflow.status === "pending") && (
                <button
                  onClick={handleCancel}
                  className="px-5 py-2.5 bg-red-50 text-red-600 rounded-xl hover:bg-red-100 transition-all text-sm font-bold border border-red-200 hover:scale-105"
                >
                  Cancel
                </button>
              )}
            </div>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-10 relative z-10">
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

        {/* DAG Visualization */}
        <div className="glass-static p-8 mb-10 animate-fade-in-up" style={{ animationDelay: "100ms" }}>
          <div className="flex items-center gap-3 mb-6">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-indigo-500 to-purple-500 flex items-center justify-center shadow-lg shadow-indigo-500/25">
              <svg className="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z" />
              </svg>
            </div>
            <div>
              <h2 className="text-lg font-bold text-slate-900">DAG Visualization</h2>
              <p className="text-xs text-slate-500">{taskPositions.length} nodes • {taskPositions.reduce((acc, p) => acc + p.dependencies.length, 0)} edges</p>
            </div>
          </div>

          <div className="overflow-x-auto rounded-2xl p-6" style={{ background: "linear-gradient(135deg, #f8fafc 0%, #f1f5f9 50%, #eef2ff 100%)" }}>
            <svg
              width={svgWidth}
              height={svgHeight}
              viewBox={`0 0 ${svgWidth} ${svgHeight}`}
              className="mx-auto"
            >
              <defs>
                <marker id="arrow" markerWidth="14" markerHeight="10" refX="13" refY="5" orient="auto">
                  <polygon points="0 0, 14 5, 0 10" fill="#94a3b8" />
                </marker>
                <filter id="node-shadow" x="-30%" y="-30%" width="160%" height="160%">
                  <feDropShadow dx="0" dy="4" stdDeviation="6" floodOpacity="0.08" />
                </filter>
                <filter id="glow-blue" x="-50%" y="-50%" width="200%" height="200%">
                  <feGaussianBlur stdDeviation="6" result="blur" />
                  <feFlood floodColor="#3b82f6" floodOpacity="0.3" result="color" />
                  <feComposite in="color" in2="blur" operator="in" result="glow" />
                  <feMerge><feMergeNode in="glow" /><feMergeNode in="SourceGraphic" /></feMerge>
                </filter>
                <filter id="glow-green" x="-50%" y="-50%" width="200%" height="200%">
                  <feGaussianBlur stdDeviation="5" result="blur" />
                  <feFlood floodColor="#10b981" floodOpacity="0.3" result="color" />
                  <feComposite in="color" in2="blur" operator="in" result="glow" />
                  <feMerge><feMergeNode in="glow" /><feMergeNode in="SourceGraphic" /></feMerge>
                </filter>
                <filter id="glow-red" x="-50%" y="-50%" width="200%" height="200%">
                  <feGaussianBlur stdDeviation="5" result="blur" />
                  <feFlood floodColor="#ef4444" floodOpacity="0.3" result="color" />
                  <feComposite in="color" in2="blur" operator="in" result="glow" />
                  <feMerge><feMergeNode in="glow" /><feMergeNode in="SourceGraphic" /></feMerge>
                </filter>
                <linearGradient id="edge-gradient" x1="0%" y1="0%" x2="0%" y2="100%">
                  <stop offset="0%" stopColor="#a5b4fc" />
                  <stop offset="100%" stopColor="#c4b5fd" />
                </linearGradient>
              </defs>

              {/* Edges */}
              {taskPositions.map((pos) =>
                pos.dependencies.map((depId) => {
                  const depPos = taskPositions.find((p) => p.id === depId);
                  if (!depPos) return null;

                  const x1 = depPos.x + offsetX;
                  const y1 = depPos.y + offsetY + nodeHeight / 2;
                  const x2 = pos.x + offsetX;
                  const y2 = pos.y + offsetY - nodeHeight / 2;
                  const midY = (y1 + y2) / 2;

                  const isActive = depPos.state === "completed" && (pos.state === "running" || pos.state === "completed");

                  return (
                    <g key={`${depId}-${pos.id}`}>
                      <path
                        d={`M ${x1} ${y1} C ${x1} ${midY}, ${x2} ${midY}, ${x2} ${y2}`}
                        fill="none"
                        stroke={isActive ? "url(#edge-gradient)" : "#e2e8f0"}
                        strokeWidth={isActive ? 3 : 2}
                        strokeDasharray={isActive ? "8 4" : "none"}
                        className={isActive ? "animate-edge-flow" : ""}
                        markerEnd="url(#arrow)"
                        style={{ animationDelay: "0.3s" }}
                      />
                    </g>
                  );
                })
              )}

              {/* Nodes */}
              {taskPositions.map((pos, i) => {
                const style = nodeStyles[pos.state] || nodeStyles.pending;
                const filterAttr =
                  pos.state === "running" ? "url(#glow-blue)" :
                  pos.state === "completed" ? "url(#glow-green)" :
                  pos.state === "failed" ? "url(#glow-red)" :
                  "url(#node-shadow)";

                return (
                  <g
                    key={pos.id}
                    className="animate-scale-in"
                    style={{ animationDelay: `${0.15 + i * 0.06}s` }}
                  >
                    {/* Node body */}
                    <rect
                      x={pos.x + offsetX - nodeWidth / 2}
                      y={pos.y + offsetY - nodeHeight / 2}
                      width={nodeWidth}
                      height={nodeHeight}
                      rx="16"
                      fill="white"
                      stroke={style.stroke}
                      strokeWidth="2.5"
                      filter={filterAttr}
                    />

                    {/* Top accent bar */}
                    <rect
                      x={pos.x + offsetX - nodeWidth / 2}
                      y={pos.y + offsetY - nodeHeight / 2}
                      width={nodeWidth}
                      height="6"
                      rx="16"
                      fill={style.stroke}
                    />
                    <rect
                      x={pos.x + offsetX - nodeWidth / 2}
                      y={pos.y + offsetY - nodeHeight / 2 + 10}
                      width={nodeWidth}
                      height="4"
                      fill="white"
                    />

                    {/* Task ID */}
                    <text
                      x={pos.x + offsetX}
                      y={pos.y + offsetY - 6}
                      textAnchor="middle"
                      dominantBaseline="middle"
                      className="text-sm font-bold"
                      fill={style.text}
                    >
                      {pos.id}
                    </text>

                    {/* State label */}
                    <rect
                      x={pos.x + offsetX - 32}
                      y={pos.y + offsetY + 10}
                      width="64"
                      height="22"
                      rx="11"
                      fill={style.stroke}
                      opacity="0.12"
                    />
                    <text
                      x={pos.x + offsetX}
                      y={pos.y + offsetY + 22}
                      textAnchor="middle"
                      dominantBaseline="middle"
                      className="text-[10px] font-bold uppercase tracking-wider"
                      fill={style.text}
                    >
                      {pos.state}
                    </text>

                    {/* Running spinner */}
                    {pos.state === "running" && (
                      <g>
                        <circle
                          cx={pos.x + offsetX + nodeWidth / 2 - 12}
                          cy={pos.y + offsetY - nodeHeight / 2 + 14}
                          r="8"
                          fill="white"
                          stroke={style.stroke}
                          strokeWidth="2"
                        />
                        <circle
                          cx={pos.x + offsetX + nodeWidth / 2 - 12}
                          cy={pos.y + offsetY - nodeHeight / 2 + 14}
                          r="6"
                          fill="none"
                          stroke={style.stroke}
                          strokeWidth="2.5"
                          strokeDasharray="20 12"
                          className="animate-spin-slow"
                          style={{ transformOrigin: `${pos.x + offsetX + nodeWidth / 2 - 12}px ${pos.y + offsetY - nodeHeight / 2 + 14}px` }}
                        />
                      </g>
                    )}

                    {/* Completed check */}
                    {pos.state === "completed" && (
                      <circle
                        cx={pos.x + offsetX + nodeWidth / 2 - 12}
                        cy={pos.y + offsetY - nodeHeight / 2 + 14}
                        r="8"
                        fill="#10b981"
                      />
                    )}
                    {pos.state === "completed" && (
                      <text
                        x={pos.x + offsetX + nodeWidth / 2 - 12}
                        y={pos.y + offsetY - nodeHeight / 2 + 15}
                        textAnchor="middle"
                        dominantBaseline="middle"
                        fill="white"
                        className="text-[9px] font-bold"
                      >
                        ✓
                      </text>
                    )}

                    {/* Failed X */}
                    {pos.state === "failed" && (
                      <circle
                        cx={pos.x + offsetX + nodeWidth / 2 - 12}
                        cy={pos.y + offsetY - nodeHeight / 2 + 14}
                        r="8"
                        fill="#ef4444"
                      />
                    )}
                    {pos.state === "failed" && (
                      <text
                        x={pos.x + offsetX + nodeWidth / 2 - 12}
                        y={pos.y + offsetY - nodeHeight / 2 + 15}
                        textAnchor="middle"
                        dominantBaseline="middle"
                        fill="white"
                        className="text-[9px] font-bold"
                      >
                        ✗
                      </text>
                    )}
                  </g>
                );
              })}
            </svg>
          </div>
        </div>

        {/* Task Details */}
        <div className="glass-static p-8 animate-fade-in-up" style={{ animationDelay: "250ms" }}>
          <div className="flex items-center gap-3 mb-6">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-purple-500 to-pink-500 flex items-center justify-center shadow-lg shadow-purple-500/25">
              <svg className="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
              </svg>
            </div>
            <div>
              <h2 className="text-lg font-bold text-slate-900">Task Details</h2>
              <p className="text-xs text-slate-500">{workflow.tasks.length} tasks defined</p>
            </div>
          </div>

          <div className="grid gap-4">
            {workflow.tasks.map((task, i) => {
              const status = workflow.task_statuses?.[task.id];
              const state = status?.state || "pending";
              const style = nodeStyles[state] || nodeStyles.pending;
              const cfg = statusConfig[state] || statusConfig.pending;

              return (
                <div
                  key={task.id}
                  className={`glass p-5 animate-fade-in-left group hover:translate-x-1 transition-transform duration-200`}
                  style={{ animationDelay: `${i * 60}ms`, borderLeftWidth: "4px", borderLeftColor: style.stroke }}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-4">
                      <div className={`w-11 h-11 rounded-xl bg-gradient-to-br ${style.gradient} flex items-center justify-center border-2`} style={{ borderColor: style.stroke + "40" }}>
                        <span className="text-sm font-extrabold" style={{ color: style.text }}>
                          {i + 1}
                        </span>
                      </div>
                      <div>
                        <span className="font-bold text-slate-900 text-base">{task.id}</span>
                        {task.dependencies && task.dependencies.length > 0 && (
                          <div className="flex items-center gap-1.5 mt-1">
                            <svg className="w-3 h-3 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M13 7l5 5m0 0l-5 5m5-5H6" />
                            </svg>
                            <span className="text-xs text-slate-500">
                              from: {task.dependencies.join(", ")}
                            </span>
                          </div>
                        )}
                      </div>
                    </div>
                    <span className={`badge-fancy ${cfg.badge}`}>
                      {cfg.icon} {state}
                    </span>
                  </div>

                  {status?.error && (
                    <div className="mt-3 p-3 bg-red-50 border border-red-200/60 rounded-xl text-sm text-red-700 font-medium">
                      {status.error}
                    </div>
                  )}

                  {status?.value !== undefined && status?.value !== null && (
                    <div className="mt-3 p-3 bg-slate-50 border border-slate-200/60 rounded-xl text-sm text-slate-700 font-mono">
                      Output: {JSON.stringify(status.value)}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      </main>
    </div>
  );
}
