<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <div class="flex items-center justify-between gap-3">
        <div class="min-w-0">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('dashboard.accountUsageWindow') }}</h2>
          <p class="truncate text-xs text-gray-500 dark:text-dark-400">
            {{ accountWindow?.name || t('dashboard.clickLoadUsageWindow') }}
          </p>
        </div>
        <button
          type="button"
          class="btn btn-secondary btn-sm shrink-0"
          :disabled="loadingAccountWindow"
          @click="loadAccountWindow(false)"
        >
          {{ loadingAccountWindow ? t('common.loading') : t(accountWindow ? 'common.refresh' : 'dashboard.loadUsageWindow') }}
        </button>
      </div>
    </div>

    <div class="p-4">
      <div v-if="accountWindowError" class="text-sm text-red-600 dark:text-red-400">
        {{ accountWindowError }}
      </div>
      <div v-else-if="accountWindow" class="space-y-3">
        <div v-for="window in accountWindow.windows" :key="window.label" class="space-y-2">
          <div class="grid grid-cols-4 gap-2 text-center text-sm">
            <div class="rounded-lg bg-gray-50 px-2 py-2 text-gray-600 dark:bg-dark-800 dark:text-gray-300">
              {{ formatRequests(window.requests) }}
            </div>
            <div class="rounded-lg bg-gray-50 px-2 py-2 text-gray-600 dark:bg-dark-800 dark:text-gray-300">
              {{ formatTokens(window.tokens) }}
            </div>
            <div class="rounded-lg bg-gray-50 px-2 py-2 text-gray-600 dark:bg-dark-800 dark:text-gray-300">
              A ${{ formatMoney(window.account_cost) }}
            </div>
            <div class="rounded-lg bg-gray-50 px-2 py-2 text-gray-600 dark:bg-dark-800 dark:text-gray-300">
              U ${{ formatMoney(window.user_cost) }}
            </div>
          </div>
          <div class="flex items-center gap-3">
            <span
              class="w-11 rounded-lg px-2 py-1 text-center text-sm font-semibold"
              :class="window.label === '7d' ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300' : 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'"
            >
              {{ window.label }}
            </span>
            <div class="h-2 min-w-0 flex-1 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
              <div class="h-full rounded-full bg-emerald-500" :style="{ width: progressWidth(window.utilization) }" />
            </div>
            <span class="w-10 text-right text-sm font-semibold tabular-nums text-gray-700 dark:text-gray-200">
              {{ formatPercent(window.utilization) }}
            </span>
            <span class="w-16 text-sm tabular-nums text-gray-400 dark:text-gray-500">
              {{ formatRemaining(window.remaining_seconds) }}
            </span>
          </div>
        </div>
      </div>
      <div v-else class="text-sm text-gray-500 dark:text-gray-400">
        {{ t('dashboard.noUsageWindowLoaded') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { usageAPI, type DashboardAccountUsageWindow } from '@/api/usage'

const { t } = useI18n()

const accountWindow = ref<DashboardAccountUsageWindow | null>(null)
const loadingAccountWindow = ref(false)
const accountWindowError = ref('')

async function loadAccountWindow(cachedOnly = false): Promise<void> {
  loadingAccountWindow.value = true
  accountWindowError.value = ''
  try {
    const data = await usageAPI.getDashboardAccountWindow({ cached_only: cachedOnly })
    accountWindow.value = data
    if (!cachedOnly && (!accountWindow.value || accountWindow.value.windows.length === 0)) {
      accountWindowError.value = t('dashboard.noAccountUsageWindow')
    }
  } catch (error) {
    console.error('Failed to load account usage window:', error)
    if (!cachedOnly) {
      accountWindowError.value = t('dashboard.loadUsageWindowFailed')
    }
  } finally {
    loadingAccountWindow.value = false
  }
}

onMounted(() => {
  loadAccountWindow(false)
})

function formatRequests(value: number): string {
  return `${formatCompact(value)} req`
}

function formatTokens(value: number): string {
  return formatCompact(value)
}

function formatCompact(value: number): string {
  const n = value || 0
  if (n >= 1_000_000_000) return `${(n / 1_000_000_000).toFixed(1)}B`
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return Math.round(n).toLocaleString()
}

function formatMoney(value: number): string {
  return (value || 0).toFixed(2)
}

function formatPercent(value: number): string {
  return `${Math.max(0, Math.round(value || 0))}%`
}

function progressWidth(value: number): string {
  return `${Math.min(Math.max(value || 0, 0), 100)}%`
}

function formatRemaining(seconds: number): string {
  if (!seconds || seconds <= 0) return '-'
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}
</script>
