import { useCallback, useRef, useState, type ChangeEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/Button';
import {
  IconArrowDownToLine,
  IconArrowUpFromLine,
  IconTrash2,
} from '@/components/ui/icons';
import { ModelPricesPage } from '@/features/monitoring/ModelPricesPage';
import { useUsageData } from '@/features/monitoring/hooks/useUsageData';
import { isUsageImportFile } from '@/features/monitoring/model/monitoringCenterPageModel';
import { useRequestMonitoringAvailability } from '@/hooks/useRequestMonitoringAvailability';
import { useNotificationStore } from '@/stores/useNotificationStore';
import { downloadBlob } from '@/utils/download';
import styles from './DataPage.module.scss';

const CLEAR_DATA_CONFIRMATION = 'CLEAR DATA';
const CLEAR_DATA_UI_CONFIRMATION = '确认';

export function DataPage() {
  const { t } = useTranslation();
  const requestMonitoringAvailability = useRequestMonitoringAvailability();
  const { showNotification, showConfirmation } = useNotificationStore();
  const { exportUsage, importUsage, clearData } = useUsageData({ loadUsageEvents: false });
  const usageImportInputRef = useRef<HTMLInputElement | null>(null);
  const [usageExporting, setUsageExporting] = useState(false);
  const [usageImporting, setUsageImporting] = useState(false);
  const [clearingData, setClearingData] = useState(false);

  const resolveUsageTransferError = useCallback(
    (error: unknown) => {
      const rawMessage =
        error instanceof Error ? error.message : String(error || t('common.unknown_error'));
      return rawMessage === 'usage_import_export_requires_usage_service'
        ? t('usage_stats.import_export_requires_usage_service')
        : rawMessage;
    },
    [t]
  );

  const handleUsageExport = useCallback(async () => {
    setUsageExporting(true);
    try {
      const response = await exportUsage();
      const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
      downloadBlob({
        filename: response.filename || `usage-events-${timestamp}.jsonl`,
        blob: response.blob,
      });
      showNotification(t('usage_stats.export_success'), 'success');
    } catch (error: unknown) {
      const message = resolveUsageTransferError(error);
      showNotification(`${t('notification.download_failed')}${message ? `: ${message}` : ''}`, 'error');
    } finally {
      setUsageExporting(false);
    }
  }, [exportUsage, resolveUsageTransferError, showNotification, t]);

  const importUsageFile = useCallback(
    async (file: File) => {
      setUsageImporting(true);
      try {
        const result = await importUsage(file);
        const unsupported = result.unsupported ?? 0;
        showNotification(
          `${t('usage_stats.import_success', {
            added: result.added ?? 0,
            skipped: result.skipped ?? 0,
            total: result.total ?? 0,
            failed: result.failed ?? 0,
          })}${unsupported > 0 ? `, ${t('usage_stats.import_unsupported', { count: unsupported })}` : ''}`,
          (result.failed ?? 0) > 0 || unsupported > 0 ? 'warning' : 'success'
        );
        if (result.format?.startsWith('legacy') || (result.warnings ?? []).length > 0) {
          showNotification(t('usage_stats.import_legacy_warning'), 'warning');
        }
      } catch (error: unknown) {
        const message = resolveUsageTransferError(error);
        showNotification(`${t('notification.upload_failed')}${message ? `: ${message}` : ''}`, 'error');
      } finally {
        setUsageImporting(false);
      }
    },
    [importUsage, resolveUsageTransferError, showNotification, t]
  );

  const handleUsageImportClick = useCallback(() => {
    if (!requestMonitoringAvailability.available) {
      showNotification(t('usage_stats.import_export_requires_usage_service'), 'warning');
      return;
    }
    usageImportInputRef.current?.click();
  }, [requestMonitoringAvailability.available, showNotification, t]);

  const handleUsageImportChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      const file = event.target.files?.[0];
      event.target.value = '';
      if (!file) return;

      if (!isUsageImportFile(file)) {
        showNotification(t('usage_stats.import_invalid'), 'error');
        return;
      }
      showConfirmation({
        title: t('usage_stats.import_confirm_title'),
        message: t('usage_stats.import_confirm_body', { name: file.name }),
        confirmText: t('usage_stats.import'),
        variant: 'primary',
        onConfirm: () => importUsageFile(file),
      });
    },
    [importUsageFile, showConfirmation, showNotification, t]
  );

  const handleClearData = useCallback(() => {
    showConfirmation({
      title: t('data_page.clear_confirm_title', { defaultValue: '确认清空请求数据？' }),
      message: (
        <div className={styles.confirmBody}>
          <p>{t('data_page.clear_confirm_events', { defaultValue: '将删除当前已存储的请求用量数据和死信记录' })}</p>
          <p>
            {t('data_page.clear_confirm_preserved', {
              defaultValue: '登录凭据、系统配置、模型价格和 API Key 别名不会被清空',
            })}
          </p>
          <p>{t('data_page.clear_confirm_irreversible', { defaultValue: '此操作不可恢复，建议先导出备份' })}</p>
        </div>
      ),
      confirmText: t('data_page.clear_data', { defaultValue: '清空数据' }),
      confirmPhrase: CLEAR_DATA_UI_CONFIRMATION,
      confirmPhraseLabel: '',
      confirmPhrasePlaceholder: t('data_page.clear_confirm_phrase_placeholder', {
        defaultValue: '请输入“确认”以清空请求数据',
      }),
      variant: 'danger',
      onConfirm: async () => {
        setClearingData(true);
        try {
          const result = await clearData(CLEAR_DATA_CONFIRMATION);
          showNotification(
            t('data_page.clear_success', {
              defaultValue: '已清空 {{events}} 条请求数据和 {{deadLetters}} 条死信记录',
              events: result.cleared.usageEvents,
              deadLetters: result.cleared.deadLetters,
            }),
            'success'
          );
        } catch (error: unknown) {
          const message = resolveUsageTransferError(error);
          showNotification(
            `${t('data_page.clear_failed', { defaultValue: '清空数据失败' })}${message ? `: ${message}` : ''}`,
            'error'
          );
        } finally {
          setClearingData(false);
        }
      },
    });
  }, [clearData, resolveUsageTransferError, showConfirmation, showNotification, t]);

  const transferAvailable = requestMonitoringAvailability.available;

  const dataTransferActions = (
    <>
      <div data-model-prices-control-item="import">
        <Button
          size="xs"
          iconOnly
          onClick={handleUsageImportClick}
          loading={usageImporting}
          disabled={!transferAvailable}
          aria-label={t('usage_stats.import')}
          title={t('usage_stats.import')}
        >
          <span data-model-prices-control-icon="true">
            <IconArrowUpFromLine size={18} />
          </span>
        </Button>
        <span data-model-prices-control-label="true" aria-hidden="true">
          {t('usage_stats.import')}
        </span>
      </div>
      <div data-model-prices-control-item="export">
        <Button
          size="xs"
          variant="secondary"
          iconOnly
          onClick={() => void handleUsageExport()}
          loading={usageExporting}
          disabled={!transferAvailable}
          aria-label={t('usage_stats.export')}
          title={t('usage_stats.export')}
        >
          <span data-model-prices-control-icon="true">
            <IconArrowDownToLine size={18} />
          </span>
        </Button>
        <span data-model-prices-control-label="true" aria-hidden="true">
          {t('usage_stats.export')}
        </span>
      </div>
      <div data-model-prices-control-item="clear">
        <Button
          size="xs"
          variant="danger"
          iconOnly
          onClick={handleClearData}
          loading={clearingData}
          disabled={!transferAvailable}
          aria-label={t('data_page.clear_data', { defaultValue: '清空数据' })}
          title={t('data_page.clear_data', { defaultValue: '清空数据' })}
        >
          <span data-model-prices-control-icon="true">
            <IconTrash2 size={18} />
          </span>
        </Button>
        <span data-model-prices-control-label="true" aria-hidden="true">
          {t('data_page.clear_data', { defaultValue: '清空数据' })}
        </span>
      </div>
      <input
        ref={usageImportInputRef}
        type="file"
        accept=".json,.jsonl,.ndjson,application/json,application/x-ndjson"
        className={styles.hiddenInput}
        onChange={handleUsageImportChange}
      />
    </>
  );

  return (
    <div className={styles.page}>
      <section className={styles.modelPricesPanel}>
        <ModelPricesPage
          embedded
          dataPageLayout
          dataTransferActions={dataTransferActions}
        />
      </section>
    </div>
  );
}
