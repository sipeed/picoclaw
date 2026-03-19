import { useEffect, useState, useCallback, useRef } from 'preact/hooks';
import { useSSE } from './hooks/use-sse';
import { PlanTab } from './components/plan/plan-tab';
import { WorkTab } from './components/work/work-tab';
import { ToolsTab } from './components/tools/tools-tab';
import { DevTab } from './components/dev/dev-tab';

const TABS = [
  { id: 'plan', label: 'Plan' },
  { id: 'work', label: 'Work' },
  { id: 'tools', label: 'Tools' },
  { id: 'dev', label: 'Dev' },
] as const;

type TabId = (typeof TABS)[number]['id'];

export function App() {
  const [activeTab, setActiveTab] = useState<TabId>('plan');
  const indicatorRef = useRef<HTMLDivElement>(null);
  const sse = useSSE();

  const switchTab = useCallback((id: TabId, index: number) => {
    setActiveTab(id);
    if (indicatorRef.current) {
      indicatorRef.current.style.transform = `translateX(${index * 100}%)`;
    }
  }, []);

  return (
    <>
      <div class="tabs">
        <div class="tabs-inner">
          <div class="tab-indicator" ref={indicatorRef} />
          {TABS.map((tab, i) => (
            <button
              key={tab.id}
              class={`tab${activeTab === tab.id ? ' active' : ''}`}
              onClick={() => switchTab(tab.id, i)}
            >
              {tab.label}
            </button>
          ))}
        </div>
      </div>

      <div class={`panel${activeTab === 'plan' ? ' active' : ''}`}>
        <PlanTab active={activeTab === 'plan'} sse={sse} />
      </div>
      <div class={`panel${activeTab === 'work' ? ' active' : ''}`}>
        <WorkTab active={activeTab === 'work'} sse={sse} />
      </div>
      <div class={`panel${activeTab === 'tools' ? ' active' : ''}`}>
        <ToolsTab active={activeTab === 'tools'} sse={sse} />
      </div>
      <div class={`panel${activeTab === 'dev' ? ' active' : ''}`}>
        <DevTab active={activeTab === 'dev'} sse={sse} />
      </div>
    </>
  );
}
