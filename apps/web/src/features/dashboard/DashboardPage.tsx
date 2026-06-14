import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useAuthStore } from '@/stores/useAuthStore';
import { useConfigStore } from '@/stores/useConfigStore';
import { logsApi, type ErrorLogFile } from '@/services/api/logs';
import {
  usageServiceApi,
  type ApiKeyAlias,
  type UsageServiceStatus,
} from '@/services/api/usageService';
import { useHeaderRefresh } from '@/hooks/useHeaderRefresh';
import { loadMonitoringMetaPayload } from '@/features/monitoring/services/monitoringMetaService';
import { buildMonitoringAuthMetaMap } from '@/features/monitoring/model/authMeta';
import { buildAuthFileMapFromMeta } from '@/features/monitoring/model/sourceDisplay';
import type { MonitoringChannelMeta } from '@/features/monitoring/model/types';
import { buildSourceInfoMap } from '@/utils/sourceResolver';
import type { AuthFileItem } from '@/types/authFile';
import { VersionCard } from './components/VersionCard';
import { CollectorStatusCard } from './components/CollectorStatusCard';
import { ChannelHealthCard, RecentFailuresCard } from './components/HealthAlertsCard';
import { RollingRateCard } from './components/RollingRateCard';
import { useDashboardUsageSummary } from './hooks/useDashboardUsageSummary';
import styles from './DashboardPage.module.scss';

interface DashboardDisplayMeta {
  authFiles: AuthFileItem[];
  channels: MonitoringChannelMeta[];
  apiKeyAliases: ApiKeyAlias[];
}

const HEALTH_REFRESH_INTERVAL_MS = 60_000;

export function DashboardPage() {
  const { t } = useTranslation();
  const connectionStatus = useAuthStore((state) => state.connectionStatus);
  const serverVersion = useAuthStore((state) => state.serverVersion);
  const serverBuildDate = useAuthStore((state) => state.serverBuildDate);
  const apiBase = useAuthStore((state) => state.apiBase);
  const managementKey = useAuthStore((state) => state.managementKey);
  const config = useConfigStore((state) => state.config);
  const usageSummary = useDashboardUsageSummary();
  const refreshUsageSummary = usageSummary.refresh;

  const [currentTime, setCurrentTime] = useState(() => new Date());
  const [cardRefreshSignal, setCardRefreshSignal] = useState(0);
  const [collectorStatus, setCollectorStatus] = useState<UsageServiceStatus | null>(null);
  const [collectorLoading, setCollectorLoading] = useState(false);
  const [collectorError, setCollectorError] = useState('');
  const [errorLogs, setErrorLogs] = useState<ErrorLogFile[]>([]);
  const [errorLogsLoading, setErrorLogsLoading] = useState(false);
  const [managerCpaBase, setManagerCpaBase] = useState('');
  const [displayMeta, setDisplayMeta] = useState<DashboardDisplayMeta>({
    authFiles: [],
    channels: [],
    apiKeyAliases: [],
  });

  // Update time every 60 seconds
  useEffect(() => {
    const id = setInterval(() => {
      setCurrentTime(new Date());
    }, 60_000);
    return () => clearInterval(id);
  }, []);

  const usageEnabled = usageSummary.enabled;
  const usageServiceBase = usageSummary.serviceBase;
  const authMetaMap = useMemo(
    () => buildMonitoringAuthMetaMap(displayMeta.authFiles),
    [displayMeta.authFiles]
  );
  const authFileMap = useMemo(() => buildAuthFileMapFromMeta(authMetaMap), [authMetaMap]);
  const sourceInfoMap = useMemo(
    () =>
      buildSourceInfoMap({
        geminiApiKeys: config?.geminiApiKeys || [],
        claudeApiKeys: config?.claudeApiKeys || [],
        codexApiKeys: config?.codexApiKeys || [],
        vertexApiKeys: config?.vertexApiKeys || [],
        openaiCompatibility: config?.openaiCompatibility || [],
      }),
    [config]
  );
  const channelByAuthIndex = useMemo(() => {
    const map = new Map<string, MonitoringChannelMeta>();
    displayMeta.channels.forEach((channel) => {
      channel.authIndices.forEach((authIndex) => {
        map.set(authIndex, channel);
      });
    });
    return map;
  }, [displayMeta.channels]);
  const apiKeyAliasMap = useMemo(() => {
    const map = new Map<string, string>();
    displayMeta.apiKeyAliases.forEach((item) => {
      const hash = item.apiKeyHash?.trim().toLowerCase();
      const alias = item.alias?.trim();
      if (hash && alias) {
        map.set(hash, alias);
      }
    });
    return map;
  }, [displayMeta.apiKeyAliases]);

  const refreshHealth = useCallback(async () => {
    if (!usageEnabled || !usageServiceBase) {
      setCollectorStatus(null);
      setCollectorError('');
      setCollectorLoading(false);
      setErrorLogs([]);
      setErrorLogsLoading(false);
      setManagerCpaBase('');
      setDisplayMeta({ authFiles: [], channels: [], apiKeyAliases: [] });
      return;
    }

    setCollectorLoading(true);
    setErrorLogsLoading(true);

    const [collectorResult, logsResult, managerConfigResult, metaResult, aliasesResult] =
      await Promise.allSettled([
        usageServiceApi.getStatus(usageServiceBase, managementKey),
        logsApi.fetchErrorLogs(),
        usageServiceApi.getManagerConfig(usageServiceBase, managementKey),
        loadMonitoringMetaPayload(config),
        usageServiceApi.getApiKeyAliases(usageServiceBase, managementKey),
      ]);

    if (collectorResult.status === 'fulfilled') {
      setCollectorStatus(collectorResult.value);
      setCollectorError('');
    } else {
      setCollectorStatus(null);
      const reason = collectorResult.reason;
      setCollectorError(reason instanceof Error ? reason.message : String(reason));
    }
    setCollectorLoading(false);

    if (logsResult.status === 'fulfilled') {
      setErrorLogs(Array.isArray(logsResult.value.files) ? logsResult.value.files : []);
    } else {
      setErrorLogs([]);
    }
    setErrorLogsLoading(false);

    setManagerCpaBase(
      managerConfigResult.status === 'fulfilled'
        ? managerConfigResult.value.config.cpaConnection?.cpaBaseUrl || apiBase || ''
        : apiBase || ''
    );

    setDisplayMeta((current) => ({
      authFiles: metaResult.status === 'fulfilled' ? metaResult.value.authFiles : current.authFiles,
      channels: metaResult.status === 'fulfilled' ? metaResult.value.channels : current.channels,
      apiKeyAliases:
        aliasesResult.status === 'fulfilled' && Array.isArray(aliasesResult.value.items)
          ? aliasesResult.value.items
          : current.apiKeyAliases,
    }));
  }, [apiBase, config, managementKey, usageEnabled, usageServiceBase]);

  const refreshDashboard = useCallback(async () => {
    setCurrentTime(new Date());
    setCardRefreshSignal((value) => value + 1);
    await Promise.all([refreshUsageSummary(), refreshHealth()]);
  }, [refreshHealth, refreshUsageSummary]);


  useEffect(() => {
    const timer = window.setTimeout(() => {
      void refreshHealth();
    }, 0);
    return () => window.clearTimeout(timer);
  }, [refreshHealth]);

  useEffect(() => {
    if (!usageEnabled) return;
    const timer = window.setInterval(() => {
      void refreshHealth();
    }, HEALTH_REFRESH_INTERVAL_MS);
    return () => window.clearInterval(timer);
  }, [refreshHealth, usageEnabled]);

  useHeaderRefresh(refreshDashboard);

  // 监听 CPA Base 切换事件
  useEffect(() => {
    const handleCpaBaseChanged = () => {
      void refreshDashboard();
    };

    window.addEventListener('cpa-base-changed', handleCpaBaseChanged);
    return () => window.removeEventListener('cpa-base-changed', handleCpaBaseChanged);
  }, [refreshDashboard]);

  return (
    <>
      {/* 1. System Overview Row */}
      <section className={styles.overviewRow}>
        <VersionCard
          appVersion={__APP_VERSION__ || t('dashboard.version_unknown')}
          apiVersion={serverVersion || t('dashboard.version_unknown')}
          cpaBase={managerCpaBase || apiBase || ''}
          apiBase={apiBase || ''}
          currentTime={currentTime}
          serverBuildDate={serverBuildDate || undefined}
          connectionStatus={connectionStatus}
          refreshSignal={cardRefreshSignal}
          usageEnabled={usageSummary.enabled}
          usageLoading={usageSummary.loading}
          usageError={usageSummary.error}
          collectorStatus={collectorStatus}
          collectorLoading={collectorLoading}
          collectorError={collectorError}
          errorLogCount={errorLogs.length}
          errorLogsLoading={errorLogsLoading}
        />
      </section>

      {/* 3. Health & Collector Status */}
      {usageSummary.enabled && (
        <section className={styles.overviewDataGrid}>
          <RollingRateCard
            summary={usageSummary.summary}
            loading={usageSummary.loading}
            error={usageSummary.error}
          />
          <ChannelHealthCard
            loading={usageSummary.loading}
            channelHealth={usageSummary.channelHealth}
            authMetaMap={authMetaMap}
            authFileMap={authFileMap}
            sourceInfoMap={sourceInfoMap}
            channelByAuthIndex={channelByAuthIndex}
            apiKeyAliasMap={apiKeyAliasMap}
          />
          <RecentFailuresCard
            loading={usageSummary.loading}
            recentFailures={usageSummary.recentFailures}
            authMetaMap={authMetaMap}
            authFileMap={authFileMap}
            sourceInfoMap={sourceInfoMap}
            channelByAuthIndex={channelByAuthIndex}
            apiKeyAliasMap={apiKeyAliasMap}
          />
          <CollectorStatusCard
            enabled={usageSummary.enabled}
            serviceBase={usageSummary.serviceBase}
            managementKey={managementKey}
            refreshSignal={cardRefreshSignal}
            status={collectorStatus}
            loading={collectorLoading}
            error={collectorError}
          />
        </section>
      )}

    </>
  );
}
