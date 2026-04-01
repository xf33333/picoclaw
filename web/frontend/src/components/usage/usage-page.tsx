import { IconCalendar, IconChartBar, IconCoin, IconMessage, IconAbc } from "@tabler/icons-react"
import { useQuery } from "@tanstack/react-query"
import dayjs from "dayjs"
import * as React from "react"
import { useTranslation } from "react-i18next"

import { getUsage, type UsageStats } from "@/api/usage"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

// Quick date range presets
const DATE_RANGES = [
  { label: "today", value: "today" },
  { label: "yesterday", value: "yesterday" },
  { label: "last_7_days", value: "last_7_days" },
  { label: "last_30_days", value: "last_30_days" },
  { label: "this_month", value: "this_month" },
  { label: "last_month", value: "last_month" },
]

function getDateRange(value: string): { start: string; end: string } {
  const now = dayjs()
  let start: dayjs.Dayjs
  let end: dayjs.Dayjs

  switch (value) {
    case "today":
      start = now.startOf("day")
      end = now.endOf("day")
      break
    case "yesterday":
      start = now.subtract(1, "day").startOf("day")
      end = now.subtract(1, "day").endOf("day")
      break
    case "last_7_days":
      start = now.subtract(6, "day").startOf("day")
      end = now.endOf("day")
      break
    case "last_30_days":
      start = now.subtract(29, "day").startOf("day")
      end = now.endOf("day")
      break
    case "this_month":
      start = now.startOf("month")
      end = now.endOf("month")
      break
    case "last_month":
      start = now.subtract(1, "month").startOf("month")
      end = now.subtract(1, "month").endOf("month")
      break
    default:
      start = now.startOf("day")
      end = now.endOf("day")
  }

  return {
    start: start.format("YYYY-MM-DD"),
    end: end.format("YYYY-MM-DD"),
  }
}

function formatNumber(n: number): string {
  if (n >= 1_000_000) {
    return `${(n / 1_000_000).toFixed(2)}M`
  }
  if (n >= 1_000) {
    return `${(n / 1_000).toFixed(1)}K`
  }
  return n.toString()
}

function formatCost(n: number, _currency: string): string {
  return `$${n.toFixed(4)}`
}

export function UsagePage() {
  const { t } = useTranslation()
  const [dateRange, setDateRange] = React.useState("today")

  const { start, end } = React.useMemo(() => getDateRange(dateRange), [dateRange])

  const { data, isLoading, error } = useQuery({
    queryKey: ["usage", start, end],
    queryFn: () => getUsage(start, end),
  })

  const stats = data?.stats ?? []
  const totalInputTokens = data?.total_input_tokens ?? 0
  const totalOutputTokens = data?.total_output_tokens ?? 0
  const totalMessageCount = data?.total_message_count ?? 0
  const totalEstimatedCost = data?.total_estimated_cost ?? 0
  const currency = data?.currency ?? "USD"

  return (
    <div className="flex flex-col gap-6 p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("usage.title")}</h1>
          <p className="text-muted-foreground mt-1">
            {t("usage.description")}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <IconCalendar className="size-4 text-muted-foreground" />
          <Select value={dateRange} onValueChange={setDateRange}>
            <SelectTrigger className="w-[180px]">
              <SelectValue placeholder={t("usage.select_date_range")} />
            </SelectTrigger>
            <SelectContent>
              {DATE_RANGES.map((range) => (
                <SelectItem key={range.value} value={range.value}>
                  {t(`usage.date_range.${range.label}`)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              {t("usage.total_messages")}
            </CardTitle>
            <IconMessage className="size-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{formatNumber(totalMessageCount)}</div>
            <p className="text-muted-foreground text-xs">
              {t("usage.messages_in_period")}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              {t("usage.input_tokens")}
            </CardTitle>
            <IconAbc className="size-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{formatNumber(totalInputTokens)}</div>
            <p className="text-muted-foreground text-xs">
              {t("usage.tokens_in_period")}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              {t("usage.output_tokens")}
            </CardTitle>
            <IconAbc className="size-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{formatNumber(totalOutputTokens)}</div>
            <p className="text-muted-foreground text-xs">
              {t("usage.tokens_in_period")}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              {t("usage.estimated_cost")}
            </CardTitle>
            <IconCoin className="size-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{formatCost(totalEstimatedCost, currency)}</div>
            <p className="text-muted-foreground text-xs">
              {t("usage.cost_estimate")}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Model Stats Table */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <IconChartBar className="size-5" />
            {t("usage.model_breakdown")}
          </CardTitle>
          <CardDescription>
            {t("usage.model_breakdown_desc")}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-8 text-muted-foreground">
              {t("labels.loading")}
            </div>
          ) : error ? (
            <div className="flex items-center justify-center py-8 text-destructive">
              {t("usage.load_error")}
            </div>
          ) : stats.length === 0 ? (
            <div className="flex items-center justify-center py-8 text-muted-foreground">
              {t("usage.no_data")}
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b">
                    <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground">
                      {t("usage.model_name")}
                    </th>
                    <th className="text-right py-3 px-4 text-sm font-medium text-muted-foreground">
                      {t("usage.messages")}
                    </th>
                    <th className="text-right py-3 px-4 text-sm font-medium text-muted-foreground">
                      {t("usage.input")}
                    </th>
                    <th className="text-right py-3 px-4 text-sm font-medium text-muted-foreground">
                      {t("usage.output")}
                    </th>
                    <th className="text-right py-3 px-4 text-sm font-medium text-muted-foreground">
                      {t("usage.total")}
                    </th>
                    <th className="text-right py-3 px-4 text-sm font-medium text-muted-foreground">
                      {t("usage.cost")}
                    </th>
                    <th className="text-right py-3 px-4 text-sm font-medium text-muted-foreground">
                      {t("usage.sessions")}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {stats.map((stat: UsageStats, index: number) => (
                    <tr
                      key={stat.model_name}
                      className={`border-b transition-colors hover:bg-muted/50 ${
                        index % 2 === 0 ? "bg-background" : "bg-muted/20"
                      }`}
                    >
                      <td className="py-3 px-4 text-sm font-medium">
                        <div>
                          <div className="flex items-center gap-2">
                            <span>{stat.model_name}</span>
                            {stat.provider && (
                              <span className="inline-flex items-center rounded-full bg-muted px-2 py-0.5 text-[10px] font-normal text-muted-foreground uppercase tracking-wider">
                                {stat.provider}
                              </span>
                            )}
                          </div>
                          {stat.model !== stat.model_name && (
                            <div className="text-muted-foreground text-xs font-normal">
                              {stat.model}
                            </div>
                          )}
                        </div>
                      </td>
                      <td className="py-3 px-4 text-sm text-right">
                        {formatNumber(stat.message_count)}
                      </td>
                      <td className="py-3 px-4 text-sm text-right">
                        {formatNumber(stat.input_tokens)}
                      </td>
                      <td className="py-3 px-4 text-sm text-right">
                        {formatNumber(stat.output_tokens)}
                      </td>
                      <td className="py-3 px-4 text-sm text-right font-medium">
                        {formatNumber(stat.total_tokens)}
                      </td>
                      <td className="py-3 px-4 text-sm text-right">
                        {formatCost(stat.estimated_cost, stat.currency)}
                      </td>
                      <td className="py-3 px-4 text-sm text-right">
                        {stat.session_count}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
