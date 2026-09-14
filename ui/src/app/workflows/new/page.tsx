"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { api, TaskDefinition } from "@/lib/api";

interface TaskInput {
  id: string;
  execute: string;
  dependencies: string;
}

const presetTemplates = [
  {
    name: "Data Pipeline",
    icon: "📊",
    gradient: "from-blue-500 to-cyan-400",
    description: "ETL with parallel transforms",
    tasks: [
      { id: "fetch", execute: "curl -s https://api.example.com/data -o data.json && echo Fetched $(wc -c < data.json) bytes", dependencies: "" },
      { id: "validate", execute: "cat data.json | python3 -m json.tool > /dev/null && echo JSON valid", dependencies: "fetch" },
      { id: "transform-a", execute: "cat data.json | head -5 && echo Transform A complete", dependencies: "validate" },
      { id: "transform-b", execute: "cat data.json | wc -l && echo Transform B complete", dependencies: "validate" },
      { id: "merge", execute: "echo Merging outputs... && echo Merge complete", dependencies: "transform-a, transform-b" },
      { id: "store", execute: "cp data.json /tmp/processed.json && echo Stored to /tmp/processed.json", dependencies: "merge" },
    ],
  },
  {
    name: "CI/CD Pipeline",
    icon: "🚀",
    gradient: "from-purple-500 to-pink-400",
    description: "Build, test, deploy",
    tasks: [
      { id: "checkout", execute: "git clone --depth 1 https://github.com/user/repo.git /tmp/repo 2>/dev/null || echo Repo already cloned", dependencies: "" },
      { id: "lint", execute: "echo Running linter... && echo Lint passed", dependencies: "checkout" },
      { id: "test", execute: "echo Running tests... && sleep 1 && echo All tests passed", dependencies: "checkout" },
      { id: "build", execute: "echo Building project... && echo Build successful", dependencies: "lint, test" },
      { id: "deploy-staging", execute: "echo Deploying to staging... && echo Staging ready", dependencies: "build" },
      { id: "deploy-prod", execute: "echo Deploying to production... && echo Deployed!", dependencies: "deploy-staging" },
    ],
  },
  {
    name: "ML Training",
    icon: "🧠",
    gradient: "from-amber-500 to-orange-400",
    description: "Data → Model → Evaluate",
    tasks: [
      { id: "download-data", execute: "echo Downloading dataset... && echo Downloaded 1000 samples", dependencies: "" },
      { id: "preprocess", execute: "echo Preprocessing data... && echo Preprocessed 1000 samples", dependencies: "download-data" },
      { id: "augment", execute: "echo Augmenting data... && echo Augmented to 5000 samples", dependencies: "preprocess" },
      { id: "train-model", execute: "echo Training model... && sleep 2 && echo Model trained, accuracy: 94.2%", dependencies: "augment" },
      { id: "evaluate", execute: "echo Evaluating model... && echo F1: 0.93, Precision: 0.95, Recall: 0.91", dependencies: "train-model" },
    ],
  },
];

export default function NewWorkflow() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [tasks, setTasks] = useState<TaskInput[]>([{ id: "task1", execute: "echo Hello", dependencies: "" }]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedTemplate, setSelectedTemplate] = useState<string | null>(null);

  const applyTemplate = (template: typeof presetTemplates[0]) => {
    setName(template.name);
    setTasks(template.tasks);
    setSelectedTemplate(template.name);
  };

  const addTask = () => {
    setTasks([...tasks, { id: `task${tasks.length + 1}`, execute: "echo Done", dependencies: "" }]);
  };

  const removeTask = (index: number) => {
    if (tasks.length > 1) {
      setTasks(tasks.filter((_, i) => i !== index));
    }
  };

  const updateTask = (index: number, field: keyof TaskInput, value: string) => {
    const updated = [...tasks];
    updated[index] = { ...updated[index], [field]: value };
    setTasks(updated);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const taskDefs: TaskDefinition[] = tasks.map((t) => ({
        id: t.id,
        execute: t.execute || "echo No command",
        dependencies: t.dependencies
          ? t.dependencies.split(",").map((d) => d.trim()).filter(Boolean)
          : [],
      }));

      const workflow = await api.submitWorkflow({ name, tasks: taskDefs });
      router.push(`/workflows/${workflow.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create workflow");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen relative">
      {/* Animated Background */}
      <div className="bg-blob bg-blob-1"></div>
      <div className="bg-blob bg-blob-2"></div>
      <div className="bg-blob bg-blob-3"></div>

      {/* Header */}
      <header className="sticky top-0 z-50 glass-static" style={{ borderRadius: 0, borderTop: "none", borderLeft: "none", borderRight: "none" }}>
        <div className="max-w-3xl mx-auto px-4 sm:px-6 py-5">
          <Link href="/" className="inline-flex items-center gap-2 text-slate-500 hover:text-indigo-600 transition-colors text-sm font-semibold group">
            <svg className="w-4 h-4 group-hover:-translate-x-1 transition-transform" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M15 19l-7-7 7-7" />
            </svg>
            Back to Dashboard
          </Link>
        </div>
      </header>

      <main className="max-w-3xl mx-auto px-4 sm:px-6 py-10 relative z-10">
        <div className="animate-fade-in-up mb-10">
          <div className="flex items-center gap-3 mb-2">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-indigo-500 to-purple-500 flex items-center justify-center shadow-lg shadow-indigo-500/25">
              <svg className="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M12 4v16m8-8H4" />
              </svg>
            </div>
            <h1 className="text-3xl font-extrabold text-slate-900">Create Workflow</h1>
          </div>
          <p className="text-slate-500 text-lg ml-[52px]">Define tasks and their execution order</p>
        </div>

        {/* Templates */}
        <div className="mb-10 animate-fade-in-up" style={{ animationDelay: "100ms" }}>
          <label className="block text-sm font-bold text-slate-700 mb-4 uppercase tracking-wider">Quick Start Templates</label>
          <div className="grid grid-cols-3 gap-4">
            {presetTemplates.map((template) => (
              <button
                key={template.name}
                type="button"
                onClick={() => applyTemplate(template)}
                className={`glass p-5 text-left group transition-all duration-300 ${
                  selectedTemplate === template.name
                    ? "ring-2 ring-indigo-400 border-indigo-300"
                    : "hover:border-indigo-200"
                }`}
              >
                <div className={`w-12 h-12 rounded-xl bg-gradient-to-br ${template.gradient} flex items-center justify-center text-2xl mb-3 shadow-lg group-hover:scale-110 transition-transform duration-300`}>
                  {template.icon}
                </div>
                <p className="font-bold text-slate-900 group-hover:text-indigo-600 transition-colors">{template.name}</p>
                <p className="text-xs text-slate-500 mt-1">{template.description}</p>
                <p className="text-xs text-slate-400 mt-2 font-medium">{template.tasks.length} tasks</p>
              </button>
            ))}
          </div>
        </div>

        <form onSubmit={handleSubmit} className="space-y-8">
          {/* Name */}
          <div className="animate-fade-in-up" style={{ animationDelay: "200ms" }}>
            <label className="block text-sm font-bold text-slate-700 mb-3 uppercase tracking-wider">Workflow Name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => { setName(e.target.value); setSelectedTemplate(null); }}
              required
              className="input-fancy text-lg font-medium"
              placeholder="e.g., Data Pipeline"
            />
          </div>

          {/* Tasks */}
          <div className="animate-fade-in-up" style={{ animationDelay: "300ms" }}>
            <div className="flex items-center justify-between mb-4">
              <label className="text-sm font-bold text-slate-700 uppercase tracking-wider">Tasks</label>
              <button
                type="button"
                onClick={addTask}
                className="inline-flex items-center gap-1.5 px-4 py-2 text-sm font-bold text-indigo-600 bg-indigo-50 rounded-xl hover:bg-indigo-100 transition-all duration-200 hover:scale-105 border border-indigo-100"
              >
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M12 4v16m8-8H4" />
                </svg>
                Add Task
              </button>
            </div>

            <div className="space-y-3">
              {tasks.map((task, index) => (
                <div
                  key={index}
                  className="glass p-4 flex items-center gap-3 animate-fade-in-left group/task"
                  style={{ animationDelay: `${index * 50}ms` }}
                >
                  <div className="w-9 h-9 rounded-xl bg-gradient-to-br from-indigo-100 to-purple-100 flex items-center justify-center text-sm font-extrabold text-indigo-600 flex-shrink-0 group-hover/task:from-indigo-200 group-hover/task:to-purple-200 transition-colors">
                    {index + 1}
                  </div>
                  <input
                    type="text"
                    value={task.id}
                    onChange={(e) => updateTask(index, "id", e.target.value)}
                    required
                    className="input-fancy flex-1 text-sm font-medium"
                    placeholder="Task ID (e.g., fetch-data)"
                  />
                  <input
                    type="text"
                    value={task.execute}
                    onChange={(e) => updateTask(index, "execute", e.target.value)}
                    required
                    className="input-fancy flex-1 text-sm font-mono"
                    placeholder="Command (e.g., echo Hello)"
                  />
                  <div className="flex items-center gap-1.5 flex-1">
                    <svg className="w-4 h-4 text-slate-400 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7l5 5m0 0l-5 5m5-5H6" />
                    </svg>
                    <input
                      type="text"
                      value={task.dependencies}
                      onChange={(e) => updateTask(index, "dependencies", e.target.value)}
                      className="input-fancy flex-1 text-sm"
                      placeholder="Dependencies (comma-separated)"
                    />
                  </div>
                  <button
                    type="button"
                    onClick={() => removeTask(index)}
                    className="w-9 h-9 rounded-xl text-slate-300 hover:text-red-500 hover:bg-red-50 transition-all duration-200 flex items-center justify-center flex-shrink-0 hover:scale-110"
                  >
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>
              ))}
            </div>

            {/* Visual task count */}
            <div className="mt-4 flex items-center gap-2 text-sm text-slate-500">
              <div className="flex -space-x-2">
                {tasks.slice(0, 5).map((_, i) => (
                  <div key={i} className="w-6 h-6 rounded-full bg-gradient-to-br from-indigo-400 to-purple-400 border-2 border-white flex items-center justify-center text-[9px] text-white font-bold">
                    {i + 1}
                  </div>
                ))}
                {tasks.length > 5 && (
                  <div className="w-6 h-6 rounded-full bg-slate-200 border-2 border-white flex items-center justify-center text-[9px] text-slate-600 font-bold">
                    +{tasks.length - 5}
                  </div>
                )}
              </div>
              <span>{tasks.length} task{tasks.length !== 1 ? "s" : ""} configured</span>
            </div>
          </div>

          {/* Error */}
          {error && (
            <div className="p-5 glass border-l-4 border-red-400 bg-red-50/50 text-red-700 animate-fade-in-up flex items-center gap-4 rounded-2xl">
              <div className="w-10 h-10 rounded-xl bg-red-100 flex items-center justify-center flex-shrink-0">
                <svg className="w-5 h-5 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <span className="font-medium">{error}</span>
            </div>
          )}

          {/* Actions */}
          <div className="flex gap-4 animate-fade-in-up" style={{ animationDelay: "400ms" }}>
            <button
              type="submit"
              disabled={loading}
              className="btn-gradient flex items-center gap-2 disabled:opacity-50 flex-1 justify-center"
            >
              {loading ? (
                <>
                  <div className="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                  Creating...
                </>
              ) : (
                <>
                  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M5 13l4 4L19 7" />
                  </svg>
                  Create & Run Workflow
                </>
              )}
            </button>
            <Link href="/" className="btn-ghost px-8">
              Cancel
            </Link>
          </div>
        </form>
      </main>
    </div>
  );
}
