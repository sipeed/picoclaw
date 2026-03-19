import './styles.css';
import 'highlight.js/styles/github-dark.min.css';
import { render } from 'preact';
import { App } from './app';

declare global {
  interface Window {
    Telegram: { WebApp: any };
    ORCH_ENABLED: boolean;
    MAP_POSITIONS: any;
    loadMapAsset: (cb: () => void) => void;
    drawMap: (ctx: CanvasRenderingContext2D) => void;
  }
}

const tg = window.Telegram.WebApp;
tg.ready();

const root = document.getElementById('app');
if (root) {
  render(<App />, root);
}
