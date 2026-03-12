---
name: picoclaw
description: End-to-End (E2E) testing for the Intent Platform. Use when asked to test complex user flows across Dashboard, Flow Designer, and Knowledge Base. Focuses on Vuetify-specific selector strategies and monorepo integration.
---

# Picoclaw E2E Testing Skill

This skill provides specialized guidance for browser-based automation within the Intent Platform monorepo (Vue 3, Vuetify, FastAPI).

## 🎯 Selector Strategy (Vuetify & Custom Components)
Since `data-testid` is not consistently used, prioritize selectors in this order:

1.  **ARIA Roles & Labels**: 
    - Buttons: `getByRole('button', { name: /login/i })`
    - Inputs: `getByLabel(/email address/i)` or `getByPlaceholder(/enter text/i)`
2.  **Vuetify Class Selectors**:
    - Primary Button: `button.bg-btn-primary`
    - Text Fields: `.v-text-field input`
    - Selectors: `.v-select` (Note: clicking these often opens a menu in a `Teleport` at the bottom of `body`)
3.  **Scoped Context**: 
    - Use `v-card` headers or `v-toolbar` titles to scope selectors when multiple forms are present.
    - Example: `wrapper.find('.login-card').find('button')`

## 🔐 Auth & Organization Flow
E2E tests must handle the multi-step login and organization selection process:

1.  **Login**: Input email/password in `LoginPage.vue`. 
2.  **Organization Selection**: 
    - If redirected to `/?select_org`, interact with the `OrganizationSelector.vue`.
    - Note: The dropdown menu is often teleported to the `body`, so wait for `.org-dropdown-menu` to appear.
3.  **SSO**: For SSO testing, look for `v-btn` with `mdi-shield-account` icon or text matching "SSO".

## 🚀 Navigation & Component Interaction
Detailed guidance for core platform features:

### Flow Designer
- **Nodes**: Nodes are rendered within `@vue-flow/core`. Use coordinates or unique labels to identify them.
- **Canvas Interaction**: Use `drag-and-drop` patterns for adding nodes from the sidebar to the canvas.
- **Modals**: Configuration typically happens in `NodeConfigurationModal.vue` or `EdgeConfigurationModal.vue`.

### Knowledge Base
- **Uploads**: Use file chooser events for `DragDropFileUpload.vue`.
- **Status Pills**: Verify processing states using `.knowledge-base-status-pill`.

### Global Shared Components
- **Modals**: Always wait for `.v-dialog--active` or the specific modal class (e.g., `.form-modal`).
- **Snackbars**: Assert feedback messages using `.v-snackbar` content.

## ⏳ Waiting & Synchronization
- **API Responses**: Wait for specific Supabase or FastAPI endpoints using `page.waitForResponse(url)`.
- **Transitions**: Vuetify components often have entrance animations (e.g., `v-fade-transition`). Ensure elements are "stable" before clicking.
- **Loading States**: Check for `v-progress-circular` or `loading` classes in buttons (`.v-btn--loading`) to ensure operations are complete.

## 🧪 Mocking & Data Strategy
The platform uses **MSW (Mock Service Worker)** for network-level mocking. 

- **E2E Mocking**: Use `page.addInitScript()` to inject MSW workers if testing in a browser without a full backend.
- **Dynamic Data**: Intercept Supabase calls to simulate different organization roles (Owner vs. Member) to test UI permission gating.
- **File Storage**: Mock GCS/S3 signed URLs to verify file previews without actual bucket access.

## 📦 Monorepo Context for E2E
Tests should be aware of the inter-service dependencies:
- **Dashboard (`services/dashboard`)**: The primary target for UI tests.
- **Python Service (`services/python`)**: The backend for most business logic. E2E tests should ideally run against a dev environment or a local docker-compose stack.
- **Supabase**: Handles auth and real-time. Use `auth.loginWithSSO` mocks to avoid external redirects.

## 📊 Reporting Format
Follow this format for every test execution report:

✅ **PASS**  Step N: [Short description of action/assertion]
❌ **FAIL**  Step N: [Reason for failure + current URL/State]

**Result: X/Y PASSED**
**Navigation Path**: Start -> Login -> [Target Page]

## 🛠 Debugging Checklist
- Is the element inside a `v-menu` or `v-dialog`? (Check the bottom of the DOM/Teleport).
- Is the button disabled? (Check `.v-btn--disabled` or `:disabled` attribute).
- Is the form validation visible? (Check for `.v-messages__message`).
- Are the translations loaded? (Wait for i18n keys to be replaced by text).
