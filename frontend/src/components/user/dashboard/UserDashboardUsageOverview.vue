<template>
  <div class="card">
    <div class="flex items-center justify-between border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('dashboard.usageOverview') }}</h2>
      <span class="badge badge-gray">{{ t('dashboard.currentUsage') }}</span>
    </div>

    <div class="p-4">
      <div v-if="loading" class="flex items-center justify-center py-8">
        <LoadingSpinner />
      </div>
      <div v-else-if="items.length === 0" class="py-6">
        <EmptyState :title="t('dashboard.noUsageOverview')" :description="t('dashboard.noUsageOverviewHint')" />
      </div>
      <div v-else class="overflow-x-auto">
        <table class="min-w-full text-sm">
          <thead>
            <tr class="border-b border-gray-100 text-xs text-gray-500 dark:border-dark-700 dark:text-gray-400">
              <th class="pb-2 text-left font-medium">{{ t('usageOverview.user') }}</th>
              <th class="pb-2 text-right font-medium">{{ t('dashboard.today') }}</th>
              <th class="pb-2 text-right font-medium">{{ t('dashboard.week') }}</th>
              <th class="pb-2 text-right font-medium">{{ t('dashboard.month') }}</th>
              <th class="pb-2 text-right font-medium">{{ t('common.total') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="item in items" :key="item.user_id" class="align-top">
              <td class="max-w-[160px] py-3 pr-4">
                <p class="truncate font-medium text-gray-900 dark:text-white">{{ getUserLabel(item) }}</p>
                <p v-if="getUserSubLabel(item)" class="truncate text-xs text-gray-500 dark:text-gray-400">
                  {{ getUserSubLabel(item) }}
                </p>
              </td>
              <td class="py-3 text-right">
                <UsageAmount :cost="item.today_cost" :requests="item.today_requests" />
              </td>
              <td class="py-3 text-right">
                <UsageAmount :cost="item.week_cost" :requests="item.week_requests" />
              </td>
              <td class="py-3 text-right">
                <UsageAmount :cost="item.month_cost" :requests="item.month_requests" />
              </td>
              <td class="py-3 text-right">
                <UsageAmount :cost="item.total_cost" :requests="item.total_requests" />
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { defineComponent, h } from 'vue'
import { useI18n } from 'vue-i18n'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import type { DashboardUsageOverviewUserItem } from '@/api/usage'

defineProps<{
  items: DashboardUsageOverviewUserItem[]
  loading: boolean
}>()

const { t } = useI18n()

const UsageAmount = defineComponent({
  props: {
    cost: { type: Number, required: true },
    requests: { type: Number, required: true }
  },
  setup(props) {
    return () =>
      h('div', { class: 'space-y-0.5 tabular-nums' }, [
        h('p', { class: 'font-semibold text-gray-900 dark:text-white' }, `$${formatCost(props.cost)}`),
        h('p', { class: 'text-xs text-gray-500 dark:text-gray-400' }, t('dashboard.requestCount', { count: formatNumber(props.requests) }))
      ])
  }
})

function getUserLabel(item: DashboardUsageOverviewUserItem): string {
  return item.username || item.email || `#${item.user_id}`
}

function getUserSubLabel(item: DashboardUsageOverviewUserItem): string {
  if (item.username && item.email && item.username !== item.email) return item.email
  return ''
}

function formatCost(value: number): string {
  return (value || 0).toFixed(4)
}

function formatNumber(value: number): string {
  return Math.round(value || 0).toLocaleString()
}
</script>
