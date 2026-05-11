<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <div class="space-y-4">
          <div
            class="rounded-lg border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-800 dark:border-blue-800/50 dark:bg-blue-900/20 dark:text-blue-200"
          >
            {{ isAdmin ? t('usageOverview.adminHint') : t('usageOverview.userPrivacyHint') }}
          </div>

          <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <div v-for="card in summaryCards" :key="card.key" class="card p-4">
              <p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                {{ card.label }}
              </p>
              <p class="mt-2 text-2xl font-bold text-gray-900 dark:text-white">
                {{ card.value }}
              </p>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ card.description }}
              </p>
            </div>
          </div>
        </div>
      </template>

      <template #filters>
        <div class="card px-6 py-4">
          <div class="flex flex-wrap items-end gap-4">
            <div>
              <label class="input-label">{{ t('usageOverview.startDate') }}</label>
              <input v-model="startDate" type="date" class="input" />
            </div>
            <div>
              <label class="input-label">{{ t('usageOverview.endDate') }}</label>
              <input v-model="endDate" type="date" class="input" />
            </div>
            <div class="flex items-center gap-3">
              <button class="btn btn-primary" :disabled="loading" @click="applyFilters">
                {{ t('usageOverview.apply') }}
              </button>
              <button class="btn btn-secondary" :disabled="loading" @click="resetFilters">
                {{ t('common.reset') }}
              </button>
            </div>
            <div class="ml-auto flex rounded-lg border border-gray-200 p-1 dark:border-dark-700">
              <button
                type="button"
                class="rounded-md px-4 py-2 text-sm font-medium transition-colors"
                :class="activeTab === 'users' ? activeTabClass : inactiveTabClass"
                @click="switchTab('users')"
              >
                {{ t('usageOverview.usersTab') }}
              </button>
              <button
                type="button"
                class="rounded-md px-4 py-2 text-sm font-medium transition-colors"
                :class="activeTab === 'accounts' ? activeTabClass : inactiveTabClass"
                @click="switchTab('accounts')"
              >
                {{ t('usageOverview.accountsTab') }}
              </button>
            </div>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="activeColumns"
          :data="activeItems"
          :loading="loading"
        >
          <template #cell-identity="{ row }">
            <div class="min-w-0">
              <p class="font-medium text-gray-900 dark:text-white">
                {{ getIdentity(row) }}
              </p>
              <p v-if="getSecondaryIdentity(row)" class="text-xs text-gray-500 dark:text-gray-400">
                {{ getSecondaryIdentity(row) }}
              </p>
            </div>
          </template>

          <template #cell-platform="{ row }">
            <span class="text-sm text-gray-700 dark:text-gray-300">
              {{ row.platform || '-' }}
            </span>
          </template>

          <template #cell-status="{ row }">
            <span
              class="inline-flex rounded-full px-2 py-0.5 text-xs font-medium"
              :class="getStatusClass(row.status)"
            >
              {{ row.status || '-' }}
            </span>
          </template>

          <template #cell-requests="{ row }">
            {{ formatNumber(row.total_requests) }}
          </template>

          <template #cell-tokens="{ row }">
            <div class="space-y-1">
              <p class="font-medium text-gray-900 dark:text-white">
                {{ formatTokens(row.total_tokens) }}
              </p>
              <p class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('usageOverview.inputShort') }} {{ formatTokens(row.input_tokens) }} /
                {{ t('usageOverview.outputShort') }} {{ formatTokens(row.output_tokens) }} /
                {{ t('usageOverview.cacheShort') }} {{ formatTokens(row.cache_tokens) }}
              </p>
            </div>
          </template>

          <template #cell-cost="{ row }">
            <div class="space-y-1">
              <p class="font-medium text-green-600 dark:text-green-400">
                {{ formatCurrency(getPrimaryCost(row)) }}
              </p>
              <p class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('usageOverview.today') }} {{ formatCurrency(row.today_cost) }}
              </p>
            </div>
          </template>

          <template #cell-last_used_at="{ row }">
            {{ formatDateTime(row.last_used_at) }}
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          :total="activeTotal"
          :page="activePage"
          :page-size="activePageSize"
          show-jump
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import type { Column } from '@/components/common/types'
import { useAuthStore } from '@/stores/auth'
import {
  usageAPI,
  type UsageOverviewAccountItem,
  type UsageOverviewSummary,
  type UsageOverviewUserItem
} from '@/api/usage'

type OverviewTab = 'users' | 'accounts'
type OverviewRow = UsageOverviewUserItem | UsageOverviewAccountItem

const { t } = useI18n()
const authStore = useAuthStore()

const loading = ref(false)
const summary = ref<UsageOverviewSummary | null>(null)
const activeTab = ref<OverviewTab>('users')
const startDate = ref(defaultStartDate())
const endDate = ref(formatDateInput(new Date()))

const userItems = ref<UsageOverviewUserItem[]>([])
const userTotal = ref(0)
const userPage = ref(1)
const userPageSize = ref(20)

const accountItems = ref<UsageOverviewAccountItem[]>([])
const accountTotal = ref(0)
const accountPage = ref(1)
const accountPageSize = ref(20)

const isAdmin = computed(() => authStore.isAdmin)

const activeTabClass = 'bg-primary-600 text-white shadow-sm'
const inactiveTabClass = 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'

const userColumns = computed<Column[]>(() => {
  const columns: Column[] = [
    { key: 'identity', label: t('usageOverview.user') },
    { key: 'requests', label: t('usageOverview.requests') },
    { key: 'tokens', label: t('usageOverview.tokens') },
    { key: 'cost', label: t('usageOverview.cost') }
  ]

  if (isAdmin.value) {
    columns.push({ key: 'last_used_at', label: t('usageOverview.lastUsedAt') })
  }

  return columns
})

const accountColumns = computed<Column[]>(() => {
  const columns: Column[] = [
    { key: 'identity', label: t('usageOverview.account') },
    { key: 'platform', label: t('usageOverview.platform') },
    { key: 'requests', label: t('usageOverview.requests') },
    { key: 'tokens', label: t('usageOverview.tokens') },
    { key: 'cost', label: t('usageOverview.accountCost') }
  ]

  if (isAdmin.value) {
    columns.splice(2, 0, { key: 'status', label: t('usageOverview.status') })
    columns.push({ key: 'last_used_at', label: t('usageOverview.lastUsedAt') })
  }

  return columns
})

const activeColumns = computed(() => (activeTab.value === 'users' ? userColumns.value : accountColumns.value))
const activeItems = computed<OverviewRow[]>(() =>
  activeTab.value === 'users' ? userItems.value : accountItems.value
)
const activeTotal = computed(() => (activeTab.value === 'users' ? userTotal.value : accountTotal.value))
const activePage = computed(() => (activeTab.value === 'users' ? userPage.value : accountPage.value))
const activePageSize = computed(() =>
  activeTab.value === 'users' ? userPageSize.value : accountPageSize.value
)

const summaryCards = computed(() => [
  {
    key: 'requests',
    label: t('usageOverview.totalRequests'),
    value: formatNumber(summary.value?.total_requests || 0),
    description: t('usageOverview.todayRequests', {
      count: formatNumber(summary.value?.today_requests || 0)
    })
  },
  {
    key: 'tokens',
    label: t('usageOverview.totalTokens'),
    value: formatTokens(summary.value?.total_tokens || 0),
    description: t('usageOverview.tokenBreakdown', {
      input: formatTokens(summary.value?.input_tokens || 0),
      output: formatTokens(summary.value?.output_tokens || 0)
    })
  },
  {
    key: 'users',
    label: t('usageOverview.users'),
    value: `${formatNumber(summary.value?.active_users || 0)} / ${formatNumber(summary.value?.total_users || 0)}`,
    description: t('usageOverview.activeTotal')
  },
  {
    key: 'accounts',
    label: t('usageOverview.accounts'),
    value: `${formatNumber(summary.value?.active_accounts || 0)} / ${formatNumber(summary.value?.total_accounts || 0)}`,
    description: t('usageOverview.activeTotal')
  },
  {
    key: 'standardCost',
    label: t('usageOverview.standardCost'),
    value: formatCurrency(summary.value?.total_cost || 0),
    description: t('usageOverview.todayCost', {
      cost: formatCurrency(summary.value?.today_cost || 0)
    })
  },
  {
    key: 'actualCost',
    label: t('usageOverview.actualCost'),
    value: formatCurrency(summary.value?.total_actual_cost || 0),
    description: t('usageOverview.userBilledCost')
  },
  {
    key: 'accountCost',
    label: t('usageOverview.accountCost'),
    value: formatCurrency(summary.value?.total_account_cost || 0),
    description: t('usageOverview.upstreamAccountCost')
  }
])

function formatDateInput(date: Date): string {
  return date.toISOString().slice(0, 10)
}

function defaultStartDate(): string {
  const date = new Date()
  date.setDate(date.getDate() - 30)
  return formatDateInput(date)
}

function formatNumber(value: number): string {
  return Math.round(value || 0).toLocaleString()
}

function formatTokens(value: number): string {
  const tokens = value || 0
  if (tokens >= 1_000_000_000) return `${(tokens / 1_000_000_000).toFixed(2)}B`
  if (tokens >= 1_000_000) return `${(tokens / 1_000_000).toFixed(2)}M`
  if (tokens >= 1_000) return `${(tokens / 1_000).toFixed(2)}K`
  return formatNumber(tokens)
}

function formatCurrency(value: number): string {
  return `$${(value || 0).toFixed(4)}`
}

function formatDateTime(value?: string): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

function getIdentity(row: OverviewRow): string {
  if ('anonymous_user' in row && row.anonymous_user) return row.anonymous_user
  if ('anonymous_account' in row && row.anonymous_account) return row.anonymous_account
  if ('username' in row && row.username) return row.username
  if ('name' in row && row.name) return row.name
  if ('email' in row && row.email) return row.email
  if ('user_id' in row && row.user_id) return `#${row.user_id}`
  if ('account_id' in row && row.account_id) return `#${row.account_id}`
  return '-'
}

function getSecondaryIdentity(row: OverviewRow): string {
  if ('username' in row && row.email && row.username !== row.email) return row.email
  if ('name' in row && row.email && row.name !== row.email) return row.email
  if ('type' in row && row.type) return row.type
  return ''
}

function getPrimaryCost(row: OverviewRow): number {
  if ('total_account_cost' in row) return row.total_account_cost
  return row.total_actual_cost ?? row.total_cost
}

function getStatusClass(status?: string): string {
  if (status === 'active') return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
  if (status === 'disabled' || status === 'inactive') {
    return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
  }
  if (status === 'error' || status === 'limited') {
    return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  }
  return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
}

function getParams(page: number, pageSize: number) {
  return {
    page,
    page_size: pageSize,
    start_date: startDate.value || undefined,
    end_date: endDate.value || undefined
  }
}

async function loadSummary(): Promise<void> {
  summary.value = await usageAPI.getUsageOverviewSummary({
    start_date: startDate.value || undefined,
    end_date: endDate.value || undefined
  })
}

async function loadUsers(): Promise<void> {
  const response = await usageAPI.getUsageOverviewUsers(getParams(userPage.value, userPageSize.value))
  userItems.value = response.items || []
  userTotal.value = response.total || 0
}

async function loadAccounts(): Promise<void> {
  const response = await usageAPI.getUsageOverviewAccounts(
    getParams(accountPage.value, accountPageSize.value)
  )
  accountItems.value = response.items || []
  accountTotal.value = response.total || 0
}

async function loadData(): Promise<void> {
  loading.value = true
  try {
    await Promise.all([loadSummary(), loadUsers(), loadAccounts()])
  } finally {
    loading.value = false
  }
}

async function applyFilters(): Promise<void> {
  userPage.value = 1
  accountPage.value = 1
  await loadData()
}

async function resetFilters(): Promise<void> {
  startDate.value = defaultStartDate()
  endDate.value = formatDateInput(new Date())
  await applyFilters()
}

function switchTab(tab: OverviewTab): void {
  activeTab.value = tab
}

async function handlePageChange(page: number): Promise<void> {
  if (activeTab.value === 'users') {
    userPage.value = page
    loading.value = true
    try {
      await loadUsers()
    } finally {
      loading.value = false
    }
    return
  }

  accountPage.value = page
  loading.value = true
  try {
    await loadAccounts()
  } finally {
    loading.value = false
  }
}

async function handlePageSizeChange(pageSize: number): Promise<void> {
  if (activeTab.value === 'users') {
    userPageSize.value = pageSize
    userPage.value = 1
  } else {
    accountPageSize.value = pageSize
    accountPage.value = 1
  }
  await handlePageChange(1)
}

onMounted(() => {
  loadData()
})
</script>
