import { describe, expect, it } from 'vitest';
import routesSource from './router/MainRoutes.tsx?raw';
import layoutSource from './components/layout/MainLayout.tsx?raw';

describe('reduced management panel', () => {
  it('keeps the overview, usage, request details and data routes in the main shell', () => {
    expect(routesSource).toContain("path: '/'");
    expect(routesSource).toContain("path: '/overview'");
    expect(routesSource).toContain("path: '/usage'");
    expect(routesSource).toContain("path: '/request-details'");
    expect(routesSource).toContain("path: '/data'");
    expect(routesSource).not.toMatch(/\/(config|ai-providers|auth-files|oauth|quota|logs|system|codex-inspection|model-prices)/);
  });

  it('keeps overview, usage, request details and data entries in top navigation', () => {
    expect(layoutSource).toContain("path: '/overview'");
    expect(layoutSource).toContain("path: '/usage'");
    expect(layoutSource).toContain("path: '/request-details'");
    expect(layoutSource).toContain("path: '/data'");
    expect(layoutSource).toContain('topbar-nav');
    expect(layoutSource).not.toContain('<aside className={`sidebar');
    expect(layoutSource).not.toContain('STORAGE_KEY_SIDEBAR');
    expect(layoutSource).not.toMatch(/\/(config|ai-providers|auth-files|oauth|quota|logs|system|codex-inspection|model-prices)/);
  });
});
