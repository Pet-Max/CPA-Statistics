import { Fragment, type ReactNode } from 'react';
import type { TFunction } from 'i18next';
import { DropdownMenu, type DropdownMenuItem } from '@/components/ui/DropdownMenu';
import { Input } from '@/components/ui/Input';
import { Select, type SelectOption } from '@/components/ui/Select';
import {
  IconChevronDown,
  IconChevronUp,
  IconCrosshair,
  IconInfo,
  IconMoreVertical,
  IconRefreshCw,
  IconSearch,
} from '@/components/ui/icons';
import {
  AccountExpandedDetails,
  AccountOverviewCard,
  AccountStatusBadge,
  AccountSummaryPrimary,
} from '@/features/monitoring/components/AccountOverviewCard';
import { PaginationControls } from '@/features/monitoring/components/MonitoringShared';
import {
  buildAccountSummaryMetrics,
  formatPercent,
  getAccountStatusTone,
  getSuccessRateClassName,
  type AccountQuotaState,
} from '@/features/monitoring/components/accountOverviewPresentation';
import { MonitoringPanel } from '@/features/monitoring/components/MonitoringPanel';
import {
  type AccountSortKey,
  type AccountSortState,
  type MonitoringAccountAuthState,
  type MonitoringAccountOverviewMode,
} from '@/features/monitoring/accountOverviewState';
import type { MonitoringAccountRow } from '@/features/monitoring/hooks/useMonitoringData';
import type { StatusBarData } from '@/utils/recentRequests';
import styles from '../MonitoringCenterPage.module.scss';

type AccountOverviewColumn = {
  key: string;
  label: string;
  sortKey?: AccountSortKey;
};

type PaginationState<T> = {
  currentPage: number;
  totalPages: number;
  pageItems: T[];
  startItem: number;
  endItem: number;
};

type AccountOverviewPanelProps = {
  embedded?: boolean;
  mode: MonitoringAccountOverviewMode;
  searchInput: string;
  columns: AccountOverviewColumn[];
  rows: MonitoringAccountRow[];
  pagination: PaginationState<MonitoringAccountRow>;
  accountSort: AccountSortState;
  accountSortOptions: ReadonlyArray<SelectOption>;
  expandedAccounts: Record<string, boolean>;
  focusedAccount: string | null;
  accountAuthStateByRowId: ReadonlyMap<string, MonitoringAccountAuthState>;
  accountStatusDataByRowId: ReadonlyMap<string, StatusBarData>;
  emptyAccountStatusData: StatusBarData;
  accountQuotaStates: Record<string, AccountQuotaState>;
  accountStatusUpdating: Record<string, boolean>;
  accountPageSize: number;
  accountPageSizeOptions: readonly number[];
  accountOverviewScopeText: string;
  hasPrices: boolean;
  overallLoading: boolean;
  locale: string;
  emptyState: ReactNode;
  t: TFunction;
  onSearchChange: (value: string) => void;
  onRefreshAll: () => void | Promise<void>;
  onAccountSortKeyChange: (key: AccountSortKey) => void;
  onModeChange: (mode: MonitoringAccountOverviewMode) => void;
  onAccountSort: (sortKey: AccountSortKey) => void;
  onAccountStatusToggle: (row: MonitoringAccountRow, enabled: boolean) => void | Promise<void>;
  onLoadAccountQuota: (account: string, force: boolean) => void | Promise<void>;
  onToggleExpanded: (rowId: string, account: string) => void;
  onFocusAccount: (row: MonitoringAccountRow) => void;
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
};

export type AccountOverviewPanelActionsProps = Pick<
  AccountOverviewPanelProps,
  | 'mode'
  | 'searchInput'
  | 'accountSort'
  | 'accountSortOptions'
  | 'overallLoading'
  | 't'
  | 'onSearchChange'
  | 'onRefreshAll'
  | 'onAccountSortKeyChange'
  | 'onModeChange'
>;

const EMPTY_ACCOUNT_AUTH_STATE: MonitoringAccountAuthState = {
  files: [],
  toggleableFileNames: [],
  enabledState: 'unavailable',
};

const getAccountColumnInfo = (columnKey: string, t: TFunction) => {
  if (columnKey === 'status') {
    return t('monitoring.account_overview_status_hint');
  }
  if (columnKey === 'success-rate') {
    return t('monitoring.account_overview_success_rate_hint');
  }
  return '';
};

function AccountColumnLabel({
  column,
  t,
}: {
  column: AccountOverviewColumn;
  t: TFunction;
}) {
  const info = getAccountColumnInfo(column.key, t);

  return (
    <span className={styles.tableHeaderWithInfo}>
      <span>{column.label}</span>
      {info ? (
        <span title={info}>
          <IconInfo size={13} className={styles.tableHeaderInfoIcon} aria-label={info} />
        </span>
      ) : null}
    </span>
  );
}

export function AccountOverviewPanelActions({
  mode,
  searchInput,
  accountSort,
  accountSortOptions,
  overallLoading,
  t,
  onSearchChange,
  onRefreshAll,
  onAccountSortKeyChange,
  onModeChange,
}: AccountOverviewPanelActionsProps) {
  return (
    <div className={styles.accountOverviewHeaderActions}>
      <div className={styles.accountOverviewToolbarRow}>
        <div className={styles.accountOverviewSearchWrap}>
          <Input
            value={searchInput}
            onChange={(event) => onSearchChange(event.target.value)}
            placeholder={t('monitoring.account_overview_search_placeholder')}
            className={styles.accountOverviewSearchInput}
            rightElement={<IconSearch size={16} />}
            aria-label={t('monitoring.account_overview_search_placeholder')}
          />
        </div>
        <button
          type="button"
          className={styles.accountOverviewToolButton}
          onClick={() => void onRefreshAll()}
          disabled={overallLoading}
        >
          <IconRefreshCw
            size={15}
            className={overallLoading ? styles.refreshIconSpinning : styles.refreshIcon}
          />
          <span>{t('common.refresh')}</span>
        </button>
        <div className={styles.accountOverviewSortBar}>
          <Select
            className={styles.accountOverviewSortSelect}
            triggerClassName={styles.accountOverviewSortSelectTrigger}
            value={accountSort.key}
            options={accountSortOptions}
            onChange={(value) => onAccountSortKeyChange(value as AccountSortKey)}
            ariaLabel={t('monitoring.account_overview_sort_label')}
            fullWidth={false}
          />
        </div>

        <div className={`${styles.segmentedControl} ${styles.accountOverviewModeToggle}`}>
          <button
            type="button"
            className={`${styles.segmentButton} ${mode === 'table' ? styles.segmentButtonActive : ''}`}
            onClick={() => onModeChange('table')}
          >
            {t('monitoring.account_overview_view_mode_table')}
          </button>
          <button
            type="button"
            className={`${styles.segmentButton} ${mode === 'card' ? styles.segmentButtonActive : ''}`}
            onClick={() => onModeChange('card')}
          >
            {t('monitoring.account_overview_view_mode_card')}
          </button>
        </div>
      </div>
    </div>
  );
}

export function AccountOverviewPanel({
  embedded = false,
  mode,
  searchInput,
  columns,
  rows,
  pagination,
  accountSort,
  accountSortOptions,
  expandedAccounts,
  focusedAccount,
  accountAuthStateByRowId,
  accountStatusDataByRowId,
  emptyAccountStatusData,
  accountQuotaStates,
  accountStatusUpdating,
  accountPageSize,
  accountPageSizeOptions,
  accountOverviewScopeText,
  hasPrices,
  overallLoading,
  locale,
  emptyState,
  t,
  onSearchChange,
  onRefreshAll,
  onAccountSortKeyChange,
  onModeChange,
  onAccountSort,
  onAccountStatusToggle,
  onLoadAccountQuota,
  onToggleExpanded,
  onFocusAccount,
  onPageChange,
  onPageSizeChange,
}: AccountOverviewPanelProps) {
  const actions = (
    <AccountOverviewPanelActions
      mode={mode}
      searchInput={searchInput}
      accountSort={accountSort}
      accountSortOptions={accountSortOptions}
      overallLoading={overallLoading}
      t={t}
      onSearchChange={onSearchChange}
      onRefreshAll={onRefreshAll}
      onAccountSortKeyChange={onAccountSortKeyChange}
      onModeChange={onModeChange}
    />
  );

  const content = (
    <>
      {mode === 'table' ? (
        <div className={`${styles.tableWrapper} ${styles.accountOverviewTableWrapper}`}>
          <table className={`${styles.table} ${styles.accountOverviewTable}`}>
            <colgroup>
              {columns.map((column) => (
                <col key={column.key} />
              ))}
            </colgroup>
            <thead>
              <tr>
                {columns.map((column) => {
                  const sortKey = column.sortKey;

                  if (!sortKey) {
                    return (
                      <th key={column.key}>
                        <AccountColumnLabel column={column} t={t} />
                      </th>
                    );
                  }

                  const isActive = accountSort.key === sortKey;
                  const SortIcon = isActive
                    ? accountSort.direction === 'desc'
                      ? IconChevronDown
                      : IconChevronUp
                    : null;

                  return (
                    <th
                      key={column.key}
                      aria-sort={
                        isActive
                          ? accountSort.direction === 'desc'
                            ? 'descending'
                            : 'ascending'
                          : 'none'
                      }
                    >
                      <button
                        type="button"
                        className={[
                          styles.sortableHeaderButton,
                          isActive ? styles.sortableHeaderButtonActive : '',
                        ]
                          .filter(Boolean)
                          .join(' ')}
                        onClick={() => onAccountSort(sortKey)}
                      >
                        <AccountColumnLabel column={column} t={t} />
                        <span className={styles.sortIndicator} aria-hidden="true">
                          {SortIcon ? <SortIcon size={14} /> : null}
                        </span>
                      </button>
                    </th>
                  );
                })}
              </tr>
            </thead>
            <tbody>
              {pagination.pageItems.map((row) => {
                const isExpanded = Boolean(expandedAccounts[row.id]);
                const isFocused = focusedAccount === row.account;
                const authState = accountAuthStateByRowId.get(row.id) ?? EMPTY_ACCOUNT_AUTH_STATE;
                const statusTone = getAccountStatusTone(authState);
                const summaryMetrics = buildAccountSummaryMetrics(row, hasPrices, locale, t);
                const metricByKey = new Map(summaryMetrics.map((metric) => [metric.key, metric]));
                const rowClassName = [
                  styles.accountSummaryRow,
                  isFocused ? styles.focusedRow : '',
                  isExpanded ? styles.accountOverviewRowExpanded : '',
                  authState.enabledState === 'disabled' ? styles.accountOverviewRowDisabled : '',
                ]
                  .filter(Boolean)
                  .join(' ');
                const accountMenuItems: DropdownMenuItem[] = [];
                if (authState.enabledState === 'enabled') {
                  accountMenuItems.push({
                    key: 'disable',
                    label: t('monitoring.account_overview_row_menu_disable'),
                    onClick: () => void onAccountStatusToggle(row, false),
                    disabled: accountStatusUpdating[row.id] === true,
                    tone: 'danger',
                  });
                } else if (authState.enabledState === 'disabled') {
                  accountMenuItems.push({
                    key: 'enable',
                    label: t('monitoring.account_overview_row_menu_enable'),
                    onClick: () => void onAccountStatusToggle(row, true),
                    disabled: accountStatusUpdating[row.id] === true,
                  });
                } else if (authState.enabledState === 'mixed') {
                  accountMenuItems.push(
                    {
                      key: 'enable-all',
                      label: t('monitoring.account_overview_row_menu_enable_all'),
                      onClick: () => void onAccountStatusToggle(row, true),
                      disabled: accountStatusUpdating[row.id] === true,
                    },
                    {
                      key: 'disable-all',
                      label: t('monitoring.account_overview_row_menu_disable_all'),
                      onClick: () => void onAccountStatusToggle(row, false),
                      disabled: accountStatusUpdating[row.id] === true,
                      tone: 'danger',
                    }
                  );
                }
                accountMenuItems.push({
                  key: 'refresh-quota',
                  label: t('monitoring.account_overview_row_menu_refresh_quota'),
                  onClick: () => void onLoadAccountQuota(row.account, true),
                });

                return (
                  <Fragment key={row.id}>
                    <tr className={rowClassName || undefined}>
                      <td>
                        <AccountSummaryPrimary
                          row={row}
                          expanded={isExpanded}
                          onToggle={() => onToggleExpanded(row.id, row.account)}
                          statusTone={statusTone}
                        />
                      </td>
                      <td>
                        <AccountStatusBadge authState={authState} t={t} />
                      </td>
                      <td>{metricByKey.get('total-calls')?.value ?? '--'}</td>
                      <td className={metricByKey.get('success-calls')?.valueClassName}>
                        {metricByKey.get('success-calls')?.value ?? '--'}
                      </td>
                      <td className={metricByKey.get('failure-calls')?.valueClassName}>
                        {metricByKey.get('failure-calls')?.value ?? '--'}
                      </td>
                      <td className={getSuccessRateClassName(row.successRate)}>
                        {formatPercent(row.successRate)}
                      </td>
                      <td>{metricByKey.get('total-tokens')?.value ?? '--'}</td>
                      <td>{metricByKey.get('estimated-cost')?.value ?? '--'}</td>
                      <td>{metricByKey.get('latest-request-time')?.value ?? '--'}</td>
                      <td>
                        <div className={styles.accountActionGroup}>
                          <button
                            type="button"
                            className={styles.inlineActionButton}
                            onClick={() => onFocusAccount(row)}
                          >
                            <IconCrosshair size={13} aria-hidden="true" />
                            <span>
                              {isFocused
                                ? t('monitoring.restore_account_scope')
                                : t('monitoring.focus_account')}
                            </span>
                          </button>
                          <DropdownMenu
                            ariaLabel={t('monitoring.account_overview_row_menu_label')}
                            triggerClassName={styles.accountRowMenuButton}
                            triggerIcon={<IconMoreVertical size={15} aria-hidden="true" />}
                            items={accountMenuItems}
                          />
                        </div>
                      </td>
                    </tr>
                    {isExpanded ? (
                      <tr className={styles.accountDetailRow}>
                        <td colSpan={columns.length}>
                          <AccountExpandedDetails
                            row={row}
                            hasPrices={hasPrices}
                            locale={locale}
                            t={t}
                            summaryMetrics={summaryMetrics}
                            quotaState={accountQuotaStates[row.account]}
                            onRefreshQuota={() => void onLoadAccountQuota(row.account, true)}
                            variant="table"
                          />
                        </td>
                      </tr>
                    ) : null}
                  </Fragment>
                );
              })}
              {rows.length === 0 ? (
                <tr>
                  <td colSpan={columns.length}>{emptyState}</td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      ) : rows.length > 0 ? (
        <div className={styles.accountOverviewCardGrid}>
          {pagination.pageItems.map((row) => {
            const authState = accountAuthStateByRowId.get(row.id) ?? EMPTY_ACCOUNT_AUTH_STATE;

            return (
              <AccountOverviewCard
                key={row.id}
                row={row}
                authState={authState}
                hasPrices={hasPrices}
                locale={locale}
                t={t}
                isExpanded={Boolean(expandedAccounts[row.id])}
                isFocused={focusedAccount === row.account}
                statusData={accountStatusDataByRowId.get(row.id) ?? emptyAccountStatusData}
                scopeText={accountOverviewScopeText}
                quotaState={accountQuotaStates[row.account]}
                statusUpdating={accountStatusUpdating[row.id] === true}
                onToggle={() => onToggleExpanded(row.id, row.account)}
                onFocus={() => onFocusAccount(row)}
                onToggleEnabled={(enabled) => void onAccountStatusToggle(row, enabled)}
                onRefreshQuota={() => void onLoadAccountQuota(row.account, true)}
              />
            );
          })}
        </div>
      ) : (
        emptyState
      )}
      <PaginationControls
        count={rows.length}
        currentPage={pagination.currentPage}
        totalPages={pagination.totalPages}
        startItem={pagination.startItem}
        endItem={pagination.endItem}
        pageSize={accountPageSize}
        pageSizeOptions={accountPageSizeOptions}
        onPageChange={onPageChange}
        onPageSizeChange={onPageSizeChange}
        t={t}
      />
    </>
  );

  if (embedded) {
    return content;
  }

  return (
    <MonitoringPanel
      title={
        <span className={styles.panelTitleWithHint}>
          {t('monitoring.account_overview_title')}
          <span title={t('monitoring.account_overview_description')}>
            <IconInfo
              size={14}
              className={styles.panelTitleHintIcon}
              aria-label={t('monitoring.account_overview_description')}
            />
          </span>
        </span>
      }
      className={styles.accountPanel}
      extra={actions}
    >
      {content}
    </MonitoringPanel>
  );
}
