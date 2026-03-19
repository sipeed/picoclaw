import type { SSEHook } from '../../hooks/use-sse';
import { SkillsSection } from './skills-section';
import { CommandsSection } from './commands-section';
import { LogsSection } from './logs-section';
import { ResearchSection } from './research-section';
import { CacheSection } from './cache-section';

interface ToolsTabProps {
  active: boolean;
  sse: SSEHook;
}

export function ToolsTab({ active, sse }: ToolsTabProps) {
  return (
    <>
      <SkillsSection active={active} sse={sse} />
      <CommandsSection />
      <LogsSection active={active} />
      <ResearchSection active={active} />
      <CacheSection active={active} />
    </>
  );
}
