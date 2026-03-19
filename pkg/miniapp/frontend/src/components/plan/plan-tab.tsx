import { useEffect, useState, useCallback } from 'preact/hooks';
import type { SSEHook } from '../../hooks/use-sse';
import { apiFetch, sendCommand } from '../../hooks/use-api';
import { escapeHtml, isFresh } from '../../utils';
import { renderMarkdown } from '../../markdown';
import { SlideApprove } from './slide-approve';
import { OrchCanvas } from './orch-canvas';

interface PlanTabProps {
  active: boolean;
  sse: SSEHook;
}

export function PlanTab({ active, sse }: PlanTabProps) {
  const [plan, setPlan] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  const loadPlan = useCallback(async () => {
    setLoading(true);
    setError(false);
    try {
      const data = await apiFetch('/miniapp/api/plan');
      setPlan(data);
    } catch {
      setError(true);
    } finally {
      setLoading(false);
    }
  }, []);

  // SSE updates
  useEffect(() => {
    if (sse.plan) {
      setPlan(sse.plan);
      setLoading(false);
    }
  }, [sse.plan]);

  // Load on tab switch if not fresh
  useEffect(() => {
    if (active && !isFresh(sse.lastUpdate, 'plan')) {
      loadPlan();
    }
  }, [active]);

  // Initial load
  useEffect(() => {
    loadPlan();
  }, []);

  if (loading && !plan) return <div class="loading">Loading plan...</div>;
  if (error && !plan) return <div class="loading">Failed to load plan.</div>;
  if (!plan) return null;

  return (
    <>
      <PlanContent plan={plan} />
      {window.ORCH_ENABLED && <OrchCanvas active={active} />}
    </>
  );
}

function PlanContent({ plan }: { plan: any }) {
  if (!plan.has_plan) {
    return <NoPlan />;
  }

  const isInterviewOrReview =
    plan.status === 'interviewing' || plan.status === 'review';

  return (
    <>
      <div class="card glass">
        <div class="card-title">Status</div>
        <div class="card-value">{plan.status}</div>
        <div style={{ color: 'var(--hint)', marginTop: '4px' }}>
          Phase {plan.current_phase} / {plan.total_phases}
        </div>
      </div>

      {isInterviewOrReview && plan.memory && (
        <div
          class="memory-view glass"
          dangerouslySetInnerHTML={{ __html: renderMarkdown(plan.memory) }}
        />
      )}

      {plan.status === 'review' && (
        <>
          <SlideApprove label="Slide to Approve" cmd="/plan start" />
          <SlideApprove
            label="Approve & Clear History"
            cmd="/plan start clear"
            warn
          />
        </>
      )}

      {!isInterviewOrReview && plan.phases && plan.phases.length > 0 && (
        <Phases phases={plan.phases} currentPhase={plan.current_phase} />
      )}
    </>
  );
}

function NoPlan() {
  const [task, setTask] = useState('');

  const handleStart = async () => {
    const t = task.trim();
    if (!t) return;
    const ok = await sendCommand('/plan ' + t);
    if (ok) setTask('');
  };

  return (
    <>
      <div class="empty-state">No active plan.</div>
      <div class="card glass" style={{ marginTop: '16px' }}>
        <div class="card-title">Start a Plan</div>
        <div style={{ display: 'flex', gap: '8px', marginTop: '8px' }}>
          <input
            class="send-input glass glass-interactive"
            placeholder="Describe your task..."
            value={task}
            onInput={(e) => setTask((e.target as HTMLInputElement).value)}
            onKeyDown={(e) => e.key === 'Enter' && handleStart()}
          />
          <button class="send-btn" onClick={handleStart}>
            Start
          </button>
        </div>
      </div>
    </>
  );
}

function Phases({
  phases,
  currentPhase,
}: {
  phases: any[];
  currentPhase: number;
}) {
  const onStepClick = (phaseNum: number, stepIdx: number, done: boolean) => {
    if (done) return;
    sendCommand('/plan done ' + stepIdx);
  };

  return (
    <>
      {phases.map((phase) => {
        const doneCount = phase.steps.filter((s: any) => s.done).length;
        const total = phase.steps.length;

        let indicatorClass: string;
        let indicator: string;
        if (
          phase.number < currentPhase ||
          (total > 0 && doneCount === total)
        ) {
          indicatorClass = 'done';
          indicator = '\u2713';
        } else if (phase.number === currentPhase) {
          indicatorClass = 'current';
          indicator = String(phase.number);
        } else {
          indicatorClass = 'pending';
          indicator = String(phase.number);
        }

        return (
          <div class="phase" key={phase.number}>
            <div class="phase-header">
              <div class={`phase-indicator ${indicatorClass}`}>{indicator}</div>
              <span class="phase-title">
                {phase.title || 'Phase ' + phase.number}
              </span>
              {total > 0 && (
                <span class="phase-progress">
                  {doneCount}/{total}
                </span>
              )}
            </div>
            {phase.steps.map((step: any) => (
              <div
                key={step.index}
                class={`step${step.done ? ' step-done' : ''}`}
                onClick={() =>
                  onStepClick(phase.number, step.index, step.done)
                }
              >
                <div class={`step-check${step.done ? ' done' : ''}`} />
                <div class={`step-text${step.done ? ' done' : ''}`}>
                  {step.description}
                </div>
              </div>
            ))}
          </div>
        );
      })}
    </>
  );
}
