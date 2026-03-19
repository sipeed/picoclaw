import { useRef, useEffect } from 'preact/hooks';
import { sendCommand } from '../../hooks/use-api';

interface SlideApproveProps {
  label: string;
  cmd: string;
  warn?: boolean;
}

export function SlideApprove({ label, cmd, warn }: SlideApproveProps) {
  const trackRef = useRef<HTMLDivElement>(null);
  const thumbRef = useRef<HTMLDivElement>(null);
  const labelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const track = trackRef.current;
    const thumb = thumbRef.current;
    if (!track || !thumb) return;

    const ac = new AbortController();
    const signal = ac.signal;

    let dragging = false;
    let startX = 0;
    let thumbStartLeft = 0;

    function getMaxLeft() {
      return track!.offsetWidth - thumb!.offsetWidth - 6;
    }

    function markApproved() {
      track!.classList.add('approved');
      if (labelRef.current) labelRef.current.textContent = 'Approved!';
      thumb!.classList.add('hidden');
    }

    function onStart(e: MouseEvent | TouchEvent) {
      if (track!.classList.contains('approved')) return;
      dragging = true;
      thumb!.classList.add('dragging');
      const clientX =
        'touches' in e ? e.touches[0].clientX : e.clientX;
      startX = clientX;
      thumbStartLeft = thumb!.offsetLeft - 3;
      e.preventDefault();
    }

    function onMove(e: MouseEvent | TouchEvent) {
      if (!dragging) return;
      const clientX =
        'touches' in e ? e.touches[0].clientX : e.clientX;
      const dx = clientX - startX;
      const newLeft = Math.max(
        0,
        Math.min(thumbStartLeft + dx, getMaxLeft()),
      );
      thumb!.style.left = newLeft + 3 + 'px';
      e.preventDefault();
    }

    function onEnd() {
      if (!dragging) return;
      dragging = false;
      thumb!.classList.remove('dragging');
      const currentLeft = thumb!.offsetLeft - 3;
      const maxLeft = getMaxLeft();
      if (currentLeft >= maxLeft * 0.8) {
        markApproved();
        sendCommand(cmd);
      } else {
        thumb!.style.left = '3px';
      }
    }

    thumb.addEventListener('touchstart', onStart, {
      passive: false,
      signal,
    });
    thumb.addEventListener('mousedown', onStart, { signal });
    document.addEventListener('touchmove', onMove, {
      passive: false,
      signal,
    });
    document.addEventListener('mousemove', onMove, { signal });
    document.addEventListener('touchend', onEnd, { signal });
    document.addEventListener('mouseup', onEnd, { signal });

    return () => ac.abort();
  }, [cmd]);

  const borderStyle = warn
    ? { borderColor: 'var(--warn, #ff9800)' }
    : undefined;
  const thumbStyle = warn
    ? { background: 'var(--warn, #ff9800)' }
    : undefined;

  return (
    <div class="slide-approve-wrap">
      <div
        class="slide-approve-track glass glass-interactive"
        ref={trackRef}
        style={borderStyle}
      >
        <div class="slide-approve-thumb" ref={thumbRef} style={thumbStyle}>
          <svg viewBox="0 0 24 24">
            <path
              d="M5 12h14m-6-6 6 6-6 6"
              stroke="currentColor"
              stroke-width="2.5"
              stroke-linecap="round"
              stroke-linejoin="round"
              fill="none"
            />
          </svg>
        </div>
        <div class="slide-approve-label" ref={labelRef}>
          {label}
        </div>
      </div>
    </div>
  );
}
