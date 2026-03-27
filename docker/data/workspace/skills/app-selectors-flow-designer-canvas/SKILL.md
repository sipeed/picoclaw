---
name: app-selectors-flow-designer-canvas
description: DOM selectors and component map for the Flow Designer Canvas page on dashboard.int3nt.info. Use when writing Playwright tests for this page.
---

# Flow Designer Canvas — Component Map

> Generated: 2026-03-27T06:40:03.873Z
> Selectors derived from actual DOM classes, IDs, and data-testid attributes.

### Flow Designer Canvas
**URL:** `/flow-designer/209`


**Text Content (5):**
- [p] "Organization" → `.org-title`
- [p] "Drag and drop nodes to start building your flow" → `.drop-text`
- [text-secondary] "Untitled"
- [p] "131%"
- [p] "Last saved just now"

**Buttons (1):**
- `page.locator('button:has-text("Publish")')`

**Icon Buttons (13):**
- mdi-pencil-outline (`mdi-pencil-outline`) → `.change-logo-btn`
- mdi-unfold-more-vertical (`mdi-unfold-more-vertical`) → `.code-editor-button`
- mdi-shape-rectangle-plus (`mdi-shape-rectangle-plus`) → `.nodes-button`
- mdi-magnify (`mdi-magnify`) → `.search-button`
- mdi-history (`mdi-history`) → `.history-button`
- mdi-timer-sand-empty (`mdi-timer-sand-empty`) → `.history-button`
- mdi-share-outline (`mdi-share-outline`) → `.history-button`
- mdi-image-refresh (`mdi-image-refresh`) → `.history-button`
- mdi-bug (`mdi-bug`) → `.history-button`
- mdi-content-save (`mdi-content-save`) → `.history-button`
- mdi-dots-vertical (`mdi-dots-vertical`) → `.action-button`
- mdi-undo (`mdi-undo`) → `.history-button`
- mdi-redo (`mdi-redo`) → `.history-button`

**Sidebar (8):**
- `page.locator('a:has-text("Dashboard")')`
- `page.locator('a:has-text("Flow Designer")')` ★
- `page.locator('a:has-text("Flow Tester")')`
- `page.locator('a:has-text("Knowledge Base")')`
- `page.locator('a:has-text("Logs")')`
- `page.locator('a:has-text("Add-Ons")')`
- `page.locator('a:has-text("Settings")')`
- `page.locator('a:has-text("Organization")')`

**Custom Elements & IDs (54):**

| Selector | Tag | Classes | Text |
|----------|-----|---------|------|
| `#app` | `div` | `` | HOrganizationTesting2026!DashboardFlow DesignerFlow TesterKn |
| `.topbar-intent` | `header` | `topbar-intent` | H |
| `.logo-container` | `div` | `logo-container` |  |
| `.logo-wrapper` | `div` | `logo-wrapper` |  |
| `.logo-intent` | `div` | `logo-intent` |  |
| `.change-logo-btn` | `button` | `change-logo-btn` |  |
| `.mdi` | `i` | `mdi notranslate` |  |
| `#menu-activator` | `div` | `avatar-container` | H |
| `.mdi` | `i` | `mdi notranslate avatar-chevron` |  |
| `.nav-drawer` | `nav` | `nav-drawer` | OrganizationTesting2026!DashboardFlow DesignerFlow TesterKno |
| `.org-title` | `p` | `org-title` | Organization |
| `.org-selector-wrapper` | `div` | `org-selector-wrapper` | Testing2026! |
| `.org-dropdown` | `div` | `org-dropdown` | Testing2026! |
| `.org-dropdown-trigger` | `div` | `org-dropdown-trigger` | Testing2026! |
| `.org-info` | `div` | `org-info` | Testing2026! |
| `.org-name` | `span` | `org-name` | Testing2026! |
| `.dropdown-arrow` | `div` | `dropdown-arrow` |  |
| `.[object` | `svg` | `[object SVGAnimatedString]` |  |
| `.[object` | `path` | `[object SVGAnimatedString]` |  |
| `.mdi` | `i` | `mdi notranslate arrow` |  |
| `.menu-item-container` | `div` | `menu-item-container` | Dashboard |
| `.mdi` | `i` | `mdi notranslate menu-item-icon` |  |
| `.menu-item` | `span` | `menu-item` | Dashboard |
| `.main-layout-margin-left` | `main` | `main-layout-margin-left` | STARTENDDrag and drop nodes to start building your flowUntit |
| `.dnd-flow` | `div` | `dnd-flow` | STARTENDDrag and drop nodes to start building your flowUntit |
| `.vue-flow` | `div` | `vue-flow basic-flow` | STARTENDDrag and drop nodes to start building your flowUntit |
| `.vue-flow__viewport` | `div` | `vue-flow__viewport vue-flow__container` | STARTEND |
| `.vue-flow__pane` | `div` | `vue-flow__pane vue-flow__container draggable` | STARTEND |
| `.vue-flow__transformationpane` | `div` | `vue-flow__transformationpane vue-flow__container` | STARTEND |
| `.vue-flow__edge-labels` | `div` | `vue-flow__edge-labels` |  |
| `.vue-flow__nodes` | `div` | `vue-flow__nodes vue-flow__container` | STARTEND |
| `.vue-flow__node` | `div` | `vue-flow__node vue-flow__node-Start nopan draggable selectable` | START |
| `#START` | `div` | `node-container` | START |
| `.terminal-node` | `div` | `terminal-node` | START |
| `.terminal-icon` | `div` | `terminal-icon` |  |
| `.vue-flow__handle` | `div` | `vue-flow__handle vue-flow__handle-bottom vue-flow__handle-null nodrag nopan source connectable connectablestart connectableend connectionindicator` |  |
| `.vue-flow__node` | `div` | `vue-flow__node vue-flow__node-End nopan draggable selectable` | END |
| `#END` | `div` | `node-container` | END |
| `.vue-flow__handle` | `div` | `vue-flow__handle vue-flow__handle-top vue-flow__handle-null nodrag nopan target connectable connectablestart connectableend connectionindicator` |  |
| `.dropzone-background` | `div` | `dropzone-background` | Drag and drop nodes to start building your flow |
| `.overlay` | `div` | `overlay` | Drag and drop nodes to start building your flow |
| `.drop-text` | `p` | `drop-text` | Drag and drop nodes to start building your flow |
| `.vue-flow__panel` | `div` | `vue-flow__panel top left` | Untitled |
| `.panel-container` | `div` | `panel-container` | Untitled |
| `.panel-container` | `div` | `panel-container relative` |  |
| `.code-editor-button` | `button` | `code-editor-button` |  |
| `.nodes-button` | `button` | `nodes-button` |  |
| `.vue-flow__panel` | `div` | `vue-flow__panel top right` | Publish |
| `.search-button` | `button` | `search-button` |  |
| `.history-button` | `button` | `history-button` |  |

---

