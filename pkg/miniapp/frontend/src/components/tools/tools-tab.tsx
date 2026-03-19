import { useEffect, useState } from 'preact/hooks';
import type { SSEHook } from '../../hooks/use-sse';
import { isFresh } from '../../utils';
import { SkillsSection } from './skills-section';
import { CommandsSection } from './commands-section';
import { LogsSection } from './logs-section';
import { ResearchSection } from './research-section';

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
    </>
  );
}
