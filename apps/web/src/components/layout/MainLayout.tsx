import { ReactNode, SVGProps, useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { NavLink, useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/Button';
import { PageTransition } from '@/components/common/PageTransition';
import { MainRoutes } from '@/router/MainRoutes';
import {
  IconDatabaseZap,
  IconSidebarDashboard,
  IconSidebarMonitor,
  IconSidebarUsage,
} from '@/components/ui/icons';
import { INLINE_LOGO_JPEG } from '@/assets/logoInline';
import { useAuthStore } from '@/stores/useAuthStore';
import { useLanguageStore } from '@/stores/useLanguageStore';
import { useNotificationStore } from '@/stores/useNotificationStore';
import { useThemeStore } from '@/stores/useThemeStore';
import { triggerHeaderRefresh } from '@/hooks/useHeaderRefresh';
import { LANGUAGE_LABEL_KEYS, LANGUAGE_ORDER } from '@/utils/constants';
import { isSupportedLanguage } from '@/utils/language';
import type { Theme } from '@/types';

const NAV_ICON_SIZE = 20;

const navIcons: Record<string, ReactNode> = {
  overview: <IconSidebarDashboard size={NAV_ICON_SIZE} />,
  usage: <IconSidebarUsage size={NAV_ICON_SIZE} />,
  requestDetails: <IconSidebarMonitor size={NAV_ICON_SIZE} />,
  data: <IconDatabaseZap size={NAV_ICON_SIZE} />,
};

const headerIconProps: SVGProps<SVGSVGElement> = {
  width: 16,
  height: 16,
  viewBox: '0 0 24 24',
  fill: 'none',
  stroke: 'currentColor',
  strokeWidth: 2,
  strokeLinecap: 'round',
  strokeLinejoin: 'round',
  'aria-hidden': 'true',
  focusable: 'false',
};

const headerIcons = {
  refresh: (
    <svg {...headerIconProps}>
      <path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8" />
      <path d="M21 3v5h-5" />
    </svg>
  ),
  language: (
    <svg {...headerIconProps}>
      <circle cx="12" cy="12" r="10" />
      <path d="M2 12h20" />
      <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
    </svg>
  ),
  sun: (
    <svg {...headerIconProps}>
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2" />
      <path d="M12 20v2" />
      <path d="m4.93 4.93 1.41 1.41" />
      <path d="m17.66 17.66 1.41 1.41" />
      <path d="M2 12h2" />
      <path d="M20 12h2" />
      <path d="m6.34 17.66-1.41 1.41" />
      <path d="m19.07 4.93-1.41 1.41" />
    </svg>
  ),
  moon: (
    <svg {...headerIconProps}>
      <path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9z" />
    </svg>
  ),
  autoTheme: (
    <svg {...headerIconProps}>
      <rect x="4" y="5" width="16" height="11" rx="2" />
      <path d="M8 21h8" />
      <path d="M12 16v5" />
      <path d="M9 11a3 3 0 0 1 5.2-2" />
      <path d="M14.5 7v2h-2" />
      <path d="M15 11a3 3 0 0 1-5.2 2" />
      <path d="M9.5 15v-2h2" />
    </svg>
  ),
  logout: (
    <svg {...headerIconProps}>
      <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
      <path d="m16 17 5-5-5-5" />
      <path d="M21 12H9" />
    </svg>
  ),
};

const THEME_OPTIONS: Array<{ key: Theme; labelKey: string; icon: ReactNode }> = [
  { key: 'auto', labelKey: 'theme.auto', icon: headerIcons.autoTheme },
  { key: 'white', labelKey: 'theme.white', icon: headerIcons.sun },
  { key: 'dark', labelKey: 'theme.dark', icon: headerIcons.moon },
];

type NavItem = {
  path: string;
  label: string;
  icon: ReactNode;
};

export function MainLayout() {
  const { t } = useTranslation();
  const { showNotification } = useNotificationStore();
  const location = useLocation();
  const logout = useAuthStore((state) => state.logout);
  const theme = useThemeStore((state) => state.theme);
  const setTheme = useThemeStore((state) => state.setTheme);
  const language = useLanguageStore((state) => state.language);
  const setLanguage = useLanguageStore((state) => state.setLanguage);

  const [languageMenuOpen, setLanguageMenuOpen] = useState(false);
  const [themeMenuOpen, setThemeMenuOpen] = useState(false);
  const contentRef = useRef<HTMLDivElement | null>(null);
  const languageMenuRef = useRef<HTMLDivElement | null>(null);
  const themeMenuRef = useRef<HTMLDivElement | null>(null);
  const headerRef = useRef<HTMLElement | null>(null);

  const fullBrandName = 'CPA Statistics';
  const abbrBrandName = 'CPAST';

  useLayoutEffect(() => {
    const updateHeaderHeight = () => {
      const height = headerRef.current?.offsetHeight;
      if (height) {
        document.documentElement.style.setProperty('--header-height', `${height}px`);
      }
    };
    updateHeaderHeight();
    const resizeObserver =
      typeof ResizeObserver !== 'undefined' && headerRef.current
        ? new ResizeObserver(updateHeaderHeight)
        : null;
    if (resizeObserver && headerRef.current) resizeObserver.observe(headerRef.current);
    window.addEventListener('resize', updateHeaderHeight);
    return () => {
      resizeObserver?.disconnect();
      window.removeEventListener('resize', updateHeaderHeight);
    };
  }, []);

  useLayoutEffect(() => {
    const updateContentCenter = () => {
      const el = contentRef.current;
      if (!el) return;
      const rect = el.getBoundingClientRect();
      document.documentElement.style.setProperty('--content-center-x', `${rect.left + rect.width / 2}px`);
    };
    updateContentCenter();
    const resizeObserver =
      typeof ResizeObserver !== 'undefined' && contentRef.current
        ? new ResizeObserver(updateContentCenter)
        : null;
    if (resizeObserver && contentRef.current) resizeObserver.observe(contentRef.current);
    window.addEventListener('resize', updateContentCenter);
    return () => {
      resizeObserver?.disconnect();
      window.removeEventListener('resize', updateContentCenter);
      document.documentElement.style.removeProperty('--content-center-x');
    };
  }, []);

  useEffect(() => {
    if (!languageMenuOpen) return;
    const handlePointerDown = (event: MouseEvent) => {
      if (!languageMenuRef.current?.contains(event.target as Node)) setLanguageMenuOpen(false);
    };
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setLanguageMenuOpen(false);
    };
    document.addEventListener('mousedown', handlePointerDown);
    document.addEventListener('keydown', handleEscape);
    return () => {
      document.removeEventListener('mousedown', handlePointerDown);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [languageMenuOpen]);

  useEffect(() => {
    if (!themeMenuOpen) return;
    const handlePointerDown = (event: MouseEvent) => {
      if (!themeMenuRef.current?.contains(event.target as Node)) setThemeMenuOpen(false);
    };
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setThemeMenuOpen(false);
    };
    document.addEventListener('mousedown', handlePointerDown);
    document.addEventListener('keydown', handleEscape);
    return () => {
      document.removeEventListener('mousedown', handlePointerDown);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [themeMenuOpen]);

  const toggleLanguageMenu = useCallback(() => {
    setLanguageMenuOpen((prev) => !prev);
    setThemeMenuOpen(false);
  }, []);

  const toggleThemeMenu = useCallback(() => {
    setThemeMenuOpen((prev) => !prev);
    setLanguageMenuOpen(false);
  }, []);

  const handleRefreshAll = async () => {
    const results = await Promise.allSettled([triggerHeaderRefresh()]);
    const rejected = results.find((result) => result.status === 'rejected');
    if (rejected && rejected.status === 'rejected') {
      const reason = rejected.reason;
      const message =
        typeof reason === 'string' ? reason : reason instanceof Error ? reason.message : '';
      showNotification(`${t('notification.refresh_failed')}${message ? `: ${message}` : ''}`, 'error');
      return;
    }
    showNotification(t('notification.data_refreshed'), 'success');
  };

  const navItems: NavItem[] = [
    { path: '/overview', label: t('nav.overview'), icon: navIcons.overview },
    { path: '/usage', label: t('nav.usage'), icon: navIcons.usage },
    { path: '/request-details', label: t('nav.request_details'), icon: navIcons.requestDetails },
    { path: '/data', label: t('nav.data'), icon: navIcons.data },
  ];
  const navOrder = navItems.map((item) => item.path);
  const normalizeMainPath = (pathname: string) => {
    const trimmedPath =
      pathname.length > 1 && pathname.endsWith('/') ? pathname.slice(0, -1) : pathname;
    if (trimmedPath === '/' || trimmedPath === '/dashboard') return '/overview';
    if (trimmedPath === '/monitoring') return '/usage';
    return trimmedPath;
  };
  const getRouteOrder = (pathname: string) => {
    const normalizedPath = normalizeMainPath(pathname);
    const exactIndex = navOrder.indexOf(normalizedPath);
    if (exactIndex !== -1) return exactIndex;
    const parentIndex = navOrder.findIndex((path) => normalizedPath.startsWith(`${path}/`));
    return parentIndex === -1 ? null : parentIndex;
  };

  const normalizedLocationPath =
    location.pathname.length > 1 && location.pathname.endsWith('/')
      ? location.pathname.slice(0, -1)
      : location.pathname;
  const currentPath = normalizeMainPath(normalizedLocationPath);
  const matchesNavPath = (item: NavItem, pathname: string) =>
    pathname === item.path || pathname.startsWith(`${item.path}/`);

  return (
    <div className="app-shell app-shell--topnav">
      <header className="main-header topbar" ref={headerRef}>
        <div className="navbar">
          <div className="topbar-brand" title={fullBrandName}>
            <NavLink to="/" className="topbar-brand-link" aria-label={fullBrandName}>
              <img src={INLINE_LOGO_JPEG} alt="CPAMC logo" className="topbar-brand-logo" />
              <span className="topbar-brand-copy">
                <span className="topbar-brand-title">{fullBrandName}</span>
                <span className="topbar-brand-subtitle">{abbrBrandName}</span>
              </span>
            </NavLink>
          </div>

          <nav className="topbar-nav" aria-label={t('common.navigation', { defaultValue: 'Navigation' })}>
            {navItems.map((item) => (
              <NavLink
                key={item.path}
                to={item.path}
                end={item.path === '/'}
                className={({ isActive }) =>
                  `topbar-nav-item ${isActive || matchesNavPath(item, currentPath) ? 'active' : ''}`
                }
                aria-current={matchesNavPath(item, currentPath) ? 'page' : undefined}
              >
                <span className="topbar-nav-icon">{item.icon}</span>
                <span className="topbar-nav-label">{item.label}</span>
              </NavLink>
            ))}
          </nav>

          <div className="navbar-right topbar-actions">
            <Button variant="ghost" size="sm" onClick={handleRefreshAll} title={t('header.refresh_all')} aria-label={t('header.refresh_all')}>
              {headerIcons.refresh}
            </Button>

            <div className={`language-menu ${languageMenuOpen ? 'open' : ''}`} ref={languageMenuRef}>
              <Button
                variant="ghost"
                size="sm"
                onClick={toggleLanguageMenu}
                title={t('language.switch')}
                aria-label={t('language.switch')}
                aria-haspopup="menu"
                aria-expanded={languageMenuOpen}
              >
                {headerIcons.language}
              </Button>
              {languageMenuOpen && (
                <div className="notification entering language-menu-popover" role="menu" aria-label={t('language.switch')}>
                  {LANGUAGE_ORDER.map((lang) => (
                    <button
                      key={lang}
                      type="button"
                      className={`language-menu-option ${language === lang ? 'active' : ''}`}
                      onClick={() => {
                        if (isSupportedLanguage(lang)) setLanguage(lang);
                        setLanguageMenuOpen(false);
                      }}
                      role="menuitemradio"
                      aria-checked={language === lang}
                    >
                      <span>{t(LANGUAGE_LABEL_KEYS[lang])}</span>
                    </button>
                  ))}
                </div>
              )}
            </div>

            <div className={`theme-menu ${themeMenuOpen ? 'open' : ''}`} ref={themeMenuRef}>
              <Button
                variant="ghost"
                size="sm"
                onClick={toggleThemeMenu}
                title={t('theme.switch')}
                aria-label={t('theme.switch')}
                aria-haspopup="menu"
                aria-expanded={themeMenuOpen}
              >
                {theme === 'auto' ? headerIcons.autoTheme : theme === 'dark' ? headerIcons.moon : headerIcons.sun}
              </Button>
              {themeMenuOpen && (
                <div className="notification entering theme-menu-popover" role="menu" aria-label={t('theme.switch')}>
                  {THEME_OPTIONS.map((option) => (
                    <button
                      key={option.key}
                      type="button"
                      className={`theme-option ${theme === option.key ? 'active' : ''}`}
                      onClick={() => {
                        setTheme(option.key);
                        setThemeMenuOpen(false);
                      }}
                      role="menuitemradio"
                      aria-checked={theme === option.key}
                      title={t(option.labelKey)}
                      aria-label={t(option.labelKey)}
                    >
                      <span className="theme-option-icon">{option.icon}</span>
                    </button>
                  ))}
                </div>
              )}
            </div>

            <Button variant="ghost" size="sm" onClick={logout} title={t('header.logout')} aria-label={t('header.logout')}>
              {headerIcons.logout}
            </Button>
          </div>
        </div>
      </header>

      <div className="main-body">
        <div className="content" ref={contentRef}>
          <main className="main-content">
            <PageTransition
              render={(location) => <MainRoutes location={location} />}
              getRouteOrder={getRouteOrder}
              getTransitionVariant={() => 'none'}
              scrollContainerRef={contentRef}
            />
          </main>
        </div>
      </div>
    </div>
  );
}
