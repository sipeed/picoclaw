import { useEffect, useState } from 'preact/hooks';
import type { SSEHook } from '../../hooks/use-sse';
import { apiFetch, sendCommand } from '../../hooks/use-api';
import { isFresh, flashSent } from '../../utils';

interface SkillsSectionProps {
  active: boolean;
  sse: SSEHook;
}

export function SkillsSection({ active, sse }: SkillsSectionProps) {
  const [skills, setSkills] = useState<any[] | null>(null);
  const [loading, setLoading] = useState(true);
  const [selected, setSelected] = useState<string | null>(null);
  const [msg, setMsg] = useState('');

  const loadSkills = async () => {
    setLoading(true);
    try {
      const data = await apiFetch('/miniapp/api/skills');
      setSkills(data);
    } catch {}
    setLoading(false);
  };

  useEffect(() => {
    if (sse.skills) {
      setSkills(sse.skills);
      setLoading(false);
    }
  }, [sse.skills]);

  useEffect(() => {
    if (active && !isFresh(sse.lastUpdate, 'skills')) loadSkills();
  }, [active]);

  const handleSend = async () => {
    if (!selected) return;
    const m = msg.trim();
    const cmd = m ? '/skill ' + selected + ' ' + m : '/skill ' + selected;
    const ok = await sendCommand(cmd);
    if (ok) setMsg('');
  };

  return (
    <div class="card glass">
      <div class="card-title">Skills</div>
      {loading && !skills ? (
        <div class="loading" style={{ padding: '12px' }}>
          Loading skills...
        </div>
      ) : !skills || skills.length === 0 ? (
        <div
          class="empty-state"
          style={{ padding: '12px 0' }}
        >
          No skills installed.
        </div>
      ) : (
        <>
          {skills.map((s) => (
            <div
              key={s.name}
              class={`skill-item glass glass-interactive${selected === s.name ? ' selected' : ''}`}
              onClick={() => {
                setSelected(selected === s.name ? null : s.name);
              }}
            >
              <div class="skill-body">
                <div class="skill-name">{s.name}</div>
                <div class="skill-desc">
                  {s.description || 'No description'}
                </div>
                <span class="skill-source">{s.source}</span>
              </div>
              <span class="skill-arrow">{'\u203A'}</span>
            </div>
          ))}
          {selected && (
            <div
              style={{
                display: 'flex',
                gap: '8px',
                marginTop: '10px',
              }}
            >
              <input
                class="send-input glass glass-interactive"
                placeholder={`Message for /${selected}...`}
                value={msg}
                onInput={(e) =>
                  setMsg((e.target as HTMLInputElement).value)
                }
                onKeyDown={(e) => e.key === 'Enter' && handleSend()}
                style={{ flex: 1 }}
              />
              <button class="send-btn" onClick={handleSend}>
                Send
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
