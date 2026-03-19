import { useState, useRef } from 'preact/hooks';
import { sendCommand } from '../../hooks/use-api';
import { flashSent } from '../../utils';

const QUICK_CMDS = ['/session', '/skills', '/plan clear'];

export function CommandsSection() {
  const [customCmd, setCustomCmd] = useState('');
  const btnRef = useRef<HTMLButtonElement>(null);

  const handleQuick = async (cmd: string, e: MouseEvent) => {
    const ok = await sendCommand(cmd);
    if (ok) flashSent(e.currentTarget as HTMLElement);
  };

  const handleCustom = async () => {
    const cmd = customCmd.trim();
    if (!cmd || !cmd.startsWith('/')) return;
    const ok = await sendCommand(cmd);
    if (ok) {
      setCustomCmd('');
      if (btnRef.current) flashSent(btnRef.current);
    }
  };

  return (
    <>
      <div class="card glass">
        <div class="card-title">Quick Commands</div>
        <div class="cmd-tiles">
          {QUICK_CMDS.map((cmd) => (
            <button
              key={cmd}
              class="cmd-tile glass glass-interactive"
              onClick={(e) => handleQuick(cmd, e)}
            >
              {cmd}
            </button>
          ))}
        </div>
      </div>
      <div class="card glass">
        <div class="card-title">Custom Command</div>
        <div style={{ display: 'flex', gap: '8px', marginTop: '8px' }}>
          <input
            class="send-input glass glass-interactive"
            placeholder="/command args..."
            value={customCmd}
            onInput={(e) =>
              setCustomCmd((e.target as HTMLInputElement).value)
            }
            onKeyDown={(e) => e.key === 'Enter' && handleCustom()}
            style={{ flex: 1 }}
          />
          <button class="send-btn" ref={btnRef} onClick={handleCustom}>
            Send
          </button>
        </div>
      </div>
    </>
  );
}
